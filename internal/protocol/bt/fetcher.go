package bt

import (
	"bufio"
	"fmt"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/util"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var client *torrent.Client

type Fetcher struct {
	Ctl *controller.Controller
	*config

	torrent *torrent.Torrent
	res     *base.Resource
	opts    *base.Options

	ready    atomic.Bool
	progress fetcher.Progress

	torrentPaths map[string]string
}

func (f *Fetcher) Name() string {
	return "bt"
}

func (f *Fetcher) Setup(ctl *controller.Controller) (err error) {
	f.Ctl = ctl
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

	// load config
	f.config = &config{}
	return
}

func (f *Fetcher) Wait() (err error) {
	for {
		if f.ready.Load() {
			done := true
			for _, selectIndex := range f.opts.SelectFiles {
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
					for _, selectIndex := range f.opts.SelectFiles {
						if i == selectIndex {
							selected = true
							break
						}
					}
					if !selected {
						util.SafeRemove(filepath.Join(f.opts.Path, file.Path()))
					}
				}
				break
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
	return nil
}

func (f *Fetcher) Resolve(req *base.Request) (res *base.Resource, err error) {
	if err = f.addTorrent(req.URL, true); err != nil {
		return
	}
	defer f.torrent.Drop()
	res = &base.Resource{
		Req:   req,
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
		res.Size += file.Length()
	}
	return
}

func (f *Fetcher) Create(res *base.Resource, opts *base.Options) (err error) {
	f.res = res
	f.opts = opts
	if len(f.opts.SelectFiles) == 0 {
		f.opts.SelectFiles = make([]int, 0)
		for i := range f.torrent.Files() {
			f.opts.SelectFiles = append(f.opts.SelectFiles, i)
		}
	}
	if f.opts.Path != "" {
		f.torrentPaths[res.Hash] = f.opts.Path
	}

	f.progress = make(fetcher.Progress, len(f.opts.SelectFiles))
	f.ready.Store(false)
	return nil
}

func (f *Fetcher) Start() (err error) {
	if err = f.addTorrent(f.res.Req.URL, false); err != nil {
		return
	}
	files := f.torrent.Files()
	if len(f.opts.SelectFiles) == len(files) {
		f.torrent.DownloadAll()
	} else {
		for _, selectIndex := range f.opts.SelectFiles {
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

func (f *Fetcher) Progress() fetcher.Progress {
	if !f.ready.Load() {
		return f.progress
	}
	for i := range f.progress {
		selectIndex := f.opts.SelectFiles[i]
		file := f.torrent.Files()[selectIndex]
		f.progress[i] = file.BytesCompleted()
	}
	return f.progress
}

func (f *Fetcher) addTorrent(url string, resolve bool) (err error) {
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
	exist, err := f.Ctl.GetConfig(&cfg)
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
	if resolve {
		return
	}
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

func (fb *FetcherBuilder) Handle(action string, params any) (ret any, err error) {
	switch action {
	// resolve tracker subscribe url
	case "resolve":
		url, ok := params.(string)
		if !ok || url == "" {
			return nil, base.BadParams
		}
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("http request fail, code: %d", resp.StatusCode)
		}
		ret := make([][]string, 0)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				ret = append(ret, []string{line})
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return ret, nil
	default:
		return nil, base.NotFound
	}
	return
}

func (fb *FetcherBuilder) Store(f fetcher.Fetcher) (data any, err error) {
	return nil, nil
}

func (fb *FetcherBuilder) Restore() (v any, f func(res *base.Resource, opts *base.Options, v any) fetcher.Fetcher) {
	return nil, nil
}
