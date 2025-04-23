package bt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

var (
	client        *torrent.Client
	lock          sync.Mutex
	torrentDirMap = make(map[string]string)
	ftMap         = make(map[string]*fileTorrentImpl)
	closeCtx      context.Context
	closeFunc     func()
)

type Fetcher struct {
	ctl    *controller.Controller
	config *config

	torrent *torrent.Torrent
	meta    *fetcher.FetcherMeta
	data    *fetcherData

	torrentReady    atomic.Bool
	torrentUpload   atomic.Bool
	torrentDropCtx  context.Context
	torrentDropFunc func()
	uploadDoneCh    chan any
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	if f.data == nil {
		f.data = &fetcherData{}
	}
	f.uploadDoneCh = make(chan any, 1)
	f.torrentDropCtx, f.torrentDropFunc = context.WithCancel(context.Background())
	f.ctl.GetConfig(&f.config)
	return
}

func (f *Fetcher) initClient() (err error) {
	lock.Lock()
	defer lock.Unlock()

	if client != nil {
		return
	}
	if closeCtx == nil {
		closeCtx, closeFunc = context.WithCancel(context.Background())
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.Seed = true
	cfg.Bep20 = fmt.Sprintf("-GP%s-", parseBep20())
	cfg.ExtendedHandshakeClientVersion = fmt.Sprintf("Gopeed %s", base.Version)
	cfg.ListenPort = f.config.ListenPort
	cfg.HTTPProxy = f.ctl.GetProxy(f.meta.Req.Proxy)
	cfg.DefaultStorage = newFileOpts(newFileClientOpts{
		ClientBaseDir: cfg.DataDir,
		HandleFileTorrent: func(infoHash metainfo.Hash, ft *fileTorrentImpl) {
			if dir, ok := torrentDirMap[infoHash.String()]; ok {
				ft.setTorrentDir(dir)
			}
			ftMap[infoHash.String()] = ft
		},
	})
	dnsResolver := &DnsCacheResolver{RefreshTimeout: 5 * time.Minute}
	cfg.TrackerDialContext = dnsResolver.DialContext
	client, err = torrent.NewClient(cfg)
	if err != nil {
		return
	}

	closeCtx, closeFunc = context.WithCancel(context.Background())
	go func() {
		dnsResolver.Run(closeCtx)
	}()
	return
}

func (f *Fetcher) Resolve(req *base.Request) error {
	f.meta.Req = req
	if err := f.addTorrent(req, false); err != nil {
		return err
	}
	f.updateRes()
	return nil
}

func (f *Fetcher) Create(opts *base.Options) (err error) {
	f.meta.Opts = opts
	if f.meta.Res != nil {
		torrentDirMap[f.meta.Res.Hash] = opts.Path
	}
	return nil
}

func (f *Fetcher) Start() (err error) {
	if !f.torrentReady.Load() {
		if err = f.addTorrent(f.meta.Req, false); err != nil {
			return
		}
	}
	if ft, ok := ftMap[f.meta.Res.Hash]; ok {
		ft.setTorrentDir(f.meta.Opts.Path)
	}
	files := f.torrent.Files()
	// If the user does not specify the file to download, all files will be downloaded by default
	if f.data.Progress == nil {
		if len(f.meta.Opts.SelectFiles) == 0 {
			f.meta.Opts.SelectFiles = make([]int, len(files))
			for i := range files {
				f.meta.Opts.SelectFiles[i] = i
			}
		}
		f.data.Progress = make(fetcher.Progress, len(f.meta.Opts.SelectFiles))
	}
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
	f.torrentDropFunc()
	f.uploadDoneCh <- nil
	if len(client.Torrents()) == 0 {
		err = closeClient()
	}
	return nil
}

func (f *Fetcher) safeDrop() {
	defer func() {
		// ignore panic
		_ = recover()
	}()

	f.torrent.Drop()
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Stats() any {
	var stats torrent.TorrentStats
	if f.torrent != nil {
		stats = f.torrent.Stats()
	} else {
		stats = torrent.TorrentStats{}
	}
	return &bt.Stats{
		TotalPeers:       stats.TotalPeers,
		ActivePeers:      stats.ActivePeers,
		ConnectedSeeders: stats.ConnectedSeeders,
		SeedBytes:        f.data.SeedBytes,
		SeedRatio:        f.seedRadio(),
		SeedTime:         f.data.SeedTime,
	}
}

func (f *Fetcher) Progress() fetcher.Progress {
	if !f.torrentReady.Load() {
		return f.data.Progress
	}
	for i := range f.data.Progress {
		selectIndex := f.meta.Opts.SelectFiles[i]
		file := f.torrent.Files()[selectIndex]
		f.data.Progress[i] = file.BytesCompleted()
	}
	return f.data.Progress
}

func (f *Fetcher) Wait() (err error) {
	for {
		select {
		case <-f.torrentDropCtx.Done():
			return
		case <-time.After(time.Second):
			if f.torrentReady.Load() && len(f.meta.Opts.SelectFiles) > 0 {
				if f.isDone() {
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
					return
				}
			}
		}
	}
}

func (f *Fetcher) isDone() bool {
	if f.meta.Opts == nil {
		return false
	}
	for _, selectIndex := range f.meta.Opts.SelectFiles {
		file := f.torrent.Files()[selectIndex]
		if file.BytesCompleted() < file.Length() {
			return false
		}
	}
	return true
}

func (f *Fetcher) updateRes() {
	res := &base.Resource{
		Range: true,
		Files: make([]*base.FileInfo, len(f.torrent.Files())),
		Hash:  f.torrent.InfoHash().String(),
	}
	// Directory torrent
	if f.torrent.Info().Length == 0 {
		res.Name = f.torrent.Name()
	}
	for i, file := range f.torrent.Files() {
		res.Files[i] = &base.FileInfo{
			Name: filepath.Base(file.DisplayPath()),
			Path: util.Dir(file.DisplayPath()),
			Size: file.Length(),
		}
	}
	res.CalcSize(nil)
	f.meta.Res = res
	if f.meta.Opts != nil {
		f.meta.Opts.InitSelectFiles(len(res.Files))
	}
}

func (f *Fetcher) Upload() (err error) {
	return f.addTorrent(f.meta.Req, true)
}

func (f *Fetcher) doUpload(fromUpload bool) {
	if !f.torrentUpload.CompareAndSwap(false, true) {
		return
	}

	// Check and update seed data
	lastData := &fetcherData{
		SeedBytes: f.data.SeedBytes,
		SeedTime:  f.data.SeedTime,
	}
	var doneTime int64 = 0
	for {
		select {
		case <-f.torrentDropCtx.Done():
			return
		case <-time.After(time.Second):
			if !f.torrentReady.Load() {
				continue
			}

			stats := f.torrentStats()
			f.data.SeedBytes = lastData.SeedBytes + stats.BytesWrittenData.Int64()

			// Check is download complete, if not don't check and stop seeding
			if !fromUpload && !f.isDone() {
				continue
			}
			if doneTime == 0 {
				doneTime = time.Now().Unix()
			}
			f.data.SeedTime = lastData.SeedTime + time.Now().Unix() - doneTime

			// If the seed forever is true, keep seeding
			if f.config.SeedKeep {
				continue
			}

			// If the seed ratio is reached, stop seeding
			if f.config.SeedRatio > 0 {
				seedRadio := f.seedRadio()
				if seedRadio >= f.config.SeedRatio {
					f.Close()
					break
				}
			}

			// If the seed time is reached, stop seeding
			if f.config.SeedTime > 0 {
				if f.data.SeedTime >= f.config.SeedTime {
					f.Close()
					break
				}
			}
		}
	}
}

// Get torrent stats maybe panic, see https://github.com/anacrolix/torrent/issues/972
func (f *Fetcher) torrentStats() torrent.TorrentStats {
	defer func() {
		if r := recover(); r != nil {
			// ignore panic
		}
	}()

	return f.torrent.Stats()
}

func (f *Fetcher) UploadedBytes() int64 {
	return f.data.SeedBytes
}

func (f *Fetcher) WaitUpload() (err error) {
	<-f.uploadDoneCh
	return nil
}

func (f *Fetcher) addTorrent(req *base.Request, fromUpload bool) (err error) {
	if err = base.ParseReqExtra[bt.ReqExtra](req); err != nil {
		return
	}
	if err = f.initClient(); err != nil {
		return
	}
	schema := util.ParseSchema(req.URL)
	privateTorrent := false
	if schema == "MAGNET" {
		f.torrent, err = client.AddMagnet(req.URL)
	} else {
		var reader io.Reader
		if schema == "FILE" {
			fileUrl, _ := url.Parse(req.URL)
			filePath := fileUrl.Path[1:]
			reader, err = os.Open(filePath)
			if err != nil {
				return
			}
		} else if schema == "DATA" {
			_, data := util.ParseDataUri(req.URL)
			reader = bytes.NewBuffer(data)
		} else {
			reader, err = os.Open(req.URL)
			if err != nil {
				return
			}
			defer reader.(io.Closer).Close()
		}

		var metaInfo *metainfo.MetaInfo
		metaInfo, err = metainfo.Load(reader)
		// Hotfix for https://github.com/anacrolix/torrent/issues/992, ignore "expected EOF" error
		// TODO remove this after the issue is fixed
		if err != nil && !strings.Contains(err.Error(), "expected EOF") {
			return err
		}

		info, er := metaInfo.UnmarshalInfo()
		if er != nil {
			return er
		}

		if info.Private != nil && *info.Private {
			privateTorrent = true
		}
		f.torrent, err = client.AddTorrent(metaInfo)
	}
	if err != nil {
		return
	}

	// Do not add external tracker to a private torrent.
	if !privateTorrent {
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
	}
	<-f.torrent.GotInfo()
	f.torrentReady.Store(true)

	go f.doUpload(fromUpload)
	return
}

func (f *Fetcher) seedRadio() float64 {
	var bytesRead int64
	if f.Meta().Res != nil {
		bytesRead = f.Meta().Res.Size
	} else {
		bytesRead = 0
	}
	if bytesRead <= 0 {
		return 0
	}

	return float64(f.data.SeedBytes) / float64(bytesRead)
}

type fetcherData struct {
	Progress  fetcher.Progress
	SeedBytes int64
	// SeedTime is the time in seconds to seed after downloading is complete.
	SeedTime int64
}

func closeClient() error {
	lock.Lock()
	defer lock.Unlock()

	if closeFunc != nil {
		closeFunc()
	}
	if client != nil {
		errs := client.Close()
		if len(errs) > 0 {
			return errs[0]
		}
		client = nil
		closeCtx = nil
		closeFunc = nil
	}
	return nil
}

type FetcherManager struct {
}

func (fm *FetcherManager) Name() string {
	return "bt"
}

func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: "MAGNET",
		},
		{
			Type:    fetcher.FilterTypeFile,
			Pattern: "TORRENT",
		},
		{
			Type:    fetcher.FilterTypeBase64,
			Pattern: "APPLICATION/X-BITTORRENT",
		},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{}
}

func (fm *FetcherManager) ParseName(u string) string {
	var name string
	url, err := url.Parse(u)
	if err != nil {
		return ""
	}

	params := url.Query()
	if params.Get("dn") != "" {
		return params.Get("dn")
	}
	if params.Get("xt") != "" {
		xt := strings.Split(params.Get("xt"), ":")
		return xt[len(xt)-1]
	}
	return name
}

func (fm *FetcherManager) AutoRename() bool {
	return false
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{
		ListenPort: 0,
		Trackers:   []string{},
		SeedKeep:   false,
		SeedRatio:  1.0,
		SeedTime:   120 * 60,
	}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (data any, err error) {
	_f := f.(*Fetcher)
	return _f.data, nil
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherData{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		return &Fetcher{
			meta: meta,
			data: v.(*fetcherData),
		}
	}
}

func (fm *FetcherManager) Close() error {
	return closeClient()
}

// parse version to bep20 format, fixed length 4, if not enough, fill 0
func parseBep20() string {
	s := strings.ReplaceAll(base.Version, ".", "")
	if len(s) < 4 {
		s += strings.Repeat("0", 4-len(s))
	}
	return s
}
