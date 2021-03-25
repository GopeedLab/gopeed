package download

import (
	"errors"
	"github.com/google/uuid"
	"github.com/monkeyWie/gopeed-core/download/base"
	"net/url"
	"strings"
	"sync"
	"time"
)

type TaskInfo struct {
	ID       string
	Res      *base.Resource
	Opts     *base.Options
	Status   base.Status
	Files    map[string]*base.FileInfo
	Progress *Progress

	fetcher base.Fetcher
	timer   *base.Timer
	locker  *sync.Mutex
}

type FetcherBuilder func() (protocols []string, builder func() base.Fetcher)

type Downloader struct {
	*base.DefaultController
	fetchBuilders map[string]func() base.Fetcher
	tasks         map[string]*TaskInfo
	listener      func(taskInfo *TaskInfo, eventKey base.EventKey)
}

func NewDownloader(fbs ...FetcherBuilder) *Downloader {
	d := &Downloader{DefaultController: base.NewController()}
	d.fetchBuilders = make(map[string]func() base.Fetcher)
	for _, f := range fbs {
		protocols, builder := f()
		for _, p := range protocols {
			d.fetchBuilders[strings.ToUpper(p)] = builder
		}
	}
	d.tasks = make(map[string]*TaskInfo)

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
					d.emit(task, base.EventKeyProgress)
				}
			}
			time.Sleep(time.Second)
		}
	}()
	return d
}

func (d *Downloader) buildFetcher(URL string) (base.Fetcher, error) {
	url, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if fetchBuilder, ok := d.fetchBuilders[strings.ToUpper(url.Scheme)]; ok {
		fetcher := fetchBuilder()
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
	return fetcher.Resolve(req)
}

func (d *Downloader) Create(res *base.Resource, opts *base.Options) (err error) {
	fetcher, err := d.buildFetcher(res.Req.URL)
	if err != nil {
		return
	}
	err = fetcher.Create(res, opts)
	if err != nil {
		return
	}
	id := uuid.New().String()
	task := &TaskInfo{
		ID:       id,
		Res:      res,
		Opts:     opts,
		Status:   base.DownloadStatusReady,
		Progress: &Progress{},
		fetcher:  fetcher,
		timer:    &base.Timer{},
		locker:   new(sync.Mutex),
	}
	d.tasks[id] = task
	task.timer.Start()
	d.emit(task, base.EventKeyStart)
	err = fetcher.Start()
	if err != nil {
		d.emit(task, base.EventKeyError)
		return
	}
	go func() {
		err = fetcher.Wait()
		if err != nil {
			d.emit(task, base.EventKeyError)
		} else {
			task.Progress.Used = task.timer.Used()
			if task.Res.TotalSize == 0 {
				task.Res.TotalSize = task.fetcher.Progress().TotalDownloaded()
			}
			used := task.Progress.Used / int64(time.Second)
			if used == 0 {
				used = 1
			}
			task.Progress.Speed = task.Res.TotalSize / used
			task.Progress.Downloaded = task.Res.TotalSize
			d.emit(d.tasks[id], base.EventKeyDone)
		}
	}()
	return
}

func (d *Downloader) Pause(id string) {
	task := d.tasks[id]
	task.locker.Lock()
	defer task.locker.Unlock()
	task.timer.Pause()
	task.fetcher.Pause()
	d.emit(task, base.EventKeyPause)
}

func (d *Downloader) Continue(id string) {
	task := d.tasks[id]
	task.locker.Lock()
	defer task.locker.Unlock()
	task.timer.Continue()
	task.fetcher.Continue()
	d.emit(task, base.EventKeyContinue)
}

func (d *Downloader) Listener(fn func(taskInfo *TaskInfo, eventKey base.EventKey)) {
	d.listener = fn
}

func (d *Downloader) emit(taskInfo *TaskInfo, eventKey base.EventKey) {
	if d.listener != nil {
		d.listener(taskInfo, eventKey)
	}
}

type Progress struct {
	// 下载耗时(纳秒)
	Used int64
	// 每秒下载字节数
	Speed int64
	// 已下载的字节数
	Downloaded int64
}
