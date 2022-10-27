package download

import (
	"errors"
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/internal/protocol/bt"
	"github.com/monkeyWie/gopeed/internal/protocol/http"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/util"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bucketTask   = "task"
	bucketSave   = "save"
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
	controller    *controller.DefaultController
	fetchBuilders map[string]fetcher.FetcherBuilder
	storage       Storage
	tasks         []*Task
	listener      Listener

	refreshInterval int
	lock            *sync.Mutex
	closed          atomic.Bool
}

type DownloaderConfig struct {
	Controller    *controller.DefaultController
	FetchBuilders []fetcher.FetcherBuilder
	Storage       Storage

	// RefreshInterval time duration to refresh task progress(ms)
	RefreshInterval int
}

func (cfg *DownloaderConfig) Init() *DownloaderConfig {
	if cfg.Controller == nil {
		cfg.Controller = controller.NewController()
	}
	if len(cfg.FetchBuilders) == 0 {
		cfg.FetchBuilders = []fetcher.FetcherBuilder{
			new(http.FetcherBuilder),
			new(bt.FetcherBuilder),
		}
	}
	if cfg.Storage == nil {
		cfg.Storage = NewMemStorage()
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 1000
	}
	return cfg
}

func NewDownloader(cfg *DownloaderConfig) *Downloader {
	if cfg == nil {
		cfg = &DownloaderConfig{}
	}
	cfg.Init()

	d := &Downloader{
		controller:      cfg.Controller,
		fetchBuilders:   make(map[string]fetcher.FetcherBuilder),
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
		for _, task := range tasks {
			initTask(task)
			if task.Status != base.DownloadStatusDone && task.Status != base.DownloadStatusError {
				task.Status = base.DownloadStatusPause
			}
		}
	}
	d.tasks = tasks

	// 每秒统计一次下载速度
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
						task.Progress.Speed = current - task.Progress.Downloaded
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

func (d *Downloader) buildFetcher(url string) (fetcher.FetcherBuilder, fetcher.Fetcher, error) {
	schema := util.ParseSchema(url)
	fetchBuilder, ok := d.fetchBuilders[schema]
	if !ok {
		fetchBuilder, ok = d.fetchBuilders[util.FileSchema]
	}
	if ok {
		fetcher := fetchBuilder.Build()
		fetcher.Setup(d.controller)
		return fetchBuilder, fetcher, nil
	}
	return nil, nil, errors.New("unsupported protocol")
}

func (d *Downloader) Resolve(req *base.Request) (*base.Resource, error) {
	_, fetcher, err := d.buildFetcher(req.URL)
	if err != nil {
		return nil, err
	}
	res, err := fetcher.Resolve(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (d *Downloader) Create(res *base.Resource, opts *base.Options) (taskId string, err error) {
	if opts == nil {
		opts = &base.Options{}
	}
	if !res.Range || opts.Connections < 1 {
		opts.Connections = 1
	}
	if len(opts.SelectFiles) == 0 {
		opts.SelectFiles = make([]int, 0)
		for i := range res.Files {
			opts.SelectFiles = append(opts.SelectFiles, i)
		}
	}

	fetcherBuilder, fetcher, err := d.buildFetcher(res.Req.URL)
	if err != nil {
		return
	}
	err = fetcher.Create(res, opts)
	if err != nil {
		return
	}

	task := NewTask()
	task.fetcherBuilder = fetcherBuilder
	task.fetcher = fetcher
	task.Res = res
	task.Opts = opts
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
		names := make([]string, 0)
		for _, file := range task.Res.Files {
			names = append(names, util.Filepath(file.Path, file.Name, task.Opts.Name))
		}
		util.SafeRemoveAll(task.Opts.Path, names)
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

func (d *Downloader) GetConfig(v any) (bool, error) {
	return d.storage.Get(bucketConfig, "config", v)
}

func (d *Downloader) PutConfig(v any) error {
	return d.storage.Put(bucketConfig, "config", v)
}

func (d *Downloader) watch(task *Task) {
	err := task.fetcher.Wait()
	if err != nil {
		task.Status = base.DownloadStatusError
		d.emit(EventKeyError, task, err)
	} else {
		task.Progress.Used = task.timer.Used()
		if task.Size == 0 {
			task.Size = task.fetcher.Progress().TotalDownloaded()
			task.Res.Size = task.Size
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
		fetcherBuilder, _, err := d.buildFetcher(task.Res.Req.URL)
		if err != nil {
			return err
		}
		task.fetcherBuilder = fetcherBuilder
		err = func() error {
			v, f := fetcherBuilder.Restore()
			if v != nil {
				err := d.storage.Pop(bucketSave, task.ID, v)
				if err != nil {
					return err
				}
				task.fetcher = f(task.Res, task.Opts, v)
			}
			return nil
		}()
		if err != nil {
			// TODO log
		}
		if task.fetcher == nil {
			task.fetcher = fetcherBuilder.Build()
			task.fetcher.Create(task.Res, task.Opts)
		}
		task.fetcher.Setup(d.controller)
		go d.watch(task)
	}
	return nil
}

func initTask(task *Task) {
	task.timer = &util.Timer{}
	task.lock = &sync.Mutex{}
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

func (b *boot) Resolve() (*base.Resource, error) {
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
	res, err := b.Resolve()
	if err != nil {
		return "", err
	}
	defaultDownloader.Listener(b.listener)
	return defaultDownloader.Create(res, opts)
}

func Boot() *boot {
	err := defaultDownloader.Setup()
	if err != nil {
		panic(err)
	}
	return &boot{}
}
