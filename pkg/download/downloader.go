package download

import (
	"errors"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/virtuald/go-paniclog"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// task info bucket
	bucketTask = "task"
	// task download data bucket
	bucketSave = "save"
	// downloader config bucket
	bucketConfig = "config"
	// downloader extension bucket
	bucketExtension = "extension"
)

type Listener func(event *Event)

type Progress struct {
	// 下载耗时(纳秒)
	Used int64 `json:"used"`
	// 每秒下载字节数
	Speed int64 `json:"speed"`
	// 已下载的字节数
	Downloaded int64 `json:"downloaded"`
}

type fetcherResolved struct {
	req     *base.Request
	fetcher fetcher.Fetcher
	rr      *ResolveResult
}

type Downloader struct {
	cfg           *DownloaderConfig
	fetchBuilders map[string]fetcher.FetcherBuilder
	resolvedMap   map[string]*fetcherResolved
	storage       Storage
	tasks         []*Task
	listener      Listener

	lock   *sync.Mutex
	closed atomic.Bool

	extensions []*Extension
}

func NewDownloader(cfg *DownloaderConfig) *Downloader {
	if cfg == nil {
		cfg = &DownloaderConfig{}
	}
	cfg.Init()

	d := &Downloader{
		cfg: cfg,

		fetchBuilders: make(map[string]fetcher.FetcherBuilder),
		resolvedMap:   make(map[string]*fetcherResolved),

		storage: cfg.Storage,
		lock:    &sync.Mutex{},

		extensions: make([]*Extension, 0),
	}
	for _, f := range cfg.FetchBuilders {
		for _, p := range f.Schemes() {
			d.fetchBuilders[strings.ToUpper(p)] = f
		}
	}

	if cfg.ProductionMode {
		logPanic(filepath.Join(cfg.StorageDir, "logs"))
	}
	return d
}

func (d *Downloader) Setup() error {
	// setup storage
	if err := d.storage.Setup([]string{bucketTask, bucketSave, bucketConfig, bucketExtension}); err != nil {
		return err
	}
	// load config from storage
	var cfg DownloaderStoreConfig
	exist, err := d.storage.Get(bucketConfig, "config", &cfg)
	if err != nil {
		return err
	}
	if exist {
		d.cfg.DownloaderStoreConfig = &cfg
	} else {
		d.cfg.DownloaderStoreConfig = &DownloaderStoreConfig{
			FirstLoad: true,
		}
	}
	// init default config
	d.cfg.DownloaderStoreConfig.Init()
	// load tasks from storage
	var tasks []*Task
	if err = d.storage.List(bucketTask, &tasks); err != nil {
		return err
	}
	if tasks == nil {
		tasks = make([]*Task, 0)
	} else {
		for i := len(tasks) - 1; i >= 0; i-- {
			task := tasks[i]
			// Remove broken tasks
			if task.Meta == nil {
				tasks = append(tasks[:i], tasks[i+1:]...)
				continue
			}
			initTask(task)
			if task.Status != base.DownloadStatusDone && task.Status != base.DownloadStatusError {
				task.Status = base.DownloadStatusPause
			}
		}
	}
	d.tasks = tasks

	// 每个tick统计一次下载速度
	go func() {

		for !d.closed.Load() {
			if len(d.tasks) > 0 {
				for _, task := range d.tasks {
					func() {
						task.lock.Lock()
						defer task.lock.Unlock()
						if task.Status != base.DownloadStatusRunning {
							return
						}
						// check if task is deleted
						if d.GetTask(task.ID) == nil {
							return
						}

						current := task.fetcher.Progress().TotalDownloaded()
						task.Progress.Used = task.timer.Used()
						task.Progress.Speed = task.calcSpeed(current-task.Progress.Downloaded, float64(d.cfg.RefreshInterval)/1000)
						task.Progress.Downloaded = current
						d.emit(EventKeyProgress, task)

						// store fetcher progress
						data, err := task.fetcherBuilder.Store(task.fetcher)
						if err != nil {
							return // TODO log
						}
						if err := d.storage.Put(bucketSave, task.ID, data); err != nil {
							return // TODO log
						}
						if err := d.storage.Put(bucketTask, task.ID, task); err != nil {
							return // TODO log
						}
					}()
				}
			}
			time.Sleep(time.Millisecond * time.Duration(d.cfg.RefreshInterval))
		}
	}()
	return nil
}

func (d *Downloader) parseFb(url string) (fetcher.FetcherBuilder, error) {
	schema := util.ParseSchema(url)
	fetchBuilder, ok := d.fetchBuilders[schema]
	if !ok {
		fetchBuilder, ok = d.fetchBuilders[util.FileSchema]
	}
	if ok {
		return fetchBuilder, nil
	}
	return nil, errors.New("unsupported protocol")
}

