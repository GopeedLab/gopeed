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
	"github.com/RoaringBitmap/roaring"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

var (
	cfg       *torrent.ClientConfig
	client    *torrent.Client
	lock      sync.Mutex
	closeCtx  context.Context
	closeFunc func()
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

	cfg = torrent.NewDefaultClientConfig()
	cfg.Seed = true
	cfg.Bep20 = fmt.Sprintf("-GP%s-", parseBep20())
	cfg.ExtendedHandshakeClientVersion = fmt.Sprintf("Gopeed %s", base.Version)
	cfg.ListenPort = f.config.ListenPort
	cfg.HTTPProxy = f.ctl.GetProxy(f.meta.Req.Proxy)
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

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) error {
	f.meta.Req = req
	f.meta.Opts = opts
	if f.meta.Opts == nil {
		f.meta.Opts = &base.Options{}
	}
	if err := f.addTorrent(req, false); err != nil {
		return err
	}
	f.updateRes()
	return nil
}

func (f *Fetcher) Start() (err error) {
	if !f.torrentReady.Load() {
		if err = f.addTorrent(f.meta.Req, false); err != nil {
			return
		}
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
	f.torrent.AllowDataDownload()
	return
}

func (f *Fetcher) Pause() (err error) {
	f.torrent.DisallowDataDownload()
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

func (f *Fetcher) Stats() *fetcher.Stats {
	var stats torrent.TorrentStats
	var pieceMap *base.PieceMap
	pieceMapReady := false
	peers := make([]*base.PeerStats, 0)
	if f.torrent != nil {
		stats = f.torrent.Stats()
		pieceMap, pieceMapReady = buildBTPieceMap(f.torrent)
		peers = buildBTPeers(f.torrent)
	} else {
		stats = torrent.TorrentStats{}
	}
	var snapshot any
	if pieceMapReady {
		snapshot = &bt.StatsSnapshot{
			SeedBytes: f.data.SeedBytes,
			SeedRatio: f.seedRadio(),
			SeedTime:  f.data.SeedTime,
			PieceMap:  pieceMap.Clone(),
		}
	}
	// When completion is temporarily unknown while anacrolix restores storage
	// or verifies pieces, the nil Snapshot tells Downloader to retain the last
	// trusted bitset instead of replacing it with an all-empty map.
	return &fetcher.Stats{
		Snapshot: snapshot,
		Runtime: &bt.StatsRuntime{
			TotalPeers:        stats.TotalPeers,
			ActivePeers:       stats.ActivePeers,
			ConnectedSeeders:  stats.ConnectedSeeders,
			ConnectedLeechers: stats.ActivePeers - stats.ConnectedSeeders,
			Peers:             peers,
		},
	}
}

// buildBTPieceMap converts anacrolix's ordered completion runs to Gopeed's
// one-bit map. ready is false until storage knows the completion state of every
// piece; this prevents initial hash checking from briefly replacing a restored
// map with an all-empty value.
func buildBTPieceMap(t *torrent.Torrent) (pieceMap *base.PieceMap, ready bool) {
	info := t.Info()
	if info == nil {
		return nil, false
	}
	pieceMap = base.NewPieceMap(info.NumPieces(), info.PieceLength)
	ready = applyBTPieceRuns(pieceMap, t.PieceStateRuns())
	return pieceMap, ready
}

func applyBTPieceRuns(pieceMap *base.PieceMap, runs []torrent.PieceStateRun) (ready bool) {
	ready = true
	pieceIndex := 0
	for _, run := range runs {
		if !run.Ok {
			ready = false
		}
		completed := run.Ok && run.Complete
		for range run.Length {
			if pieceIndex >= pieceMap.PieceCount() {
				return ready
			}
			_ = pieceMap.Update(pieceIndex, completed)
			pieceIndex++
		}
	}
	return ready && pieceIndex == pieceMap.PieceCount()
}

func buildBTPeers(t *torrent.Torrent) []*base.PeerStats {
	connections := t.PeerConns()
	totalPieces := 0
	var missingWantedPieces *roaring.Bitmap
	// Magnet torrents can connect to peers before their metadata arrives. Keep
	// those runtime peer stats, but leave completion and relevance unavailable
	// until the metadata provides a trusted total piece count.
	if info := t.Info(); info != nil {
		totalPieces = info.NumPieces()
		missingWantedPieces = btMissingWantedPieces(t.PieceStateRuns(), totalPieces)
	}
	peers := make([]*base.PeerStats, 0, len(connections))
	for _, connection := range connections {
		peerStats := connection.Stats()
		peers = append(peers, &base.PeerStats{
			Address:       safeBTPeerAddress(connection),
			Client:        btPeerClient(connection),
			DownloadSpeed: int64(peerStats.DownloadRate),
			UploadSpeed:   int64(peerStats.LastWriteUploadRate),
			PieceCount:    peerStats.RemotePieceCount,
			Completion:    btPeerCompletion(peerStats.RemotePieceCount, totalPieces),
			Relevance:     btPeerRelevance(connection.PeerPieces(), missingWantedPieces),
			Source:        normalizeBTPeerSource(connection.Discovery),
			Transport:     normalizeBTTransport(connection.Network),
		})
	}
	return peers
}

// btPeerCompletion converts anacrolix's remote piece count into the ratio the
// API exposes. A nil value distinguishes unavailable metadata from a real 0%.
func btPeerCompletion(pieceCount, totalPieces int) *float64 {
	if totalPieces <= 0 {
		return nil
	}
	if pieceCount < 0 {
		pieceCount = 0
	} else if pieceCount > totalPieces {
		pieceCount = totalPieces
	}
	completion := float64(pieceCount) / float64(totalPieces)
	return &completion
}

// btMissingWantedPieces builds the denominator used by peer relevance. Pieces
// belonging only to skipped files are excluded because their priority is none.
func btMissingWantedPieces(runs []torrent.PieceStateRun, totalPieces int) *roaring.Bitmap {
	if totalPieces <= 0 {
		return nil
	}
	missing := roaring.New()
	pieceIndex := 0
	for _, run := range runs {
		for range run.Length {
			if pieceIndex >= totalPieces {
				return nil
			}
			if run.Priority != torrent.PiecePriorityNone && !(run.Ok && run.Complete) {
				missing.Add(uint32(pieceIndex))
			}
			pieceIndex++
		}
	}
	if pieceIndex != totalPieces {
		return nil
	}
	return missing
}

// btPeerRelevance reports how much of the local wanted, missing set the remote
// peer can provide. A zero value is meaningful; nil means there is no denominator.
func btPeerRelevance(peerPieces, missingWantedPieces *roaring.Bitmap) *float64 {
	if peerPieces == nil || missingWantedPieces == nil || missingWantedPieces.IsEmpty() {
		return nil
	}
	relevantPieces := roaring.And(peerPieces, missingWantedPieces).GetCardinality()
	relevance := float64(relevantPieces) / float64(missingWantedPieces.GetCardinality())
	return &relevance
}

// A uTP socket may close between PeerConns and this snapshot. Some uTP address
// implementations panic after close, so one disappearing address must not make
// the entire stats endpoint fail.
func safeBTPeerAddress(connection *torrent.PeerConn) (address string) {
	defer func() {
		if recover() != nil {
			address = ""
		}
	}()
	if connection.RemoteAddr == nil {
		return ""
	}
	return connection.RemoteAddr.String()
}

func btPeerClient(connection *torrent.PeerConn) string {
	value := connection.PeerClientName.Load()
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func normalizeBTPeerSource(source torrent.PeerSource) string {
	switch source {
	case torrent.PeerSourceTracker:
		return "tracker"
	case torrent.PeerSourceDhtGetPeers, torrent.PeerSourceDhtAnnouncePeer:
		return "dht"
	case torrent.PeerSourcePex:
		return "pex"
	case torrent.PeerSourceIncoming:
		return "incoming"
	case torrent.PeerSourceDirect:
		return "direct"
	case torrent.PeerSourceUtHolepunch:
		return "holepunch"
	default:
		return "unknown"
	}
}

func normalizeBTTransport(network string) string {
	network = strings.ToLower(network)
	switch {
	case strings.Contains(network, "webrtc"):
		return "webrtc"
	case strings.Contains(network, "udp"):
		// anacrolix labels uTP sockets with their underlying udp network.
		return "utp"
	case strings.Contains(network, "tcp"):
		return "tcp"
	default:
		return "unknown"
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

// Patch modifies the BT task settings.
// Invalid file indices are silently ignored.
func (f *Fetcher) Patch(req *base.Request, opts *base.Options) error {
	if opts == nil {
		return nil
	}

	if opts.SelectFiles != nil {
		selectFiles := opts.SelectFiles

		// Get file count from resource metadata
		fileCount := 0
		if f.meta.Res != nil {
			fileCount = len(f.meta.Res.Files)
		}

		// Filter out invalid indices (silently ignore)
		validSelectFiles := make([]int, 0, len(selectFiles))
		for _, idx := range selectFiles {
			if idx >= 0 && idx < fileCount {
				validSelectFiles = append(validSelectFiles, idx)
			}
		}

		if f.torrent != nil {
			files := f.torrent.Files()

			// Cancel all current file downloads first
			f.torrent.CancelPieces(0, f.torrent.NumPieces())

			// Apply new file selection
			if len(validSelectFiles) == len(files) {
				f.torrent.DownloadAll()
			} else {
				for _, selectIndex := range validSelectFiles {
					file := files[selectIndex]
					file.Download()
				}
			}
		}

		f.meta.Opts.SelectFiles = validSelectFiles
		// Recalculate the resource size based on new selection
		if f.meta.Res != nil {
			f.meta.Res.CalcSize(validSelectFiles)
		}
		// Reset progress tracking for new file selection
		f.data.Progress = make(fetcher.Progress, len(validSelectFiles))
	}

	return nil
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
	var spec *torrent.TorrentSpec
	if schema == "MAGNET" {
		spec, err = torrent.TorrentSpecFromMagnetUri(req.URL)
		if err != nil {
			return
		}
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
		spec, err = torrent.TorrentSpecFromMetaInfoErr(metaInfo)
		if err != nil {
			return
		}
	}
	spec.Storage = storage.NewFileOpts(storage.NewFileClientOpts{
		ClientBaseDir: cfg.DataDir,
		TorrentDirMaker: func(baseDir string, info *metainfo.Info, infoHash metainfo.Hash) string {
			return f.meta.Opts.Path
		},
	})
	f.torrent, _, err = client.AddTorrentSpec(spec)
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
