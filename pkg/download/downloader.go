package download

import (
	"errors"
	"github.com/google/uuid"
	"github.com/monkeyWie/gopeed-core/internal/controller"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/internal/protocol/http"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"net/url"
	"strings"
	"sync"
	"time"
)

type FetcherBuilder func() (protocols []string, builder func() fetcher.Fetcher)

type TaskInfo struct {
	ID       string
	Res      *base.Resource
	Opts     *base.Options
	Status   base.Status
	Files    map[string]*base.FileInfo
	Progress *Progress

	fetcher fetcher.Fetcher
	timer   *util.Timer
	locker  *sync.Mutex
}

type Progress struct {
	// 下载耗时(纳秒)
	Used int64
	// 每秒下载字节数
	Speed int64
	// 已下载的字节数
	Downloaded int64
}

type Downloader struct {
	*controller.DefaultController
	fetchBuilders map[string]func() fetcher.Fetcher
	tasks         map[string]*TaskInfo
	listener      func(taskInfo *TaskInfo, eventKey EventKey)
}

var DefaultDownloader = NewDownloader(http.FetcherBuilder)

func NewDownloader(fbs ...FetcherBuilder) *Downloader {
	d := &Downloader{DefaultController: controller.NewController()}
	d.fetchBuilders = make(map[string]func() fetcher.Fetcher)
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
					d.emit(task, EventKeyProgress)
				}
			}
			time.Sleep(time.Second)
		}
	}()
	return d
}

func (d *Downloader) buildFetcher(URL string) (fetcher.Fetcher, error) {
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
		timer:    &util.Timer{},
		locker:   new(sync.Mutex),
	}
	d.tasks[id] = task
	task.timer.Start()
	d.emit(task, EventKeyStart)
	err = fetcher.Start()
	if err != nil {
		d.emit(task, EventKeyError)
		return
	}
	go func() {
		err = fetcher.Wait()
		if err != nil {
			d.emit(task, EventKeyError)
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
			d.emit(d.tasks[id], EventKeyDone)
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
	d.emit(task, EventKeyPause)
}

func (d *Downloader) Continue(id string) {
	task := d.tasks[id]
	task.locker.Lock()
	defer task.locker.Unlock()
	task.timer.Continue()
	task.fetcher.Continue()
	d.emit(task, EventKeyContinue)
}

func (d *Downloader) Listener(fn func(taskInfo *TaskInfo, eventKey EventKey)) {
	d.listener = fn
}

func (d *Downloader) emit(taskInfo *TaskInfo, eventKey EventKey) {
	if d.listener != nil {
		d.listener(taskInfo, eventKey)
	}
}

// Resolve 解析资源
func Resolve(url string, extra ...interface{}) (*base.Resource, error) {
	var e interface{}
	if len(extra) == 0 {
		e = nil
	} else {
		e = extra[0]
	}
	return DefaultDownloader.Resolve(&base.Request{
		URL:   url,
		Extra: e,
	})
}
