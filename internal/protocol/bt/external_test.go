package bt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
)

func init() {
	externalPollInterval = 20 * time.Millisecond
}

// externalTestConfig is the stored BitTorrent config of a test, so a test can
// change the switch between calls.
type externalTestConfig struct {
	cfg atomic.Pointer[config]
}

func newExternalTestConfig(url, directory string) *externalTestConfig {
	c := new(FetcherManager).DefaultConfig().(*config)
	on := true
	c.External = externalConfig{Enable: &on, URL: url, Directory: directory}
	tc := &externalTestConfig{}
	tc.cfg.Store(c)
	return tc
}

func (tc *externalTestConfig) controller() *controller.Controller {
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		data, _ := json.Marshal(tc.cfg.Load())
		_ = json.Unmarshal(data, v)
	}
	return ctl
}

func (tc *externalTestConfig) setEnable(on bool) {
	c := *tc.cfg.Load()
	c.External.Enable = &on
	tc.cfg.Store(&c)
}

func buildExternalFetcher(t *testing.T, tc *externalTestConfig) *Fetcher {
	t.Helper()
	// The built-in engine's client is shared by the package. Close what an
	// earlier test left, so assertNoBuiltInClient sees only this test.
	if err := closeClient(); err != nil {
		t.Fatal(err)
	}
	f := new(FetcherManager).Build().(*Fetcher)
	f.Setup(tc.controller())
	t.Cleanup(func() { f.Close() })
	return f
}

func waitExternal(t *testing.T, f *Fetcher) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- f.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Wait did not return")
		return nil
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertNoBuiltInClient(t *testing.T) {
	t.Helper()
	lock.Lock()
	defer lock.Unlock()
	if client != nil {
		t.Fatal("the built-in BitTorrent engine started; the rtorrent hand-off must not open its listen port")
	}
}

func TestExternalTorrentFileWithSelection(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, hash := test.WriteTorrent(t, "bundle, \"one\"", map[string][]byte{
		"a.bin":     make([]byte, 40000),
		"sub/b.bin": make([]byte, 20000),
		"c.bin":     make([]byte, 30000),
	})

	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	res := f.Meta().Res
	if res.Name != `bundle, "one"` || len(res.Files) != 3 || res.Hash != strings.ToLower(hash) {
		t.Fatalf("resolved %+v", res)
	}
	if rt.Called("load.raw") != 0 {
		t.Fatal("resolve must not hand the torrent over yet")
	}
	// Files are in torrent order: a.bin, c.bin, sub/b.bin.
	dir := t.TempDir()
	f.Meta().Opts = &base.Options{Path: dir, SelectFiles: []int{0, 2}}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	d := rt.Download(hash)
	if d == nil || !d.Started {
		t.Fatalf("rtorrent download %+v, want started", d)
	}
	if d.Directory != dir {
		t.Fatalf("directory %q, want the task folder %q", d.Directory, dir)
	}
	if d.Files[0].Priority != 1 || d.Files[1].Priority != 0 || d.Files[2].Priority != 1 {
		t.Fatalf("priorities %d %d %d, want 1 0 1", d.Files[0].Priority, d.Files[1].Priority, d.Files[2].Priority)
	}
	assertNoBuiltInClient(t)

	rt.CompleteFile(hash, 0)
	waitDone := make(chan error, 1)
	go func() { waitDone <- f.Wait() }()
	eventually(t, "the progress of the first file", func() bool { return f.Progress()[0] == 40000 })
	if p := f.Progress(); len(p) != 2 || p[1] != 0 {
		t.Fatalf("progress %v, want [40000 0]", p)
	}
	select {
	case err := <-waitDone:
		t.Fatalf("Wait returned %v before the selected files were complete", err)
	default:
	}
	rt.CompleteFile(hash, 2)
	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return after the selected files were complete")
	}
	if p := f.Progress(); p[0] != 40000 || p[1] != 20000 {
		t.Fatalf("final progress %v", p)
	}
}

