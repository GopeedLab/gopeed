package bt

import (
	"bytes"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var (
	client        *torrent.Client
	lock          sync.Mutex
	torrentDirMap = make(map[string]string)
	ftMap         = make(map[string]*fileTorrentImpl)
)

type Fetcher struct {
	ctl *controller.Controller

	torrent *torrent.Torrent
	meta    *fetcher.FetcherMeta

	torrentReady atomic.Bool
	create       atomic.Bool
	progress     fetcher.Progress
}

func (f *Fetcher) Name() string {
	return "bt"
}

func (f *Fetcher) Setup(ctl *controller.Controller) (err error) {
	f.ctl = ctl
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	return
}

func (f *Fetcher) initClient() (err error) {
	lock.Lock()
	defer lock.Unlock()

	if client != nil {
		return
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.ListenPort = 0
	cfg.DefaultStorage = newFileOpts(newFileClientOpts{
		ClientBaseDir: cfg.DataDir,
		HandleFileTorrent: func(infoHash metainfo.Hash, ft *fileTorrentImpl) {
			if dir, ok := torrentDirMap[infoHash.String()]; ok {
				ft.setTorrentDir(dir)
			}
			ftMap[infoHash.String()] = ft
		},
	})
	client, err = torrent.NewClient(cfg)
	return
}

func (f *Fetcher) Resolve(req *base.Request) error {
	if err := base.ParseReqExtra[bt.ReqExtra](req); err != nil {
		return err
	}

	if err := f.addTorrent(req); err != nil {
		return err
	}
	go func() {
		// recycle unused torrent resource
		time.Sleep(time.Minute * 3)
		if !f.create.Load() {
			f.torrentReady.Store(false)
			f.safeDrop()
		}
	}()
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
			Path: util.Dir(file.Path()),
			Size: file.Length(),
		}
		res.Size += file.Length()
	}
	f.meta.Req = req
	f.meta.Res = res
	return nil
}

func (f *Fetcher) Create(opts *base.Options) (err error) {
	f.create.Store(true)
	f.meta.Opts = opts
	if len(opts.SelectFiles) == 0 {
		opts.SelectFiles = make([]int, 0)
		for i := range f.torrent.Files() {
			opts.SelectFiles = append(opts.SelectFiles, i)
		}
	}
	if opts.Path != "" {
		torrentDirMap[f.meta.Res.Hash] = path.Join(f.meta.Opts.Path, f.meta.Res.RootDir)
		if ft, ok := ftMap[f.meta.Res.Hash]; ok {
			// reuse resolve fetcher
			ft.setTorrentDir(torrentDirMap[f.meta.Res.Hash])
		}
	}

	f.progress = make(fetcher.Progress, len(f.meta.Opts.SelectFiles))
	return nil
}

func (f *Fetcher) Start() (err error) {
	if !f.torrentReady.Load() {
		if err = f.addTorrent(f.meta.Req); err != nil {
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
	f.torrentReady.Store(false)
	f.safeDrop()
	return
}

func (f *Fetcher) Continue() (err error) {
	return f.Start()
}

func (f *Fetcher) Close() (err error) {
	f.safeDrop()
	return nil
}

func (f *Fetcher) safeDrop() {
	defer func() {
		// ignore panic
		_ = recover()
	}()

	f.torrent.Drop()
}

func (f *Fetcher) Wait() (err error) {
	for {
		if f.torrentReady.Load() {
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
	if !f.torrentReady.Load() {
		return f.progress
	}
	for i := range f.progress {
		selectIndex := f.meta.Opts.SelectFiles[i]
		file := f.torrent.Files()[selectIndex]
		f.progress[i] = file.BytesCompleted()
	}
	return f.progress
}

func (f *Fetcher) addTorrent(req *base.Request) (err error) {
	if err = f.initClient(); err != nil {
		return
	}
	schema := util.ParseSchema(req.URL)
	if schema == "MAGNET" {
		f.torrent, err = client.AddMagnet(req.URL)
	} else if schema == "APPLICATION/X-BITTORRENT" {
		_, data := util.ParseDataUri(req.URL)
		buf := bytes.NewBuffer(data)
		var metaInfo *metainfo.MetaInfo
		metaInfo, err = metainfo.Load(buf)
		if err != nil {
			return err
		}
		f.torrent, err = client.AddTorrent(metaInfo)
	} else {
		f.torrent, err = client.AddTorrentFromFile(req.URL)
	}
	if err != nil {
		return
	}
	var cfg config
	exist, err := f.ctl.GetConfig(&cfg)
	if err != nil {
		return
	}

	// use map to deduplicate
	trackers := make(map[string]bool)
	if req.Extra != nil {
		extra := req.Extra.(*bt.ReqExtra)
		if len(extra.Trackers) > 0 {
			for _, tracker := range extra.Trackers {
				trackers[tracker] = true
			}
		}
	}
	if exist && len(cfg.Trackers) > 0 {
		for _, tracker := range cfg.Trackers {
			trackers[tracker] = true
		}
	}
	if len(trackers) > 0 {
		announceList := make([][]string, 0)
		for tracker := range trackers {
			announceList = append(announceList, []string{tracker})
		}
		f.torrent.AddTrackers(announceList)
	}
	<-f.torrent.GotInfo()
	f.torrentReady.Store(true)
	return
}

type FetcherBuilder struct {
}

var schemes = []string{"FILE", "MAGNET", "APPLICATION/X-BITTORRENT"}

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
