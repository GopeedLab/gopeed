package download

import (
	"errors"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"os"
	"path"
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

type Downloader struct {
	fetchBuilders map[string]fetcher.FetcherBuilder
	fetcherMap    map[string]fetcher.Fetcher
	storage       Storage
	tasks         []*Task
	listener      Listener

	refreshInterval int
	lock            *sync.Mutex
	closed          atomic.Bool
}

func NewDownloader(cfg *DownloaderConfig) *Downloader {
	if cfg == nil {
		cfg = &DownloaderConfig{}
	}
	cfg.Init()

	d := &Downloader{
		fetchBuilders: make(map[string]fetcher.FetcherBuilder),
		fetcherMap:    make(map[string]fetcher.Fetcher),

		refreshInterval: cfg.RefreshInterval,
		storage:         cfg.Storage,
		lock:            &sync.Mutex{},
	}
	for _, f := range cfg.FetchBuilders {
		for _, p := range f.Schemes() {
			d.fetchBuilders[strings.ToUpper(p)] = f
		}
	}
	return d
}

func (d *Downloader) Setup() error {
	// setup storage
	if err := d.storage.Setup([]string{bucketTask, bucketSave, bucketConfig}); err != nil {
		return err
	}
	// load tasks from storage
	var tasks []*Task
	if err := d.storage.List(bucketTask, &tasks); err != nil {
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
						if task.Status == base.DownloadStatusDone ||
							task.Status == base.DownloadStatusError ||
							task.Status == base.DownloadStatusPause {
							return
						}
						// check if task is deleted
						if d.GetTask(task.ID) == nil {
							return
						}

						current := task.fetcher.Progress().TotalDownloaded()
						task.Progress.Used = task.timer.Used()
						task.Progress.Speed = task.calcSpeed(current-task.Progress.Downloaded, float64(d.refreshInterval)/1000)
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
			time.Sleep(time.Millisecond * time.Duration(d.refreshInterval))
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
	fb, err := d.parseFb(req.URL)
	if err != nil {
		return
	}
	fetcher := fb.Build()
	d.setupFetcher(fetcher)
	err = fetcher.Resolve(req)
	if err != nil {
		return
	}
	rrId, err := gonanoid.New()
	if err != nil {
		return
	}
	d.fetcherMap[rrId] = fetcher
	rr = &ResolveResult{
		ID:  rrId,
		Res: fetcher.Meta().Res,
	}
	return
}

func (d *Downloader) DirectCreate(req *base.Request, opts *base.Options) (taskId string, err error) {
	rr, err := d.Resolve(req)
	if err != nil {
		return
	}
	return d.Create(rr.ID, opts)
}

func (d *Downloader) Create(rrId string, opts *base.Options) (taskId string, err error) {
	if opts == nil {
		opts = &base.Options{}
	}
	fetcher, ok := d.fetcherMap[rrId]
	if !ok {
		return "", errors.New("invalid resource id")
	}
	meta := fetcher.Meta()
	meta.Opts = opts
	res := meta.Res
	if len(opts.SelectFiles) == 0 {
		opts.SelectFiles = make([]int, 0)
		for i := range res.Files {
			opts.SelectFiles = append(opts.SelectFiles, i)
		}
	}
	if opts.Path == "" {
		exist, storeConfig, err := d.GetConfig()
		if err != nil {
			return "", err
		}
		if exist {
			opts.Path = storeConfig.DownloadDir
		}
	}

	fb, err := d.parseFb(fetcher.Meta().Req.URL)
	if err != nil {
		return
	}

	// check if the download file is duplicated and rename it automatically.
	files := res.Files
	if res.RootDir != "" {
		fullDirPath := path.Join(opts.Path, res.RootDir)
		newName, err := util.CheckDuplicateAndRename(fullDirPath)
		if err != nil {
			return "", err
		}
		res.RootDir = newName
	} else if len(files) == 1 {
		fullPath := meta.Filepath(files[0])
		newName, err := util.CheckDuplicateAndRename(fullPath)
		if err != nil {
			return "", err
		}
		opts.Name = newName
	}

	err = fetcher.Create(opts)
	if err != nil {
		return
	}

	task := NewTask()
	task.fetcherBuilder = fb
	task.fetcher = fetcher
	task.Meta = fetcher.Meta()
	task.Progress = &Progress{}
	task.Status = base.DownloadStatusRunning
	// calculate select files size
	for _, selectIndex := range opts.SelectFiles {
		task.Size += res.Files[selectIndex].Size
	}
	initTask(task)
	task.timer.Start()
	err = func() error {
		d.lock.Lock()
		defer d.lock.Unlock()
		if err := d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
			return err
		}
		d.tasks = append(d.tasks, task)
		return nil
	}()
	if err != nil {
		return
	}
	d.emit(EventKeyStart, task)
	taskId = task.ID
	go func() {
		err := fetcher.Start()
		if err != nil {
			d.emit(EventKeyError, task, err)
			return
		}
		d.watch(task)
	}()
	delete(d.fetcherMap, rrId)
	return
}

func (d *Downloader) Pause(id string) (err error) {
	task := d.GetTask(id)
	if task == nil {
		return
	}
	task.lock.Lock()
	defer task.lock.Unlock()
	if task.Status == base.DownloadStatusRunning {
		task.Status = base.DownloadStatusPause
		task.timer.Pause()
		task.fetcher.Pause()
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.emit(EventKeyPause, task)
	}
	return
}

func (d *Downloader) Continue(id string) (err error) {
	task := d.GetTask(id)
	if task == nil {
		return
	}
	task.lock.Lock()
	defer task.lock.Unlock()
	if task.Status == base.DownloadStatusPause || task.Status == base.DownloadStatusError {
		task.Status = base.DownloadStatusRunning
		task.Progress.Speed = 0
		err = d.restoreFetcher(task)
		if err != nil {
			return
		}
		task.timer.Start()
		err = task.fetcher.Continue()
		if err != nil {
			return
		}
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.emit(EventKeyContinue, task)
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
	if force {
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
	task = nil
	return
}

func (d *Downloader) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.closed.Store(true)
	for _, task := range d.tasks {
		d.Pause(task.ID)
	}
	return d.storage.Close()
}

func (d *Downloader) Clear() error {
	if err := d.Close(); err != nil {
		return err
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
	return d.tasks
}

func (d *Downloader) GetConfig() (bool, *DownloaderStoreConfig, error) {
	var cfg DownloaderStoreConfig
	exist, err := d.storage.Get(bucketConfig, "config", &cfg)
	if err != nil {
		return false, nil, err
	}
	return exist, &cfg, nil
}

func (d *Downloader) PutConfig(v *DownloaderStoreConfig) error {
	return d.storage.Put(bucketConfig, "config", v)
}

func (d *Downloader) getProtocolConfig(name string, v any) (bool, error) {
	exist, cfg, err := d.GetConfig()
	if err != nil {
		return false, err
	}
	if !exist {
		return false, nil
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
		task.fetcher.Create(task.Meta.Opts)
		go d.watch(task)
	}
	return nil
}

func initTask(task *Task) {
	task.timer = &util.Timer{}
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
