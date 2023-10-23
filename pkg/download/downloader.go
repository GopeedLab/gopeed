package download

import (
	"errors"
	"fmt"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/logger"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/virtuald/go-paniclog"
	"math"
	"os"
	"path/filepath"
	"sort"
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
	// downloader extension storage bucket
	bucketExtensionStorage = "extension_storage"
)

var (
	ErrTaskNotFound = errors.New("task not found")
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
	Logger          *logger.Logger
	ExtensionLogger *logger.Logger

	cfg             *DownloaderConfig
	fetcherBuilders map[string]fetcher.FetcherBuilder
	fetcherCache    map[string]fetcher.Fetcher
	storage         Storage
	tasks           []*Task
	waitTasks       []*Task
	listener        Listener

	lock               *sync.Mutex
	fetcherMapLock     *sync.RWMutex
	checkDuplicateLock *sync.Mutex
	closed             atomic.Bool

	extensions []*Extension
}

func NewDownloader(cfg *DownloaderConfig) *Downloader {
	if cfg == nil {
		cfg = &DownloaderConfig{}
	}
	cfg.Init()

	d := &Downloader{
		cfg:             cfg,
		fetcherBuilders: make(map[string]fetcher.FetcherBuilder),
		fetcherCache:    make(map[string]fetcher.Fetcher),
		waitTasks:       make([]*Task, 0),
		storage:         cfg.Storage,

		lock:               &sync.Mutex{},
		fetcherMapLock:     &sync.RWMutex{},
		checkDuplicateLock: &sync.Mutex{},

		extensions: make([]*Extension, 0),
	}
	for _, f := range cfg.FetchBuilders {
		for _, p := range f.Schemes() {
			d.fetcherBuilders[strings.ToUpper(p)] = f
		}
	}

	d.Logger = logger.NewLogger(cfg.ProductionMode, filepath.Join(cfg.StorageDir, "logs", "core.log"))
	d.ExtensionLogger = logger.NewLogger(cfg.ProductionMode, filepath.Join(cfg.StorageDir, "logs", "extension.log"))
	if cfg.ProductionMode {
		logPanic(filepath.Join(cfg.StorageDir, "logs"))
	}
	return d
}