func (d *Downloader) setupFetcher(fetcher fetcher.Fetcher) {
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) (bool, error) {
		return d.getProtocolConfig(fetcher.Name(), v)
	}
	fetcher.Setup(ctl)
}

func (d *Downloader) Resolve(req *base.Request) (rr *ResolveResult, err error) {
	rrId, err := gonanoid.New()
	if err != nil {
		return
	}

	fr := &fetcherResolved{
		req: req,
		rr:  rr,
	}

	res := d.triggerOnResolve(req)
	if res != nil && len(res.Files) > 0 {
		rr = &ResolveResult{
			ID:  rrId,
			Res: res,
		}
		d.resolvedMap[rrId] = fr
		return
	}

	fetcher, err := d.buildFetcher(req.URL)
	if err != nil {
		return
	}
	err = fetcher.Resolve(req)
	if err != nil {
		return
	}
	rr = &ResolveResult{
		ID:  rrId,
		Res: fetcher.Meta().Res,
	}
	fr.fetcher = fetcher
	d.resolvedMap[rrId] = fr
	return
}

func (d *Downloader) CreateDirect(req *base.Request, opts *base.Options) (taskId string, err error) {
	var fetcher fetcher.Fetcher
	fetcher, err = d.buildFetcher(req.URL)
	if err != nil {
		return
	}
	fetcher.Meta().Req = req
	return d.doCreate(fetcher, opts)
}

func (d *Downloader) CreateDirectBatch(optReqs []*base.Request, opts *base.Options) (taskIds []string, err error) {
	for _, optReq := range optReqs {
		taskId, err := d.CreateDirect(optReq, opts.Clone())
		if err != nil {
			return nil, err
		}
		taskIds = append(taskIds, taskId)
	}
	return
}

func (d *Downloader) Create(rrId string, opts *base.Options) (taskId string, err error) {
	if opts == nil {
		opts = &base.Options{}
	}
	fr, ok := d.resolvedMap[rrId]
	if !ok {
		return "", errors.New("invalid resource id")
	}
	defer delete(d.resolvedMap, rrId)

	if fr.fetcher != nil {
		// normal task
		return d.doCreate(fr.fetcher, opts)
	} else {
		// resource is changed by the extension
		res := fr.rr.Res
		if len(res.Files) == 0 {
			return "", errors.New("invalid resource files")
		}
		// create task for selected file
		opts.InitSelectFiles(len(res.Files))
		for index := range opts.SelectFiles {
			file := res.Files[index]
			if file.Req != nil {
				fb, err := d.parseFb(file.Req.URL)
				if err != nil {
					return "", err
				}
				fetcher := fb.Build()
				fetcher.Resolve(file.Req)
				if _, err = d.doCreate(fetcher, opts.Clone()); err != nil {
					return "", err
				}
			}
		}
	}
	return
}

func (d *Downloader) Pause(id string) (err error) {
	task := d.GetTask(id)
	if task == nil {
		return
	}
	if err = d.doPause(task, base.DownloadStatusPause); err != nil {
		return err
	}
	return d.notifyRunning()
}

func (d *Downloader) Continue(id string) (err error) {
	task := d.GetTask(id)
	if task == nil {
		return
	}
	d.tryRunning(func(remain int) {
		if remain == 0 {
			// no more task can be running, need to doPause a running task
			for _, t := range d.tasks {
				if t.Status == base.DownloadStatusRunning {
					err = d.doPause(t, base.DownloadStatusWait)
					break
				}
			}
		}
		if err != nil {
			return
		}
	})
	if err != nil {
		return
	}
	return d.doContinue(task)
}

func (d *Downloader) PauseAll() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.doPauseAll()
}

func (d *Downloader) ContinueAll() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	runningCount := 0
	continueTasks := make([]*Task, 0)
	for _, task := range d.tasks {
		if task.Status == base.DownloadStatusRunning {
			runningCount++
		} else if task.Status != base.DownloadStatusDone {
			continueTasks = append(continueTasks, task)
		}
	}
	// calculate how many tasks can be continued, can't exceed maxRunning
	continueCount := int(math.Min(float64(d.cfg.MaxRunning-runningCount), float64(len(continueTasks))))
	for i := 0; i < continueCount; i++ {
		if err = d.doContinue(continueTasks[i]); err != nil {
			return
		}
	}
	return
}

