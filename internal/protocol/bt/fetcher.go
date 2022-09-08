package bt

import (
	"github.com/anacrolix/torrent"
	"github.com/monkeyWie/gopeed-core/internal/controller"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"path/filepath"
	"sync"
)

var client *torrent.Client

type Fetcher struct {
	Ctl controller.Controller

	torrent *torrent.Torrent
	opts    *base.Options
}

func (f *Fetcher) Setup(ctl controller.Controller) (err error) {
	f.Ctl = ctl
	var once sync.Once
	once.Do(func() {
		client, err = torrent.NewClient(nil)
	})
	return
}

func (f *Fetcher) Wait() (err error) {
	client.WaitAll()
	return nil
}

func (f *Fetcher) Resolve(req *base.Request) (res *base.Resource, err error) {
	schema := util.ParseSchema(req.URL)
	if schema == "MAGNET" {
		f.torrent, err = client.AddMagnet(req.URL)
	} else {
		f.torrent, err = client.AddTorrentFromFile(req.URL)
	}
	if err != nil {
		return
	}
	<-f.torrent.GotInfo()
	res = &base.Resource{
		Req:   req,
		Size:  f.torrent.Length(),
		Range: true,
		Files: make([]*base.FileInfo, len(f.torrent.Files())),
	}
	for i, file := range f.torrent.Files() {
		res.Files[i] = &base.FileInfo{
			Name: filepath.Base(file.DisplayPath()),
			Path: filepath.Dir(file.Path()),
			Size: file.Length(),
		}
	}
	return
}

func (f *Fetcher) Create(res *base.Resource, opts *base.Options) (err error) {
	f.opts = opts
	return nil
}

func (f *Fetcher) Start() (err error) {
	if len(f.opts.SelectFiles) == 0 {
		f.torrent.DownloadAll()
	} else {
		for _, index := range f.opts.SelectFiles {
			f.torrent.Files()[index].Download()
		}
	}
	return
}

func (f *Fetcher) Pause() (err error) {
	f.torrent.DisallowDataDownload()
	return
}

func (f *Fetcher) Continue() (err error) {
	f.torrent.AllowDataDownload()
	return
}

func (f *Fetcher) Progress() fetcher.Progress {
	p := make(fetcher.Progress, len(f.opts.SelectFiles))
	for i := range p {
		file := f.torrent.Files()[i]
		p[i] = file.BytesCompleted()
	}
	return p
}

type FetcherBuilder struct {
}

var schemes = []string{"FILE", "MAGNET"}

func (fb *FetcherBuilder) Schemes() []string {
	return schemes
}

func (fb *FetcherBuilder) Build() fetcher.Fetcher {
	return &Fetcher{}
}