func (d *Downloader) Setup() error {
	// setup storage
	if err := d.storage.Setup([]string{bucketTask, bucketSave, bucketConfig, bucketExtension, bucketExtensionStorage}); err != nil {
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
	// sort by create time
	sort.Slice(d.tasks, func(i, j int) bool {
		return d.tasks[i].CreatedAt.Before(d.tasks[j].CreatedAt)
	})

	// load extensions from storage
	var extensions []*Extension
	if err = d.storage.List(bucketExtension, &extensions); err != nil {
		return err
	}
	if extensions == nil {
		extensions = make([]*Extension, 0)
	}
	d.extensions = extensions

	// calculate download speed every tick
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
							d.Logger.Error().Stack().Err(err).Msgf("serialize fetcher failed: %s", task.ID)
							return
						}
						if err := d.storage.Put(bucketSave, task.ID, data); err != nil {
							d.Logger.Error().Stack().Err(err).Msgf("persist fetcher failed: %s", task.ID)
							return
						}
						if err := d.storage.Put(bucketTask, task.ID, task); err != nil {
							d.Logger.Error().Stack().Err(err).Msgf("persist task failed: %s", task.ID)
							return
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
	fetchBuilder, ok := d.fetcherBuilders[schema]
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

	res := d.triggerOnResolve(req)
	if res != nil && len(res.Files) > 0 {
		rr = &ResolveResult{
			Res: res,
		}
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
	d.fetcherMapLock.Lock()
	d.fetcherCache[rrId] = fetcher
	d.fetcherMapLock.Unlock()
	rr = &ResolveResult{
		ID:  rrId,
		Res: fetcher.Meta().Res,
	}
	return
}

type dispatchAction int

const (
	dispatchActionCreate dispatchAction = iota
	dispatchActionPause
	dispatchActionContinue
	dispatchActionDelete
)

func (d *Downloader) sync(fn func() error) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	return fn()
}

func (d *Downloader) dispatch(task *Task, action dispatchAction, cb func(task *Task)) {
	switch action {
	case dispatchActionCreate:
		func() {
			d.lock.Lock()
			defer d.lock.Unlock()

			d.tasks = append(d.tasks, task)
			remainRunningCount := d.remainRunningCount()
			if remainRunningCount == 0 {
				task.Status = base.DownloadStatusWait
				d.waitTasks = append(d.waitTasks, task)
				return
			}

			fmt.Println("create task 111")

			go func() {
				d.lock.Lock()
				defer d.lock.Unlock()

				cb(task)

				fmt.Println("create task 222")
			}()
		}()
		break
	case dispatchActionPause:
		go func() {
			d.lock.Lock()
			defer d.lock.Unlock()

			cb(task)
			d.dequeue()
		}()
		break
	case dispatchActionContinue:
		go func() {
			d.lock.Lock()
			defer d.lock.Unlock()

			remainRunningCount := d.remainRunningCount()
			if remainRunningCount == 0 {
				for _, t := range d.tasks {
					if t != task && t.Status == base.DownloadStatusRunning {
						d.doPause(t)
						t.Status = base.DownloadStatusWait
						d.waitTasks = append(d.waitTasks, t)
					}
				}
			}
			cb(task)
		}()
		break
	case dispatchActionDelete:
		func() {
			d.lock.Lock()
			defer d.lock.Unlock()

			for i, t := range d.tasks {
				if t.ID == task.ID {
					d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
					break
				}
			}

			go func() {
				d.lock.Lock()
				defer d.lock.Unlock()

				cb(task)
				d.dequeue()
			}()
		}()
		break
	}
}

func (d *Downloader) dequeue() {
	remainRunningCount := d.remainRunningCount()
	if remainRunningCount == 0 {
		return
	}
	if len(d.waitTasks) > 0 {
		wt := d.waitTasks[0]
		d.waitTasks = d.waitTasks[1:]
		d.doStart(wt)
	}
}

func (d *Downloader) remainRunningCount() int {
	runningCount := 0
	for _, t := range d.tasks {
		if t.Status == base.DownloadStatusRunning {
			runningCount++
		}
	}
	return d.cfg.MaxRunning - runningCount
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

func (d *Downloader) Create(rrId string, opts *base.Options) (taskId string, err error) {
	if opts == nil {
		opts = &base.Options{}
	}
	d.fetcherMapLock.RLock()
	fetcher, ok := d.fetcherCache[rrId]
	d.fetcherMapLock.RUnlock()
	if !ok {
		return "", errors.New("invalid resource id")
	}
	defer func() {
		d.fetcherMapLock.Lock()
		delete(d.fetcherCache, rrId)
		d.fetcherMapLock.Unlock()
	}()
	return d.doCreate(fetcher, opts)
}

func (d *Downloader) Pause(id string) (err error) {
	task := d.GetTask(id)
	if task == nil {
		return ErrTaskNotFound
	}
	if task.Status == base.DownloadStatusPause {
		return
	}
	task.Status = base.DownloadStatusPause
	d.dispatch(task, dispatchActionPause, func(task *Task) {
		d.doPause(task)
	})
	return
}

func (d *Downloader) Continue(id string) (err error) {
	task := d.GetTask(id)
	if task == nil {
		return ErrTaskNotFound
	}
	if task.Status == base.DownloadStatusRunning {
		return
	}
	task.Status = base.DownloadStatusRunning
	d.dispatch(task, dispatchActionContinue, func(task *Task) {
		d.doContinue(task)
	})
	return
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
	task := d.GetTask(id)
	if task == nil {
		return ErrTaskNotFound
	}
	d.dispatch(task, dispatchActionDelete, func(task *Task) {
		d.doDelete(task, force)
	})
	return
}

func (d *Downloader) doDelete(task *Task, force bool) (err error) {
	err = func() error {
		if task.fetcher != nil {
			if err := task.fetcher.Close(); err != nil {
				return err
			}
		}
		if err := d.storage.Delete(bucketTask, task.ID); err != nil {
			return err
		}
		if err := d.storage.Delete(bucketSave, task.ID); err != nil {
			return err
		}
		if force && task.Meta.Res != nil {
			if task.Meta.Res.Name != "" {
				if err := os.RemoveAll(task.Meta.FolderPath()); err != nil {
					return err
				}
			} else {
				if err := util.SafeRemove(task.Meta.SingleFilepath()); err != nil {
					return err
				}
			}
		}
		d.emit(EventKeyDelete, task)
		task = nil
		return nil
	}()
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("delete task failed, task id: %s", task.ID)
	}

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
	d.tasks = make([]*Task, 0)
	d.extensions = make([]*Extension, 0)
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
		if eventKey == EventKeyError {
			d.emit(EventKeyFinally, task, err)
		}
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
		d.emit(EventKeyFinally, task, err)
	}
	d.dequeue()
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
			d.Logger.Error().Stack().Err(err).Msgf("deserialize fetcher failed, task id: %s", task.ID)
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
	if err = fetcher.Create(opts); err != nil {
		return
	}
	if err = d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
		return
	}
	taskId = task.ID

	go d.watch(task)
	d.sync(func() error {
		d.tasks = append(d.tasks, task)
		d.doStart()
	})
	return
}

