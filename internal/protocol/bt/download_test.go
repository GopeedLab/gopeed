package bt

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	btprotocol "github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/anacrolix/torrent"
	pp "github.com/anacrolix/torrent/peer_protocol"
	"golang.org/x/time/rate"
)

type btDownloadHarness struct {
	*Fetcher
	pendingRequests atomic.Int64
}

// These tests exercise the BT fetcher, native storage and real peer protocol.
// All content is generated locally and served by a loopback seeder.
func btDownloadFetcher(t *testing.T, f fetcher.Fetcher) *btDownloadHarness {
	t.Helper()
	if f == nil {
		f = new(FetcherManager).Build()
	}
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		encoded, err := json.Marshal(new(FetcherManager).DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, v); err != nil {
			t.Fatal(err)
		}
	}
	f.Setup(ctl)
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
		if err := closeClient(); err != nil {
			t.Error(err)
		}
	})
	h := &btDownloadHarness{Fetcher: f.(*Fetcher)}
	// Keep the test client on loopback and observe real cancellation replies.
	// Waiting for them avoids the pinned library's immediate cancel/resume
	// request scheduling bug, without relying on arbitrary sleep durations.
	cfg = torrent.TestingConfig(t)
	cfg.DisableTrackers = false
	cfg.Seed = true
	cfg.DisableUTP = true
	cfg.DisableIPv6 = true
	cfg.DisableWebtorrent = true
	cfg.KeepAliveTimeout = time.Minute
	cfg.MaxAllocPeerRequestDataPerConn = 1 << 20
	cfg.Callbacks.SentRequest = []func(torrent.PeerRequestEvent){func(torrent.PeerRequestEvent) {
		h.pendingRequests.Add(1)
	}}
	cfg.Callbacks.ReadMessage = func(_ *torrent.PeerConn, msg *pp.Message) {
		if !msg.Keepalive && (msg.Type == pp.Piece || msg.Type == pp.Reject) {
			h.pendingRequests.Add(-1)
		}
	}
	var err error
	client, err = torrent.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func btResolve(t *testing.T, f *btDownloadHarness, source string) {
	t.Helper()
	if err := f.Resolve(&base.Request{URL: source}); err != nil {
		t.Fatal(err)
	}
}

func btStart(t *testing.T, f *btDownloadHarness) {
	t.Helper()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
}

func btAwaitDone(t *testing.T, f *btDownloadHarness) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- f.Wait() }()
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("BT download failed: %v", err)
		}
		if !f.isDone() {
			t.Fatal("Wait returned before selected files were verified")
		}
	case <-timeout.C:
		var status strings.Builder
		client.WriteStatus(&status)
		t.Log(status.String())
		t.Fatalf("BT download did not finish: progress=%v stats=%+v pieces=%+v peers=%+v", f.Progress(), f.torrent.Stats(), f.torrent.PieceStateRuns(), f.Stats().Runtime)
	}
}

func btAwaitPartial(t *testing.T, f *btDownloadHarness) {
	t.Helper()
	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			progress := f.Progress().TotalDownloaded()
			if progress >= 32*1024 && progress < f.Meta().Res.Size {
				return
			}
		case <-timeout.C:
			t.Fatal("BT transfer did not reach partial progress")
		}
	}
}

func btPause(t *testing.T, f *btDownloadHarness) {
	t.Helper()
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if f.pendingRequests.Load() == 0 {
			// The message callback runs under the client lock. Acquiring that
			// lock ensures the final cancellation reply has finished processing.
			client.Stats()
			return
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			t.Fatal("peer did not acknowledge cancelled requests")
		}
	}
}

func TestBTDownloadSources(t *testing.T) {
	for _, kind := range []string{"magnet", "torrent", "file URI", "data URI"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newBTFixture(t, map[string][]byte{"": btPayload(96*1024, kind)}, 0)
			source := fixture.Magnet
			switch kind {
			case "torrent":
				source = fixture.TorrentFile
			case "file URI":
				source = fixture.FileURI()
			case "data URI":
				source = fixture.DataURI
			}
			f := btDownloadFetcher(t, nil)
			btResolve(t, f, source)
			if f.Meta().Opts != nil {
				t.Fatal("Resolve assigned download options")
			}
			f.storage.mu.RLock()
			bound := f.storage.owner != nil
			f.storage.mu.RUnlock()
			if bound {
				t.Fatal("Resolve bound download storage")
			}
			resolvedTorrent := f.torrent
			dir := filepath.Join(t.TempDir(), "chosen-after-resolve")
			f.Meta().Opts = &base.Options{Path: dir}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("output directory exists before Start: %v", err)
			}
			btStart(t, f)
			if f.torrent != resolvedTorrent {
				t.Fatal("Start replaced the resolved torrent")
			}
			btAwaitDone(t, f)
			fixture.AssertFiles(t, dir)
		})
	}
}