func (d *Downloader) Delete(id string, force bool) (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	task := d.GetTask(id)
	if task == nil {
		return
	}
	task.lock.Lock()
	defer task.lock.Unlock()
	if task.fetcher != nil {
		err = task.fetcher.Close()
		if err != nil {
			return
		}
	}
	for i, t := range d.tasks {
		if t.ID == id {
			d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
			break
		}
	}
	err = d.storage.Delete(bucketTask, id)
	if err != nil {
		return
	}
	err = d.storage.Delete(bucketSave, id)
	if err != nil {
		return
	}
	if force && task.Meta.Res != nil {
		if task.Meta.Res.RootDir != "" {
			if err = os.RemoveAll(path.Join(task.Meta.Opts.Path, task.Meta.Res.RootDir)); err != nil {
				return
			}
		} else {
			for _, file := range task.Meta.Res.Files {
				if err = util.SafeRemove(task.Meta.Filepath(file)); err != nil {
					return err
				}
			}
		}
	}
	d.emit(EventKeyDelete, task)
	if task.Status == base.DownloadStatusRunning {
		if err = d.doNotifyRunning(); err != nil {
			return err
		}
	}
	task = nil
	return
}

func (d *Downloader) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.closed.Store(true)
	if err := d.doPauseAll(); err != nil {
		return err
	}
	return d.storage.Close()
}

func (d *Downloader) Clear() error {
	if !d.closed.Load() {
		if err := d.Close(); err != nil {
			return err
		}
	}
	if err := d.storage.Clear(); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) Listener(fn Listener) {
	d.listener = fn
}

func (d *Downloader) emit(eventKey EventKey, task *Task, errs ...error) {
	if d.listener != nil {
		var err error
		if len(errs) > 0 {
			err = errs[0]
		}
		d.listener(&Event{
			Key:  eventKey,
			Task: task,
			Err:  err,
		})
	}
}

func (d *Downloader) GetTask(id string) *Task {
	for _, task := range d.tasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (d *Downloader) GetTasks() []*Task {
	// reverse tasks, so that the latest task is in the front
	tasks := make([]*Task, len(d.tasks))
	for i := 0; i < len(d.tasks); i++ {
		tasks[i] = d.tasks[len(d.tasks)-i-1]
	}
	return tasks
}

func (d *Downloader) GetConfig() (*DownloaderStoreConfig, error) {
	return d.cfg.DownloaderStoreConfig, nil
}

func (d *Downloader) PutConfig(v *DownloaderStoreConfig) error {
	d.cfg.DownloaderStoreConfig = v
	return d.storage.Put(bucketConfig, "config", v)
}

func (d *Downloader) getProtocolConfig(name string, v any) (bool, error) {
	cfg, err := d.GetConfig()
	if err != nil {
		return false, err
	}
	if cfg.ProtocolConfig == nil || cfg.ProtocolConfig[name] == nil {
		return false, nil
	}
	if err := util.MapToStruct(cfg.ProtocolConfig[name], v); err != nil {
		return false, err
	}
	return true, nil
}

// wait task done
func (d *Downloader) watch(task *Task) {
	err := task.fetcher.Wait()
	if err != nil {
		task.Status = base.DownloadStatusError
		d.emit(EventKeyError, task, err)
	} else {
		task.Progress.Used = task.timer.Used()
		if task.Size == 0 {
			task.Size = task.fetcher.Progress().TotalDownloaded()
			task.Meta.Res.Size = task.Size
		}
		used := task.Progress.Used / int64(time.Second)
		if used == 0 {
			used = 1
		}
		task.Progress.Speed = task.Size / used
		task.Progress.Downloaded = task.Size
		task.Status = base.DownloadStatusDone
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.emit(EventKeyDone, task)
	}
	d.emit(EventKeyFinally, task, err)
	d.notifyRunning()
}

func (d *Downloader) restoreFetcher(task *Task) error {
	if task.fetcher == nil {
		fb, err := d.parseFb(task.Meta.Req.URL)
		if err != nil {
			return err
		}
		task.fetcherBuilder = fb
		err = func() error {
			v, f := fb.Restore()
			if v != nil {
				err := d.storage.Pop(bucketSave, task.ID, v)
				if err != nil {
					return err
				}
			}
			task.fetcher = f(task.Meta, v)
			return nil
		}()
		if err != nil {
			// TODO log
		}
		if task.fetcher == nil {
			task.fetcher = fb.Build()
		}
		d.setupFetcher(task.fetcher)
		go d.watch(task)
	}
	task.fetcher.Create(task.Meta.Opts)
	return nil
}

func (d *Downloader) doCreate(fetcher fetcher.Fetcher, opts *base.Options) (taskId string, err error) {
	if opts == nil {
		opts = &base.Options{}
	}

	meta := fetcher.Meta()
	meta.Opts = opts
	if opts.Path == "" {
		storeConfig, err := d.GetConfig()
		if err != nil {
			return "", err
		}
		opts.Path = storeConfig.DownloadDir
	}

	fb, err := d.parseFb(fetcher.Meta().Req.URL)
	if err != nil {
		return
	}

	task := NewTask()
	task.fetcherBuilder = fb
	task.fetcher = fetcher
	task.Meta = fetcher.Meta()
	task.Progress = &Progress{}
	task.Status = base.DownloadStatusReady
	initTask(task)
	err = func() error {
		d.lock.Lock()
		defer d.lock.Unlock()
		d.tasks = append(d.tasks, task)
		return d.storage.Put(bucketTask, task.ID, task.clone())
	}()
	if err != nil {
		return
	}
	if err = fetcher.Create(opts); err != nil {
		return
	}
	taskId = task.ID

	go d.watch(task)
	go func() {
		d.tryRunning(func(remain int) {
			if remain > 0 {
				if err := d.start(task); err != nil {
					// TODO log
				}
			}
		})
	}()
	return
}

func (d *Downloader) tryRunning(cb func(remain int)) {
	d.lock.Lock()
	defer d.lock.Unlock()

	runningCount := 0
	for _, task := range d.tasks {
		if task.Status == base.DownloadStatusRunning {
			runningCount++
		}
	}
	cb(d.cfg.MaxRunning - runningCount)
}

func (d *Downloader) notifyRunning() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	return d.doNotifyRunning()
}

