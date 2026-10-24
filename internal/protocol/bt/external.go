package bt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/anacrolix/torrent/metainfo"
)

// externalData marks a task that was handed to rtorrent. It is saved with the
// task, so the task stays with rtorrent whatever the setting says later.
type externalData struct {
	// Hash is the v1 info hash in rtorrent's form, 40 uppercase hex digits.
	Hash string
	// Magnet is true when rtorrent fetches the metadata and Gopeed learns the
	// files from rtorrent.
	Magnet bool
	// Added is true once rtorrent holds the download.
	Added bool
	// Directory is the folder rtorrent saves into. A magnet's real download
	// replaces its metadata download and needs the folder again.
	Directory string
}

// externalState is what the fetcher knows while it runs; it is not saved.
type externalState struct {
	torrent []byte // the .torrent bytes read at resolve
	status  *rtorrentStatus
	running bool // started in Gopeed and not paused since
}

var errExternalOff = errors.New("this task was handed to rtorrent, but sending torrents to rtorrent is off now")

// externalPollInterval is how often a running task asks rtorrent for its state.
var externalPollInterval = time.Second

// externalMaxFailures is how many polls in a row may fail, for example while
// rtorrent restarts, before the task fails.
const externalMaxFailures = 30

func (f *Fetcher) isExternal() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data.External != nil
}

// externalResolveLocked reads the torrent without the built-in engine. A
// .torrent is parsed here, so the file list and the file selection work as
// usual. A magnet is named after its dn or info hash; rtorrent fetches the
// metadata after the start. The caller holds f.mu.
func (f *Fetcher) externalResolveLocked(req *base.Request) error {
	if err := base.ParseReqExtra[bt.ReqExtra](req); err != nil {
		return err
	}
	f.meta.Req = req
	f.ext = &externalState{}
	if util.ParseSchema(req.URL) == "MAGNET" {
		m, err := metainfo.ParseMagnetUri(req.URL)
		if err != nil {
			return fmt.Errorf("rtorrent needs a magnet link with a v1 info hash (urn:btih): %w", err)
		}
		hash := strings.ToUpper(m.InfoHash.HexString())
		name := m.DisplayName
		if name == "" {
			name = hash
		}
		f.data.External = &externalData{Hash: hash, Magnet: true}
		f.meta.Res = &base.Resource{Name: name, Range: true, Hash: strings.ToLower(hash)}
		return nil
	}
	data, err := readTorrentSource(req.URL)
	if err != nil {
		return err
	}
	mi, err := metainfo.Load(bytes.NewReader(data))
	// Same hotfix as addTorrent, see https://github.com/anacrolix/torrent/issues/992
	if err != nil && !strings.Contains(err.Error(), "expected EOF") {
		return err
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		return err
	}
	if !info.HasV1() {
		return errors.New("rtorrent supports v1 torrents only, and this torrent has no v1 info hash")
	}
	hash := strings.ToUpper(mi.HashInfoBytes().HexString())
	f.ext.torrent = data
	f.data.External = &externalData{Hash: hash}
	f.meta.Res = resourceFromInfo(&info, hash)
	if f.meta.Opts != nil {
		f.meta.Opts.InitSelectFiles(len(f.meta.Res.Files))
	}
	return nil
}

// resourceFromInfo builds the resource the same way updateRes does for the
// built-in engine.
func resourceFromInfo(info *metainfo.Info, hash string) *base.Resource {
	files := info.UpvertedFiles()
	res := &base.Resource{
		Range: true,
		Files: make([]*base.FileInfo, len(files)),
		Hash:  strings.ToLower(hash),
	}
	if len(files) > 0 {
		res.Name, _ = torrentFileLayout(info, files[0])
	}
	for i := range files {
		display := files[i].DisplayPath(info)
		res.Files[i] = &base.FileInfo{
			Name: filepath.Base(display),
			Path: util.Dir(display),
			Size: files[i].Length,
		}
	}
	res.CalcSize(nil)
	return res
}

// readTorrentSource reads a .torrent from a file:// URL, a data: URI or a
// local path, the sources the built-in engine reads. An http(s) .torrent link
// is downloaded by the HTTP protocol first and arrives here as a local file.
func readTorrentSource(raw string) ([]byte, error) {
	switch util.ParseSchema(raw) {
	case "FILE":
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		p := u.Path
		if runtime.GOOS == "windows" && len(p) > 2 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		return os.ReadFile(filepath.FromSlash(p))
	case "DATA":
		_, data := util.ParseDataUri(raw)
		return data, nil
	default:
		return os.ReadFile(raw)
	}
}

// externalTarget reads the current settings: is the hand-off on, and where.
func (f *Fetcher) externalTarget() (bool, *externalConfig) {
	var current *config
	f.ctl.GetConfig(&current)
	if current == nil {
		return false, nil
	}
	on, endpoint := current.External.target()
	cfg := current.External
	cfg.URL = endpoint
	return on, &cfg
}

// externalClient builds the rtorrent client from the current settings.
func (f *Fetcher) externalClient() (*rtorrentClient, *externalConfig, error) {
	on, cfg := f.externalTarget()
	if !on {
		return nil, nil, errExternalOff
	}
	c, err := newRtorrentClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	return c, cfg, nil
}