func TestExternalDirectoryFromConfig(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "/srv/torrents, incoming")
	torrentPath, _, hash := test.WriteTorrent(t, "single.bin", map[string][]byte{"": make([]byte, 5000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: "file://" + filepath.ToSlash(torrentPath)}); err != nil {
		t.Fatal(err)
	}
	if res := f.Meta().Res; res.Name != "" || len(res.Files) != 1 || res.Files[0].Name != "single.bin" {
		t.Fatalf("resolved %+v %+v", res, res.Files[0])
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if d := rt.Download(hash); d.Directory != "/srv/torrents, incoming" {
		t.Fatalf("directory %q", d.Directory)
	}
	// Every file is selected, so the priorities stay as rtorrent set them.
	if rt.Called("d.update_priorities") != 0 {
		t.Fatal("a full selection must not touch the priorities")
	}
}

func TestExternalDataURI(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, hash := test.WriteTorrent(t, "data.bin", map[string][]byte{"": make([]byte, 1000)})
	raw, err := os.ReadFile(torrentPath)
	if err != nil {
		t.Fatal(err)
	}
	f := buildExternalFetcher(t, tc)
	uri := "data:application/x-bittorrent;base64," + base64.StdEncoding.EncodeToString(raw)
	if err := f.Resolve(&base.Request{URL: uri}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if rt.Download(hash) == nil {
		t.Fatal("rtorrent did not get the torrent")
	}
}

func TestExternalMagnetGetsFolderAndFilesAfterMetadata(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	_, mi, hash := test.WriteTorrent(t, "show", map[string][]byte{"e01.mkv": make([]byte, 50000), "e02.mkv": make([]byte, 60000)})
	magnet := "magnet:?xt=urn:btih:" + strings.ToLower(hash) + "&dn=The+Show&tr=http%3A%2F%2Ftracker.example%2Fannounce"

	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: magnet}); err != nil {
		t.Fatal(err)
	}
	if res := f.Meta().Res; res.Name != "The Show" || len(res.Files) != 0 || res.Hash != strings.ToLower(hash) {
		t.Fatalf("resolved %+v", res)
	}
	dir := t.TempDir()
	f.Meta().Opts = &base.Options{Path: dir}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if d := rt.Download(hash); d == nil || !d.Meta || !d.Started || d.Directory != dir {
		t.Fatalf("metadata download %+v", d)
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- f.Wait() }()

	rt.FinishMetadata(hash, mi)
	eventually(t, "the real download to start in the task folder", func() bool {
		d := rt.Download(hash)
		return d != nil && !d.Meta && d.Started && d.Directory == dir
	})
	eventually(t, "the files from rtorrent", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return len(f.meta.Res.Files) == 2
	})
	f.mu.Lock()
	res, selected := f.meta.Res, append([]int(nil), f.meta.Opts.SelectFiles...)
	f.mu.Unlock()
	if res.Name != "show" || res.Size != 110000 || res.Files[0].Name != "e01.mkv" || len(selected) != 2 {
		t.Fatalf("resource after metadata %+v, selected %v", res, selected)
	}

	rt.Complete(hash)
	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return after the download was complete")
	}
	assertNoBuiltInClient(t)
}