func TestBTDirectDownloadWithExtraTrackers(t *testing.T) {
	for _, source := range []string{"request", "config", "subscription"} {
		t.Run(source, func(t *testing.T) {
			fixture := makeBTFixture(t, map[string][]byte{
				"a.bin":        btPayload(20*1024, "direct-a"),
				"nested/b.bin": btPayload(48*1024, "direct-b"),
			}, 0, false, false)
			magnet, err := url.Parse(fixture.Magnet)
			if err != nil {
				t.Fatal(err)
			}
			query := magnet.Query()
			query.Del("tr")
			query.Del("x.pe")
			magnet.RawQuery = query.Encode()
			req := &base.Request{URL: magnet.String()}
			f := btDownloadFetcher(t, nil)
			if source == "request" {
				req.Extra = &btprotocol.ReqExtra{Trackers: []string{fixture.TrackerURL}}
			} else {
				backendConfig := new(FetcherManager).DefaultConfig().(*config)
				if source == "config" {
					backendConfig.CustomTrackers = []string{fixture.TrackerURL}
				} else {
					backendConfig.TrackerSubscribeURLs = []string{"https://subscription.example/list"}
					backendConfig.State.TrackerSources[backendConfig.TrackerSubscribeURLs[0]] = []string{fixture.TrackerURL}
				}
				f.ctl.GetConfig = func(dst any) {
					raw, err := json.Marshal(backendConfig)
					if err != nil {
						t.Error(err)
						return
					}
					if err := json.Unmarshal(raw, dst); err != nil {
						t.Error(err)
					}
				}
			}
			// CreateDirect supplies options before Downloader resolves the task.
			dir := filepath.Join(t.TempDir(), "direct-download")
			opts := &base.Options{Path: dir, SelectFiles: []int{}}
			f.Meta().Opts = opts
			resolved := make(chan error, 1)
			go func() { resolved <- f.Resolve(req) }()
			select {
			case err := <-resolved:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				f.torrentDropFunc()
				<-resolved
				t.Fatal("direct magnet download did not obtain metadata from its extra tracker")
			}
			if f.Meta().Opts != opts || len(opts.SelectFiles) != len(fixture.Files) {
				t.Fatal("resolution lost direct download options or did not select all files")
			}
			btStart(t, f)
			btAwaitDone(t, f)
			fixture.AssertFiles(t, dir)
		})
	}
}

func TestBTDownloadSelectedFiles(t *testing.T) {
	fixture := newBTFixture(t, map[string][]byte{
		"a.bin": btPayload(20*1024, "a"), "nested/b.bin": btPayload(48*1024, "b"),
	}, 0)
	f := btDownloadFetcher(t, nil)
	dir := t.TempDir()
	btResolve(t, f, fixture.Magnet)
	selected := -1
	for i, file := range f.Meta().Res.Files {
		if file.Name == "b.bin" {
			selected = i
		}
	}
	if selected < 0 {
		t.Fatal("selected file absent from metadata")
	}
	f.Meta().Opts = &base.Options{Path: dir}
	if err := f.Patch(nil, &base.Options{SelectFiles: []int{selected}}); err != nil {
		t.Fatal(err)
	}
	btStart(t, f)
	btAwaitDone(t, f)
	fixture.AssertFiles(t, dir, filepath.Join("bundle", "nested", "b.bin"))
	if _, err := os.Stat(filepath.Join(dir, "bundle", "a.bin")); !os.IsNotExist(err) {
		t.Fatalf("unselected output remains: %v", err)
	}
	if f.Meta().Res.Size != 48*1024 {
		t.Fatal("selected resource size is incorrect")
	}
}

func TestBTPauseAndResume(t *testing.T) {
	fixture := newBTFixture(t, map[string][]byte{"": btPayload(512*1024, "pause")}, 64*1024)
	f := btDownloadFetcher(t, nil)
	dir := t.TempDir()
	btResolve(t, f, fixture.Magnet)
	f.Meta().Opts = &base.Options{Path: dir}
	btStart(t, f)
	btAwaitPartial(t, f)
	btPause(t, f)
	resolvedTorrent := f.torrent
	fixture.Limiter.SetLimit(rate.Inf)
	// Once cancelled requests have settled, no more data should arrive.
	stats := f.torrent.Stats()
	readWhilePaused := stats.BytesReadData.Int64()
	time.Sleep(200 * time.Millisecond)
	stats = f.torrent.Stats()
	if stats.BytesReadData.Int64() != readWhilePaused || f.isDone() {
		t.Fatal("data download continued while paused")
	}
	btStart(t, f)
	if f.torrent != resolvedTorrent {
		t.Fatal("resume replaced the resolved torrent")
	}
	btAwaitDone(t, f)
	fixture.AssertFiles(t, dir)
}

type btFetcherCheckpoint struct {
	Meta *fetcher.FetcherMeta `json:"meta"`
	Data json.RawMessage      `json:"data"`
}

