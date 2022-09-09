package download

import (
	"errors"
	"github.com/monkeyWie/gopeed-core/internal/controller"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/internal/protocol/bt"
	"github.com/monkeyWie/gopeed-core/internal/protocol/http"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"strings"
	"sync"
	"time"
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
	*controller.DefaultController
	fetchBuilders map[string]fetcher.FetcherBuilder
	tasks         []*Task
	listener      Listener
}

func NewDownloader(fbs ...fetcher.FetcherBuilder) *Downloader {
	d := &Downloader{DefaultController: controller.NewController()}
	d.fetchBuilders = make(map[string]fetcher.FetcherBuilder)
	for _, f := range fbs {
		for _, p := range f.Schemes() {
			d.fetchBuilders[strings.ToUpper(p)] = f
		}
	}
	d.tasks = make([]*Task, 0)

	// 每秒统计一次下载速度
	go func() {
		for {
			if len(d.tasks) > 0 {
				for _, task := range d.tasks {
					if task.Status == base.DownloadStatusDone ||
						task.Status == base.DownloadStatusError ||
						task.Status == base.DownloadStatusPause {
						continue
					}
					current := task.fetcher.Progress().TotalDownloaded()
					task.Progress.Used = task.timer.Used()
					task.Progress.Speed = current - task.Progress.Downloaded
					task.Progress.Downloaded = current
					d.emit(EventKeyProgress, task)
				}
			}
			time.Sleep(time.Second)
		}
	}()
	return d
}

func (d *Downloader) buildFetcher(url string) (fetcher.Fetcher, error) {
	schema := util.ParseSchema(url)
	fetchBuilder, ok := d.fetchBuilders[schema]
	if !ok {
		fetchBuilder, ok = d.fetchBuilders[util.FileSchema]
	}
	if ok {
		fetcher := fetchBuilder.Build()
		fetcher.Setup(d.DefaultController)
		return fetcher, nil
	}
	return nil, errors.New("unsupported protocol")
}

func (d *Downloader) Resolve(req *base.Request) (*base.Resource, error) {
	fetcher, err := d.buildFetcher(req.URL)
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
		opts.SelectFiles = make([]int, len(res.Files))
		for i := range opts.SelectFiles {
			opts.SelectFiles[i] = i
		}
	}

	fetcher, err := d.buildFetcher(res.Req.URL)
	if err != nil {
		return
	}
	err = fetcher.Create(res, opts)
	if err != nil {
		return
	}

	task := NewTask()
	task.fetcher = fetcher
	task.Res = res
	task.Opts = opts
	task.Progress = &Progress{}
	task.timer = &util.Timer{}
	task.locker = new(sync.Mutex)
	task.timer.Start()
	task.Status = base.DownloadStatusRunning
	d.tasks = append(d.tasks, task)
	d.emit(EventKeyStart, task)
	taskId = task.ID

	err = fetcher.Start()
	if err != nil {
		return
	}
	go func() {
		err = fetcher.Wait()
		if err != nil {
			task.Status = base.DownloadStatusError
			d.emit(EventKeyError, task, err)
		} else {
			task.Progress.Used = task.timer.Used()
			if task.Res.Size == 0 {
				task.Res.Size = task.fetcher.Progress().TotalDownloaded()
			}
			used := task.Progress.Used / int64(time.Second)
			if used == 0 {
				used = 1
			}
			task.Progress.Speed = task.Res.Size / used
			task.Progress.Downloaded = task.Res.Size
			task.Status = base.DownloadStatusDone
			d.emit(EventKeyDone, task)
		}
		d.emit(EventKeyFinally, task, err)
	}()
	return
}

func (d *Downloader) Pause(id string) {
	task := d.GetTask(id)
	if task == nil {
		return
	}
	task.locker.Lock()
	defer task.locker.Unlock()
	if task.Status == base.DownloadStatusRunning {
		task.Status = base.DownloadStatusPause
		task.timer.Pause()
		task.fetcher.Pause()
		d.emit(EventKeyPause, task)
	}
}

func (d *Downloader) Continue(id string) {
	task := d.GetTask(id)
	if task == nil {
		return
	}
	task.locker.Lock()
	defer task.locker.Unlock()
	if task.Status == base.DownloadStatusPause || task.Status == base.DownloadStatusError {
		task.Status = base.DownloadStatusRunning
		task.timer.Continue()
		task.fetcher.Continue()
		d.emit(EventKeyContinue, task)
	}
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

var defaultDownloader = NewDownloader(new(http.FetcherBuilder), new(bt.FetcherBuilder))

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
	return &boot{}
}