func TestExternalPauseContinueAndDelete(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, hash := test.WriteTorrent(t, "p.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if d := rt.Download(hash); d.Started {
		t.Fatal("pause must stop the download in rtorrent")
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if rt.Called("load.raw") != 1 {
		t.Fatal("continue must not load the torrent again")
	}
	if d := rt.Download(hash); !d.Started {
		t.Fatal("continue must start the download in rtorrent")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if rt.Download(hash) != nil {
		t.Fatal("deleting the task must remove the download from rtorrent")
	}
}

func TestExternalDeleteWithRtorrentUnreachable(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, _ := test.WriteTorrent(t, "u.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	c := *tc.cfg.Load()
	c.External.URL = "scgi://127.0.0.1:1"
	tc.cfg.Store(&c)
	if err := f.Close(); err != nil {
		t.Fatalf("an unreachable rtorrent must not keep the task: %v", err)
	}
}

func TestExternalTaskKeepsItsEngine(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, hash := test.WriteTorrent(t, "k.bin", map[string][]byte{"": make([]byte, 3000)})
	fm := new(FetcherManager)
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}

	// Save and restore through JSON, the way the downloader does.
	data, err := fm.Store(f)
	if err != nil {
		t.Fatal(err)
	}
	blob, _ := json.Marshal(data)
	metaBlob, _ := json.Marshal(f.Meta())
	restore := func() *Fetcher {
		v, build := fm.Restore()
		if err := json.Unmarshal(blob, v); err != nil {
			t.Fatal(err)
		}
		var meta fetcher.FetcherMeta
		if err := json.Unmarshal(metaBlob, &meta); err != nil {
			t.Fatal(err)
		}
		r := build(&meta, v).(*Fetcher)
		r.Setup(tc.controller())
		return r
	}

	tc.setEnable(false)
	r := restore()
	if !r.isExternal() {
		t.Fatal("a restored task must stay with rtorrent")
	}
	if err := r.Start(); !errors.Is(err, errExternalOff) {
		t.Fatalf("start with the hand-off off: %v, want errExternalOff", err)
	}

	tc.setEnable(true)
	r = restore()
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	if rt.Called("load.raw") != 1 || !rt.Download(hash).Started {
		t.Fatal("a restored task must start its rtorrent download again, not load it twice")
	}

	// A task of the built-in engine stays there when the switch is on.
	builtIn := &fetcherData{Progress: fetcher.Progress{0}}
	_, build := fm.Restore()
	restored := build(&fetcher.FetcherMeta{Req: &base.Request{URL: torrentPath}}, builtIn).(*Fetcher)
	restored.Setup(tc.controller())
	if restored.isExternal() {
		t.Fatal("a built-in task must not move to rtorrent")
	}
}

func TestExternalRefusesTorrentRtorrentAlreadyHas(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, mi, _ := test.WriteTorrent(t, "dup.bin", map[string][]byte{"": make([]byte, 3000)})
	rt.Add(mi, true)
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err == nil || !strings.Contains(err.Error(), "already has") {
		t.Fatalf("start: %v, want an error that rtorrent already has the torrent", err)
	}
}

func TestExternalWaitFailsWhenRtorrentLosesTheDownload(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, hash := test.WriteTorrent(t, "gone.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	rt.Remove(hash)
	if err := waitExternal(t, f); !errors.Is(err, errRtorrentGone) {
		t.Fatalf("Wait: %v, want errRtorrentGone", err)
	}
}

func TestExternalOverHTTPWithBasicAuth(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeHTTPEndpoint(t, "rt", "s3cret"), "")
	c := *tc.cfg.Load()
	c.External.Username, c.External.Password = "rt", "s3cret"
	tc.cfg.Store(&c)
	torrentPath, _, hash := test.WriteTorrent(t, "h.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if !rt.Download(hash).Started {
		t.Fatal("not started over HTTP")
	}

	c.External.Password = "wrong"
	tc.cfg.Store(&c)
	f2 := buildExternalFetcher(t, tc)
	path2, _, _ := test.WriteTorrent(t, "h2.bin", map[string][]byte{"": make([]byte, 3000)})
	if err := f2.Resolve(&base.Request{URL: path2}); err != nil {
		t.Fatal(err)
	}
	f2.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f2.Start(); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("start with a wrong password: %v, want HTTP 401", err)
	}
}

func TestExternalOverUnixSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "rt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "rpc.sock")
	if len(sock) > 100 {
		t.Skip("the temp folder path is too long for a Unix socket")
	}
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGIUnix(t, sock), "")
	torrentPath, _, hash := test.WriteTorrent(t, "s.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if !rt.Download(hash).Started {
		t.Fatal("not started over the Unix socket")
	}
}

func TestExternalStats(t *testing.T) {
	rt := test.NewFakeRtorrent(t)
	tc := newExternalTestConfig(rt.ServeSCGI(t), "")
	torrentPath, _, hash := test.WriteTorrent(t, "st.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	rt.SetUpTotal(hash, 1234)
	if _, err := f.externalPoll(); err != nil {
		t.Fatal(err)
	}
	stats := f.Stats()
	if s, ok := stats.Snapshot.(*bt.StatsSnapshot); !ok || s.SeedBytes != 1234 {
		t.Fatalf("snapshot %#v", stats.Snapshot)
	}
	if f.UploadedBytes() != 1234 {
		t.Fatalf("uploaded %d", f.UploadedBytes())
	}
	if err := f.WaitUpload(); err != nil {
		t.Fatal(err)
	}
}

func TestExternalRefusesMagnetWithoutV1Hash(t *testing.T) {
	tc := newExternalTestConfig("scgi://127.0.0.1:1", "")
	f := buildExternalFetcher(t, tc)
	err := f.Resolve(&base.Request{URL: "magnet:?xt=urn:btmh:1220caf1e1c30e81cb361b9ee167c4aa64228a7fa4fa9f6105232b28ad099f3a302e"})
	if err == nil || !strings.Contains(err.Error(), "v1 info hash") {
		t.Fatalf("resolve: %v, want an error about the v1 info hash", err)
	}
}

func TestExternalBase32Magnet(t *testing.T) {
	tc := newExternalTestConfig("scgi://127.0.0.1:1", "")
	f := buildExternalFetcher(t, tc)
	// The base32 form of 0123456789ABCDEF0123456789ABCDEF01234567.
	if err := f.Resolve(&base.Request{URL: "magnet:?xt=urn:btih:AERUKZ4JVPG66AJDIVTYTK6N54ASGRLH"}); err != nil {
		t.Fatal(err)
	}
	if got := f.data.External.Hash; got != "0123456789ABCDEF0123456789ABCDEF01234567" {
		t.Fatalf("hash %s", got)
	}
	if f.Meta().Res.Name != "0123456789ABCDEF0123456789ABCDEF01234567" {
		t.Fatalf("name %q, want the info hash", f.Meta().Res.Name)
	}
}