// externalStart hands the download to rtorrent on the first start, or starts
// it again in rtorrent after a pause. A new download is loaded stopped, gets
// its folder and file selection through calls of their own, and only then
// starts. The network calls run without f.mu, so progress reads never wait
// for rtorrent.
func (f *Fetcher) externalStart() error {
	f.mu.Lock()
	if f.meta.Opts == nil || f.meta.Opts.Path == "" {
		f.mu.Unlock()
		return errors.New("download directory is required when starting a torrent")
	}
	if f.ext == nil {
		f.ext = &externalState{}
	}
	ext := *f.data.External
	reqURL := f.meta.Req.URL
	optsPath := f.meta.Opts.Path
	torrentData := f.ext.torrent
	fileCount := 0
	if f.meta.Res != nil {
		fileCount = len(f.meta.Res.Files)
	}
	var selection []int
	if !ext.Magnet && len(f.meta.Opts.SelectFiles) > 0 && len(f.meta.Opts.SelectFiles) < fileCount {
		selection = append(selection, f.meta.Opts.SelectFiles...)
	}
	f.mu.Unlock()

	ctx := f.torrentDropCtx
	if err := ctx.Err(); err != nil {
		return err
	}
	client, cfg, err := f.externalClient()
	if err != nil {
		return err
	}
	if !ext.Added {
		ok, err := client.has(ctx, ext.Hash)
		if err != nil {
			return err
		}
		if ok {
			return fmt.Errorf("rtorrent already has the torrent %s; remove it there, or manage it in rtorrent", ext.Hash)
		}
		dir := cfg.Directory
		if dir == "" {
			dir = optsPath
		}
		if ext.Magnet {
			err = client.loadMagnet(ctx, reqURL)
		} else {
			if torrentData == nil {
				// Resolved before a restart: read the source again.
				if torrentData, err = readTorrentSource(reqURL); err != nil {
					return err
				}
			}
			err = client.loadRaw(ctx, torrentData)
		}
		if err != nil {
			return err
		}
		if err = client.waitLoaded(ctx, ext.Hash, 10*time.Second); err != nil {
			return err
		}
		if err = client.setDirectory(ctx, ext.Hash, dir); err != nil {
			return err
		}
		if selection != nil {
			if err = client.setSelection(ctx, ext.Hash, fileCount, selection); err != nil {
				return err
			}
		}
		f.mu.Lock()
		f.data.External.Directory = dir
		f.data.External.Added = true
		f.mu.Unlock()
	}

	f.mu.Lock()
	if f.data.Progress == nil {
		n := 1
		if f.meta.Res != nil && len(f.meta.Res.Files) > 0 {
			if len(f.meta.Opts.SelectFiles) == 0 {
				f.meta.Opts.InitSelectFiles(len(f.meta.Res.Files))
			}
			n = len(f.meta.Opts.SelectFiles)
		}
		f.data.Progress = make(fetcher.Progress, n)
	}
	f.ext.running = true
	f.mu.Unlock()
	return client.start(ctx, ext.Hash)
}

func (f *Fetcher) externalPause() error {
	f.mu.Lock()
	if f.ext != nil {
		f.ext.running = false
	}
	ext := *f.data.External
	f.mu.Unlock()
	if !ext.Added {
		return nil
	}
	client, _, err := f.externalClient()
	if err != nil {
		return err
	}
	return client.stop(context.Background(), ext.Hash)
}

// externalClose runs when the task is deleted: rtorrent forgets the download.
// rtorrent never deletes data; the downloader removes the files when the user
// asks for that. An unreachable rtorrent does not keep the task.
func (f *Fetcher) externalClose() {
	f.mu.Lock()
	ext := *f.data.External
	f.mu.Unlock()
	if !ext.Added {
		return
	}
	client, _, err := f.externalClient()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.erase(ctx, ext.Hash)
}

