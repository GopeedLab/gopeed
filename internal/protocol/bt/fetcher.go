package bt

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/util"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var client *torrent.Client

type Fetcher struct {
	ctl *controller.Controller

	torrent *torrent.Torrent
	meta    *fetcher.FetcherMeta

	ready    atomic.Bool
	progress fetcher.Progress

	torrentPaths map[string]string
}

func (f *Fetcher) Name() string {
	return "bt"
}

func (f *Fetcher) Setup(ctl *controller.Controller) (err error) {
	f.ctl = ctl
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	var once sync.Once
	once.Do(func() {
		cfg := torrent.NewDefaultClientConfig()
		cfg.ListenPort = 0

		// Support custom download path for each torrent
		pieceCompletion, err := storage.NewDefaultPieceCompletionForDir(cfg.DataDir)
		if err != nil {
			pieceCompletion = storage.NewMapPieceCompletion()
		}
		clientImplCloser := storage.NewFileOpts(storage.NewFileClientOpts{
			ClientBaseDir: cfg.DataDir,
			TorrentDirMaker: func(baseDir string, info *metainfo.Info, infoHash metainfo.Hash) string {
				if dir, ok := f.torrentPaths[infoHash.String()]; ok {
					return dir
				}
				return baseDir
			},
			PieceCompletion: pieceCompletion,
		})
		cfg.DefaultStorage = clientImplCloser
		client, err = torrent.NewClient(cfg)
		if err != nil {
			return
		}

		f.torrentPaths = make(map[string]string)
	})
	return
}

func (f *Fetcher) Resolve(req *base.Request) error {
	if err := f.addTorrent(req.URL); err != nil {
		return err
	}
	defer f.torrent.Drop()
	res := &base.Resource{
		Name:    f.torrent.Name(),
		Range:   true,
		RootDir: f.torrent.Name(),
		Files:   make([]*base.FileInfo, len(f.torrent.Files())),
		Hash:    f.torrent.InfoHash().String(),
	}
	for i, file := range f.torrent.Files() {
		res.Files[i] = &base.FileInfo{
			Name: filepath.Base(file.DisplayPath()),
			Path: util.Dir(path.Join(f.torrent.Info().Name, file.Path())),
			Size: file.Length(),
		}
		res.Size += file.Length()
	}
	f.meta.Req = req
	f.meta.Res = res
	return nil
}

func (f *Fetcher) Create(opts *base.Options) (err error) {
	f.meta.Opts = opts
	if len(opts.SelectFiles) == 0 {
		opts.SelectFiles = make([]int, 0)
		for i := range f.torrent.Files() {
			opts.SelectFiles = append(opts.SelectFiles, i)
		}
	}
	if opts.Path != "" {
		f.torrentPaths[f.meta.Res.Hash] = path.Join(f.meta.Opts.Path, f.meta.Res.RootDir)
	}

	f.progress = make(fetcher.Progress, len(f.meta.Opts.SelectFiles))
	f.ready.Store(false)
	return nil
}

func (f *Fetcher) Start() (err error) {
	if !f.ready.Load() {
		if err = f.addTorrent(f.meta.Req.URL); err != nil {
			return
		}
	}
	files := f.torrent.Files()
	if len(f.meta.Opts.SelectFiles) == len(files) {
		f.torrent.DownloadAll()
	} else {
		for _, selectIndex := range f.meta.Opts.SelectFiles {
			file := files[selectIndex]
			file.Download()
		}
	}
	return
}

func (f *Fetcher) Pause() (err error) {
	f.ready.Store(false)
	f.torrent.Drop()
	return
}

func (f *Fetcher) Continue() (err error) {
	return f.Start()
}

func (f *Fetcher) Close() (err error) {
	f.torrent.Drop()
	return nil
}

func (f *Fetcher) Wait() (err error) {
	for {
		if f.ready.Load() {
			done := true
			for _, selectIndex := range f.meta.Opts.SelectFiles {
				file := f.torrent.Files()[selectIndex]
				if file.BytesCompleted() < file.Length() {
					done = false
					break
				}
			}
			if done {
				// remove unselected files
				for i, file := range f.torrent.Files() {
					selected := false
					for _, selectIndex := range f.meta.Opts.SelectFiles {
						if i == selectIndex {
							selected = true
							break
						}
					}
					if !selected {
						util.SafeRemove(filepath.Join(f.meta.Opts.Path, file.Path()))
					}
				}
				break
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
	return nil
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Progress() fetcher.Progress {
	if !f.ready.Load() {
		return f.progress
	}
	for i := range f.progress {
		selectIndex := f.meta.Opts.SelectFiles[i]
		file := f.torrent.Files()[selectIndex]
		f.progress[i] = file.BytesCompleted()
	}
	return f.progress
}

func (f *Fetcher) addTorrent(url string) (err error) {
	schema := util.ParseSchema(url)
	if schema == "MAGNET" {
		f.torrent, err = client.AddMagnet(url)
	} else {
		f.torrent, err = client.AddTorrentFromFile(url)
	}
	if err != nil {
		return
	}
	var cfg config
	exist, err := f.ctl.GetConfig(&cfg)
	if err != nil {
		return
	}
	if exist && len(cfg.Trackers) > 0 {
		announceList := make([][]string, 0)
		for _, tracker := range cfg.Trackers {
			announceList = append(announceList, []string{tracker})
		}
		f.torrent.AddTrackers(announceList)
	}
	<-f.torrent.GotInfo()
	f.ready.Store(true)
	return
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

func (fb *FetcherBuilder) Store(f fetcher.Fetcher) (data any, err error) {
	return nil, nil
}

func (fb *FetcherBuilder) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return nil, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		return &Fetcher{
			meta: meta,
		}
	}
}
