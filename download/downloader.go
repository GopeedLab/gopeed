package download

import (
	"errors"
	"github.com/google/uuid"
	"github.com/monkeyWie/gopeed-core/download/base"
	"net/url"
	"strings"
	"sync"
)

type TaskInfo struct {
	ID     string
	Res    *base.Resource
	Opts   *base.Options
	Status base.Status
	Files  map[string]*base.FileInfo

	fetcher base.Fetcher
	locker  *sync.Mutex
}

type FetcherBuilder func() (protocols []string, builder func() base.Fetcher)

type Downloader struct {
	*base.DefaultController
	fetchBuilders map[string]func() base.Fetcher
	tasks         map[string]*TaskInfo
	listener      func(taskInfo *TaskInfo, status base.Status)
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
	return d
}

func (d *Downloader) buildFetcher(URL string) (base.Fetcher, error) {
	url, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if fetchBuilder, ok := d.fetchBuilders[strings.ToUpper(url.Scheme)]; ok {
		fetcher := fetchBuilder()
		fetcher.Init(d.DefaultController)
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
	d.tasks[id] = &TaskInfo{
		ID:      id,
		Res:     res,
		Opts:    opts,
		Status:  base.DownloadStatusReady,
		fetcher: fetcher,
		locker:  new(sync.Mutex),
	}
	go func() {
		err = fetcher.Start()
		if err != nil {
			d.emit(d.tasks[id], base.DownloadStatusError)
		}
	}()
	d.emit(d.tasks[id], base.DownloadStatusStart)
	return
}

func (d *Downloader) Pause(id string) {
	task := d.tasks[id]
	task.locker.Lock()
	defer task.locker.Unlock()
	d.tasks[id].fetcher.Pause()
	d.emit(d.tasks[id], base.DownloadStatusPause)
}

func (d *Downloader) Continue(id string) {
	task := d.tasks[id]
	task.locker.Lock()
	defer task.locker.Unlock()
	d.tasks[id].fetcher.Continue()
	d.emit(d.tasks[id], base.DownloadStatusStart)
}

func (d *Downloader) Listener(fn func(taskInfo *TaskInfo, status base.Status)) {
	d.listener = fn
}

func (d *Downloader) emit(taskInfo *TaskInfo, status base.Status) {
	if d.listener != nil {
		d.listener(taskInfo, status)
	}
}
