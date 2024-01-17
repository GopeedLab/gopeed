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
	"net/http"
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
	ctl    *controller.Controller
	config *config

	torrent *torrent.Torrent
	meta    *fetcher.FetcherMeta

	torrentReady atomic.Bool
	create       atomic.Bool
	progress     fetcher.Progress
}

func (f *Fetcher) Name() string {
	return "bt"
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	exist := f.ctl.GetConfig(&f.config)
	if !exist {
		f.config = &config{
			ListenPort: 0,
			Trackers:   []string{},
		}
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
	cfg.ListenPort = f.config.ListenPort
	if f.ctl.ProxyUrl != nil {
		cfg.HTTPProxy = http.ProxyURL(f.ctl.ProxyUrl)
	}
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
	f.updateRes()
	f.meta.Req = req
	return nil
}

func (f *Fetcher) Create(opts *base.Options) (err error) {
	f.create.Store(true)
	f.meta.Opts = opts
	if f.meta.Res != nil {
		torrentDirMap[f.meta.Res.Hash] = f.meta.FolderPath()
	}
	return nil
}

func (f *Fetcher) Start() (err error) {
	if !f.torrentReady.Load() {
		if err = f.addTorrent(f.meta.Req); err != nil {
			return
		}
	}
	if ft, ok := ftMap[f.meta.Res.Hash]; ok {
		ft.setTorrentDir(f.meta.FolderPath())
	}
	files := f.torrent.Files()
	// If the user does not specify the file to download, all files will be downloaded by default
	if len(f.meta.Opts.SelectFiles) == 0 {
		f.meta.Opts.SelectFiles = make([]int, len(files))
		for i := range files {
			f.meta.Opts.SelectFiles[i] = i
		}
	}
	f.progress = make(fetcher.Progress, len(f.meta.Opts.SelectFiles))
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
		if f.torrentReady.Load() && len(f.meta.Opts.SelectFiles) > 0 {
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
						util.SafeRemove(filepath.Join(f.meta.Opts.Path, f.meta.Res.Name, file.Path()))
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

func (f *Fetcher) Stats() any {
	stats := f.torrent.Stats()
	if f.meta.Stats == nil {
		baseStats := &bt.Stats{
			TotalPeers:       stats.TotalPeers,
			ActivePeers:      stats.ActivePeers,
			ConnectedSeeders: stats.ConnectedSeeders,
		}
		f.meta.Stats = baseStats
	} else {
		if baseStats, ok := f.meta.Stats.(*bt.Stats); ok {
			baseStats.TotalPeers = stats.TotalPeers
			baseStats.ActivePeers = stats.ActivePeers
			baseStats.ConnectedSeeders = stats.ConnectedSeeders
		}
	}
	return f.meta.Stats
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

func (f *Fetcher) updateRes() {
	res := &base.Resource{
		Name:  f.torrent.Name(),
		Range: true,
		Files: make([]*base.FileInfo, len(f.torrent.Files())),
		Hash:  f.torrent.InfoHash().String(),
	}
	for i, file := range f.torrent.Files() {
		res.Files[i] = &base.FileInfo{
			Name: filepath.Base(file.DisplayPath()),
			Path: util.Dir(file.Path()),
			Size: file.Length(),
		}
	}
	res.CalcSize(nil)
	f.meta.Res = res
	if f.meta.Opts != nil {
		f.meta.Opts.InitSelectFiles(len(res.Files))
	}
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
	if len(f.config.Trackers) > 0 {
		for _, tracker := range f.config.Trackers {
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