func (d *Downloader) doNotifyRunning() error {
	for _, task := range d.tasks {
		if task.Status == base.DownloadStatusReady || task.Status == base.DownloadStatusWait {
			return d.doContinue(task)
		}
	}
	return nil
}

func (d *Downloader) doPauseAll() (err error) {
	for _, task := range d.tasks {
		if err = d.doPause(task, base.DownloadStatusPause); err != nil {
			return
		}
	}
	return
}

func (d *Downloader) start(task *Task) error {
	task.lock.Lock()
	defer task.lock.Unlock()
	return d.doStart(task)
}

func (d *Downloader) doStart(task *Task) error {
	task.Status = base.DownloadStatusRunning
	task.Progress.Speed = 0
	task.timer.Start()
	if err := d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
		return err
	}
	if err := task.fetcher.Start(); err != nil {
		return err
	}
	d.emit(EventKeyStart, task)
	return nil
}

func (d *Downloader) doPause(task *Task, status base.Status) (err error) {
	task.lock.Lock()
	defer task.lock.Unlock()
	if task.Status != base.DownloadStatusDone {
		task.Status = status
		task.timer.Pause()
		task.fetcher.Pause()
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.emit(EventKeyPause, task)
	}
	return
}

func (d *Downloader) doContinue(task *Task) (err error) {
	task.lock.Lock()
	defer task.lock.Unlock()
	if task.Status != base.DownloadStatusRunning && task.Status != base.DownloadStatusDone {
		err = d.restoreFetcher(task)
		if err != nil {
			return
		}
		err = d.doStart(task)
	}
	return
}

// redirect stderr to log file, when panic happened log it
func logPanic(logDir string) {
	if err := util.CreateDirIfNotExist(logDir); err != nil {
		return
	}
	f, err := os.Create(filepath.Join(logDir, "crash.log"))
	if err != nil {
		return
	}
	paniclog.RedirectStderr(f)
}

func (d *Downloader) buildFetcher(url string) (fetcher.Fetcher, error) {
	fb, err := d.parseFb(url)
	if err != nil {
		return nil, err
	}
	fetcher := fb.Build()
	d.setupFetcher(fetcher)
	return fetcher, nil
}

func initTask(task *Task) {
	task.timer = util.NewTimer(task.Progress.Used)
	task.lock = &sync.Mutex{}
	task.speedArr = make([]int64, 0)
}

var defaultDownloader = NewDownloader(nil)

type boot struct {
	url      string
	extra    interface{}
	listener Listener
}

func (b *boot) URL(url string) *boot {
	b.url = url
	return b
}

func (b *boot) Extra(extra interface{}) *boot {
	b.extra = extra
	return b
}

func (b *boot) Resolve() (*ResolveResult, error) {
	return defaultDownloader.Resolve(&base.Request{
		URL:   b.url,
		Extra: b.extra,
	})
}

func (b *boot) Listener(listener Listener) *boot {
	b.listener = listener
	return b
}

func (b *boot) Create(opts *base.Options) (string, error) {
	rr, err := b.Resolve()
	if err != nil {
		return "", err
	}
	defaultDownloader.Listener(b.listener)
	return defaultDownloader.Create(rr.ID, opts)
}

func Boot() *boot {
	err := defaultDownloader.Setup()
	if err != nil {
		panic(err)
	}
	return &boot{}
}