func (f *Fetcher) externalPatch(opts *base.Options) error {
	f.mu.Lock()
	if f.meta.Opts == nil {
		f.meta.Opts = &base.Options{}
	}
	if opts == nil || opts.SelectFiles == nil || f.meta.Res == nil || len(f.meta.Res.Files) == 0 {
		f.mu.Unlock()
		return nil
	}
	fileCount := len(f.meta.Res.Files)
	valid := make([]int, 0, len(opts.SelectFiles))
	for _, i := range opts.SelectFiles {
		if i >= 0 && i < fileCount {
			valid = append(valid, i)
		}
	}
	ext := *f.data.External
	f.mu.Unlock()

	if ext.Added {
		client, _, err := f.externalClient()
		if err != nil {
			return err
		}
		var selection []int
		if len(valid) < fileCount {
			selection = valid
		}
		if err := client.setSelection(context.Background(), ext.Hash, fileCount, selection); err != nil {
			return err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.meta.Opts.SelectFiles = valid
	f.meta.Res.CalcSize(valid)
	f.data.Progress = make(fetcher.Progress, len(valid))
	return nil
}

// externalWait polls rtorrent until every selected file is complete, the task
// is deleted, or rtorrent loses the download. Like the built-in engine, a
// pause does not end the wait.
func (f *Fetcher) externalWait() error {
	ticker := time.NewTicker(externalPollInterval)
	defer ticker.Stop()
	failures := 0
	for {
		select {
		case <-f.torrentDropCtx.Done():
			return nil
		case <-ticker.C:
		}
		done, err := f.externalPoll()
		switch {
		case errors.Is(err, errRtorrentGone), errors.Is(err, errExternalOff):
			return err
		case err != nil:
			if f.torrentDropCtx.Err() != nil {
				return nil
			}
			failures++
			if failures >= externalMaxFailures {
				return err
			}
			continue
		}
		failures = 0
		if done {
			return nil
		}
	}
}

// externalPoll reads the download once and updates progress, stats and, for a
// magnet whose metadata just arrived, the resource.
func (f *Fetcher) externalPoll() (bool, error) {
	f.mu.Lock()
	ext := *f.data.External
	needFiles := ext.Magnet && (f.meta.Res == nil || len(f.meta.Res.Files) == 0)
	running := f.ext != nil && f.ext.running
	f.mu.Unlock()
	if !ext.Added {
		return false, nil
	}
	client, _, err := f.externalClient()
	if err != nil {
		return false, err
	}
	ctx := f.torrentDropCtx
	st, err := client.status(ctx, ext.Hash)
	if err != nil {
		return false, err
	}
	var files []rtorrentFile
	if !st.Meta {
		if files, err = client.files(ctx, ext.Hash); err != nil {
			return false, err
		}
	}
	if !st.Meta && needFiles {
		if err := f.externalAdoptMetadata(ctx, client, &ext, st, files, running); err != nil {
			return false, err
		}
		// The state changed; read it again on the next poll.
		return false, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ext == nil {
		f.ext = &externalState{}
	}
	f.ext.status = st
	f.data.SeedBytes = st.UpTotal
	if st.Meta {
		return false, nil
	}
	if len(f.data.Progress) == len(f.meta.Opts.SelectFiles) && len(files) == len(f.meta.Res.Files) {
		for i, index := range f.meta.Opts.SelectFiles {
			if index >= 0 && index < len(files) {
				f.data.Progress[i] = files[index].completedBytes()
			}
		}
	}
	return st.done(), nil
}

// externalAdoptMetadata runs once a magnet's metadata has arrived. rtorrent
// then replaces the metadata download with the real one, stopped and in its
// default folder, so the folder is set again before it starts. The resource
// gets the real name, size and files.
func (f *Fetcher) externalAdoptMetadata(ctx context.Context, client *rtorrentClient, ext *externalData, st *rtorrentStatus, files []rtorrentFile, running bool) error {
	if ext.Directory != "" && filepath.Dir(st.Directory) != filepath.Clean(ext.Directory) && st.Directory != filepath.Clean(ext.Directory) {
		if st.Started {
			// rtorrent 0.16 keeps the real download stopped, but a started
			// download cannot change its folder.
			if err := client.stop(ctx, ext.Hash); err != nil {
				return err
			}
		}
		if err := client.setDirectory(ctx, ext.Hash, ext.Directory); err != nil {
			return err
		}
	}
	res := &base.Resource{Range: true, Hash: strings.ToLower(ext.Hash), Files: make([]*base.FileInfo, len(files))}
	if st.MultiFile {
		res.Name = st.Name
	}
	for i, file := range files {
		display := file.Path
		if !st.MultiFile {
			display = st.Name
		}
		res.Files[i] = &base.FileInfo{
			Name: filepath.Base(display),
			Path: util.Dir(display),
			Size: file.SizeBytes,
		}
	}
	res.CalcSize(nil)

	// The resource and the options are swapped as whole values, never emptied
	// and refilled, because the downloader reads them while the task runs.
	f.mu.Lock()
	opts := *f.meta.Opts
	opts.SelectFiles = nil
	opts.InitSelectFiles(len(res.Files))
	f.meta.Res = res
	f.meta.Opts = &opts
	f.data.Progress = make(fetcher.Progress, len(res.Files))
	if f.ext == nil {
		f.ext = &externalState{}
	}
	f.ext.status = st
	f.mu.Unlock()
	if running {
		return client.start(ctx, ext.Hash)
	}
	return nil
}

func (f *Fetcher) externalStatsLocked() *fetcher.Stats {
	runtime := &bt.StatsRuntime{Peers: []*base.PeerStats{}}
	snapshot := &bt.StatsSnapshot{SeedBytes: f.data.SeedBytes}
	if f.ext != nil && f.ext.status != nil {
		st := f.ext.status
		snapshot.SeedRatio = float64(st.Ratio) / 1000
		runtime.TotalPeers = int(st.Peers)
		runtime.ActivePeers = int(st.Peers)
		runtime.ConnectedSeeders = int(st.Seeders)
		runtime.ConnectedLeechers = int(st.Peers - st.Seeders)
	}
	return &fetcher.Stats{Snapshot: snapshot, Runtime: runtime}
}
