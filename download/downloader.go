package download

import (
	"errors"
	"github.com/google/uuid"
	"github.com/monkeyWie/gopeed-core/download/common"
	"net/url"
	"strings"
)

type TaskInfo struct {
	ID     string
	Res    *common.Resource
	Opts   *common.Options
	Status common.Status
	Files  map[string]*common.FileInfo

	process common.Process
}

type Downloader struct {
	fetches map[string]common.Fetcher
	tasks   map[string]*TaskInfo
	*common.Controller
}

func NewDownloader(fetchers ...common.Fetcher) *Downloader {
	d := &Downloader{Controller: common.NewController()}
	d.fetches = make(map[string]common.Fetcher)
	for _, f := range fetchers {
		for _, p := range f.Protocols() {
			d.fetches[strings.ToUpper(p)] = f
		}
		f.InitCtl(d.Controller)
	}
	d.tasks = make(map[string]*TaskInfo)
	return d
}

func (d *Downloader) getFetcher(URL string) (common.Fetcher, error) {
	url, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if fetch, ok := d.fetches[strings.ToUpper(url.Scheme)]; ok {
		return fetch, nil
	}
	return nil, errors.New("unsupported protocol")
}

func (d *Downloader) Resolve(req *common.Request) (*common.Resource, error) {
	fetcher, err := d.getFetcher(req.URL)
	if err != nil {
		return nil, err
	}
	return fetcher.Resolve(req)
}

func (d *Downloader) Create(res *common.Resource, opts *common.Options) error {
	fetcher, err := d.getFetcher(res.Req.URL)
	if err != nil {
		return err
	}
	process, err := fetcher.Create(res, opts)
	if err != nil {
		return err
	}
	id := uuid.New().String()
	d.tasks[id] = &TaskInfo{
		ID:      id,
		Res:     res,
		Opts:    opts,
		Status:  common.DownloadStatusReady,
		process: process,
	}
	return process.Start()
}

func (d *Downloader) Pause(id string) {
	d.tasks[id].process.Pause()
}