func btStore(t *testing.T, f *btDownloadHarness) []byte {
	t.Helper()
	data, err := new(FetcherManager).Store(f.Fetcher)
	if err != nil {
		t.Fatal(err)
	}
	encodedData, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(btFetcherCheckpoint{Meta: f.Meta(), Data: encodedData})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func btRestore(t *testing.T, encoded []byte) *btDownloadHarness {
	t.Helper()
	var checkpoint btFetcherCheckpoint
	if err := json.Unmarshal(encoded, &checkpoint); err != nil {
		t.Fatal(err)
	}
	data, restore := new(FetcherManager).Restore()
	if err := json.Unmarshal(checkpoint.Data, data); err != nil {
		t.Fatal(err)
	}
	return btDownloadFetcher(t, restore(checkpoint.Meta, data))
}

func btCloseForRestart(t *testing.T, f *btDownloadHarness) {
	t.Helper()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := new(FetcherManager).Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBTColdStartResume(t *testing.T) {
	for _, kind := range []string{"v1", "hybrid"} {
		t.Run(kind, func(t *testing.T) {
			var fixture *btFixture
			if kind == "hybrid" {
				fixture = newBTHybridFixture(t, btPayload(512*1024, "restart-hybrid"))
				fixture.Limiter.SetLimit(64 * 1024)
			} else {
				fixture = newBTFixture(t, map[string][]byte{"": btPayload(512*1024, "restart")}, 64*1024)
			}
			f := btDownloadFetcher(t, nil)
			dir := t.TempDir()
			btResolve(t, f, fixture.Magnet)
			f.Meta().Opts = &base.Options{Path: dir}
			btStart(t, f)
			btAwaitPartial(t, f)
			btPause(t, f)
			checkpoint := btStore(t, f)
			hash := f.Meta().Res.Hash
			btCloseForRestart(t, f)
			seedStats := fixture.Torrent.Stats()
			sentBeforeRestart := seedStats.BytesWrittenData.Int64()
			restored := btRestore(t, checkpoint)
			if restored.Meta().Opts.Path != dir || restored.Meta().Res.Hash != hash {
				t.Fatal("download directory or hash was lost")
			}
			fixture.Limiter.SetLimit(rate.Inf)
			btStart(t, restored)
			btAwaitDone(t, restored)
			fixture.AssertFiles(t, dir)
			seedStats = fixture.Torrent.Stats()
			if sent := seedStats.BytesWrittenData.Int64() - sentBeforeRestart; sent >= 512*1024 {
				t.Fatalf("restart redownloaded the entire file: %d bytes", sent)
			}
		})
	}
}

func TestBTDownloadHybrid(t *testing.T) {
	fixture := newBTHybridFixture(t, btPayload(80*1024, "hybrid-v2"))
	f := btDownloadFetcher(t, nil)
	dir := t.TempDir()
	btResolve(t, f, fixture.Magnet)
	f.Meta().Opts = &base.Options{Path: dir}
	btStart(t, f)
	btAwaitDone(t, f)
	fixture.AssertFiles(t, dir)
}

func TestBTRepairCorruptedExistingFile(t *testing.T) {
	fixture := newBTFixture(t, map[string][]byte{"": btPayload(128*1024, "repair")}, 0)
	dir := t.TempDir()
	damaged := append([]byte(nil), fixture.Files["payload.bin"]...)
	damaged[20000] ^= 0xff
	if err := os.WriteFile(filepath.Join(dir, "payload.bin"), damaged, 0600); err != nil {
		t.Fatal(err)
	}
	f := btDownloadFetcher(t, nil)
	btResolve(t, f, fixture.Magnet)
	f.Meta().Opts = &base.Options{Path: dir}
	btStart(t, f)
	btAwaitDone(t, f)
	fixture.AssertFiles(t, dir)
}

func TestBTPatchSelectionWhilePaused(t *testing.T) {
	fixture := newBTFixture(t, map[string][]byte{"a.bin": btPayload(256*1024, "patch-a"), "b.bin": btPayload(256*1024, "patch-b")}, 64*1024)
	f := btDownloadFetcher(t, nil)
	dir := t.TempDir()
	btResolve(t, f, fixture.Magnet)
	f.Meta().Opts = &base.Options{Path: dir, SelectFiles: []int{0}}
	btStart(t, f)
	btAwaitPartial(t, f)
	btPause(t, f)
	if err := f.Patch(nil, &base.Options{SelectFiles: []int{1}}); err != nil {
		t.Fatal(err)
	}
	fixture.Limiter.SetLimit(rate.Inf)
	btStart(t, f)
	btAwaitDone(t, f)
	fixture.AssertFiles(t, dir, filepath.Join("bundle", "b.bin"))
	for _, suffix := range []string{"", ".part"} {
		if _, err := os.Stat(filepath.Join(dir, "bundle", "a.bin") + suffix); !os.IsNotExist(err) {
			t.Fatalf("unselected output remains: %v", err)
		}
	}
}

func TestBTDownloadMultiFileWithEmptyFile(t *testing.T) {
	fixture := newBTFixture(t, map[string][]byte{"empty.txt": {}, "nested/data.bin": btPayload(48*1024, "empty-file")}, 0)
	f := btDownloadFetcher(t, nil)
	dir := t.TempDir()
	btResolve(t, f, fixture.Magnet)
	f.Meta().Opts = &base.Options{Path: dir}
	btStart(t, f)
	btAwaitDone(t, f)
	fixture.AssertFiles(t, dir)
}