func (d *Downloader) doPauseAll() (err error) {
	for _, task := range d.tasks {
		if err = d.doPause(task); err != nil {
			return
		}
	}
	return
}

func (d *Downloader) doStart(task *Task) (err error) {
	d.triggerOnStart(task)

	err = func() error {
		isCreate := task.Status == base.DownloadStatusReady
		task.Status = base.DownloadStatusRunning
		if task.Meta.Res == nil {
			err := task.fetcher.Resolve(task.Meta.Req)
			if err != nil {
				task.Status = base.DownloadStatusError
				d.emit(EventKeyError, task, err)
				return err
			}
			isCreate = true
		}

		if isCreate {
			d.checkDuplicateLock.Lock()
			defer d.checkDuplicateLock.Unlock()
			// check if the download file is duplicated and rename it automatically.
			if task.Meta.Res.Name != "" {
				fullDirPath := task.Meta.FolderPath()
				newName, err := util.CheckDuplicateAndRename(fullDirPath)
				if err != nil {
					return err
				}
				task.Meta.Opts.Name = newName
			} else {
				fullFilePath := task.Meta.SingleFilepath()
				newName, err := util.CheckDuplicateAndRename(fullFilePath)
				if err != nil {
					return err
				}
				task.Meta.Opts.Name = newName
			}
		}

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
	}()
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("start task failed, task id: %s", task.ID)
	}

	return
}

func (d *Downloader) doPause(task *Task) (err error) {
	err = func() error {
		if task.Status != base.DownloadStatusDone {
			task.Status = base.DownloadStatusPause
			task.timer.Pause()
			if err := task.fetcher.Pause(); err != nil {
				return err
			}
			if err := d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
				return err
			}
			d.emit(EventKeyPause, task)
		}
		return nil
	}()
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("pause task failed, task id: %s", task.ID)
		return
	}

	return
}

func (d *Downloader) doContinue(task *Task) (err error) {
	err = func() error {
		if task.Status != base.DownloadStatusRunning && task.Status != base.DownloadStatusDone {
			err := d.restoreFetcher(task)
			if err != nil {
				return err
			}
		}
		if err := d.doStart(task); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("continue task failed, task id: %s", task.ID)
		return
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
