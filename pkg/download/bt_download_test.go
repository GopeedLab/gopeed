package download

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	protocolbt "github.com/GopeedLab/gopeed/internal/protocol/bt"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"golang.org/x/time/rate"
)

var btTestEvents sync.Map // *Downloader -> chan *Event

// These tests exercise the actual Downloader, BT Fetcher, storage and peer
// protocol. All content is generated locally and served by a loopback seeder.
func btDownloader(t *testing.T, store Storage) *Downloader {
	t.Helper()
	d := NewDownloader(&DownloaderConfig{FetchManagers: []fetcher.FetcherManager{&protocolbt.FetcherManager{}}, Storage: store, RefreshInterval: 20})
	events := make(chan *Event, 8)
	btTestEvents.Store(d, events)
	d.Listener(func(e *Event) {
		if e.Key == EventKeyDone || e.Key == EventKeyError {
			select {
			case events <- e:
			default:
			}
		}
	})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer btTestEvents.Delete(d)
		if !d.closed.Load() {
			if err := d.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	return d
}

func btAwaitDone(t *testing.T, d *Downloader, start func() (string, error)) string {
	t.Helper()
	events, ok := btTestEvents.Load(d)
	if !ok {
		t.Fatal("BT test event listener was not installed")
	}
	done := events.(chan *Event)
	id, err := start()
	if err != nil {
		t.Fatal(err)
	}
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case e := <-done:
			if e.Task.ID != id {
				continue
			}
			if e.Err != nil {
				t.Fatalf("BT download failed: %v", e.Err)
			}
			return id
		case <-timeout.C:
			task := d.GetTask(id)
			task.lock.Lock()
			progress := task.fetcher.Progress()
			stats := task.fetcher.Stats()
			task.lock.Unlock()
			t.Fatalf("BT task %s did not finish: progress=%v stats=%+v", id, progress, stats)
		}
	}
}

func TestBTDownloadSources(t *testing.T) {
	for _, kind := range []string{"magnet", "torrent", "file URI", "data URI"} {
		t.Run(kind, func(t *testing.T) {
			fixture := test.NewBTFixture(t, map[string][]byte{"": test.BTPayload(96*1024, kind)}, 0)
			source := fixture.Magnet
			switch kind {
			case "torrent":
				source = fixture.TorrentFile
			case "file URI":
				source = fixture.FileURI()
			case "data URI":
				source = fixture.DataURI
			}
			d := btDownloader(t, nil)
			dir := filepath.Join(t.TempDir(), "chosen-after-resolve")
			rr, err := d.Resolve(&base.Request{URL: source})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("Resolve created output directory: %v", err)
			}
			if d.fetcherCache[rr.ID].Meta().Opts != nil {
				t.Fatal("Resolve assigned download options")
			}
			btAwaitDone(t, d, func() (string, error) { return d.CreateWithOptions(rr.ID, &base.Options{Path: dir}) })
			fixture.AssertFiles(t, dir)
		})
	}
}

func TestBTDownloadSelectedFiles(t *testing.T) {
	fixture := test.NewBTFixture(t, map[string][]byte{
		"a.bin": test.BTPayload(20*1024, "a"), "nested/b.bin": test.BTPayload(48*1024, "b"),
	}, 0)
	d := btDownloader(t, nil)
	dir := t.TempDir()
	rr, err := d.Resolve(&base.Request{URL: fixture.Magnet})
	if err != nil {
		t.Fatal(err)
	}
	selected := -1
	for i, f := range rr.Res.Files {
		if f.Name == "b.bin" {
			selected = i
		}
	}
	if selected < 0 {
		t.Fatal("selected file absent from metadata")
	}
	id := btAwaitDone(t, d, func() (string, error) {
		return d.CreateWithOptions(rr.ID, &base.Options{Path: dir, SelectFiles: []int{selected}})
	})
	fixture.AssertFiles(t, dir, filepath.Join("bundle", "nested", "b.bin"))
	if _, err := os.Stat(filepath.Join(dir, "bundle", "a.bin")); !os.IsNotExist(err) {
		t.Fatalf("unselected file was downloaded: %v", err)
	}
	if d.GetTask(id).Meta.Res.Size != 48*1024 {
		t.Fatal("selected resource size is incorrect")
	}
}

func btAwaitPartial(t *testing.T, d *Downloader, id string) int64 {
	t.Helper()
	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			task := d.GetTask(id)
			task.lock.Lock()
			f := task.fetcher
			progress := f.Progress().TotalDownloaded()
			size := task.Meta.Res.Size
			task.lock.Unlock()
			if progress >= 32*1024 && progress < size {
				return progress
			}
		case <-timeout.C:
			t.Fatal("BT transfer did not reach partial progress")
		}
	}
}

func TestBTPauseAndResume(t *testing.T) {
	fixture := test.NewBTFixture(t, map[string][]byte{"": test.BTPayload(512*1024, "pause")}, 64*1024)
	d := btDownloader(t, nil)
	dir := t.TempDir()
	id, err := d.CreateDirect(&base.Request{URL: fixture.Magnet}, &base.Options{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	btAwaitPartial(t, d, id)
	if err := d.Pause(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatal(err)
	}
	task := d.GetTask(id)
	task.statusLock.Lock()
	paused := task.Status == base.DownloadStatusPause
	task.statusLock.Unlock()
	if !paused {
		t.Fatal("task did not pause")
	}
	fixture.Limiter.SetLimit(rate.Inf)
	btAwaitDone(t, d, func() (string, error) { return id, d.Continue(&TaskFilter{IDs: []string{id}}) })
	fixture.AssertFiles(t, dir)
}

func TestBTColdStartResume(t *testing.T) {
	t.Run("v1", func(t *testing.T) { testBTColdStartResume(t, false) })
	t.Run("hybrid", func(t *testing.T) { testBTColdStartResume(t, true) })
}

func testBTColdStartResume(t *testing.T, v2 bool) {
	var fixture *test.BTFixture
	if v2 {
		fixture = test.NewBTHybridFixture(t, test.BTPayload(512*1024, "restart-hybrid"))
		fixture.Limiter.SetLimit(64 * 1024)
	} else {
		fixture = test.NewBTFixture(t, map[string][]byte{"": test.BTPayload(512*1024, "restart")}, 64*1024)
	}
	database := t.TempDir()
	dir := t.TempDir()
	d := btDownloader(t, NewBoltStorage(database))
	id, err := d.CreateDirect(&base.Request{URL: fixture.Magnet}, &base.Options{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	btAwaitPartial(t, d, id)
	if err := d.Pause(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatal(err)
	}
	// Verify the fetcher checkpoint contains the metainfo required for restart.
	task := d.GetTask(id)
	task.lock.Lock()
	data, err := task.fetcherManager.Store(task.fetcher)
	if err != nil {
		task.lock.Unlock()
		t.Fatal(err)
	}
	encoded, err := json.Marshal(data)
	task.lock.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte("metaInfo")) {
		t.Fatal("metainfo was not persisted")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	seedStats := fixture.Torrent.Stats()
	sentBeforeRestart := seedStats.BytesWrittenData.Int64()
	d2 := btDownloader(t, NewBoltStorage(database))
	restored := d2.GetTask(id)
	if restored == nil {
		t.Fatal("task was not restored")
	}
	if restored.Meta.Opts.Path != dir {
		t.Fatal("download directory was lost")
	}
	fixture.Limiter.SetLimit(rate.Inf)
	btAwaitDone(t, d2, func() (string, error) { return id, d2.Continue(&TaskFilter{IDs: []string{id}}) })
	fixture.AssertFiles(t, dir)
	seedStats = fixture.Torrent.Stats()
	if sent := seedStats.BytesWrittenData.Int64() - sentBeforeRestart; sent >= 512*1024 {
		t.Fatalf("restart redownloaded the entire file: %d bytes", sent)
	}
}

func TestBTCompletedTaskRestoresWithoutSourceFile(t *testing.T) {
	fixture := test.NewBTFixture(t, map[string][]byte{"": test.BTPayload(64*1024, "offline")}, 0)
	database := t.TempDir()
	dir := t.TempDir()
	d := btDownloader(t, NewBoltStorage(database))
	id := btAwaitDone(t, d, func() (string, error) {
		return d.CreateDirect(&base.Request{URL: fixture.TorrentFile}, &base.Options{Path: dir})
	})
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(fixture.TorrentFile); err != nil {
		t.Fatal(err)
	}
	d2 := btDownloader(t, NewBoltStorage(database))
	task := d2.GetTask(id)
	if task == nil {
		t.Fatal("completed task missing")
	}
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		task.lock.Lock()
		ready := task.fetcher != nil && task.fetcher.Stats().Snapshot != nil
		task.lock.Unlock()
		if ready {
			break
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			t.Fatal("completed BT task did not restore from cached metainfo")
		}
	}
	fixture.AssertFiles(t, dir)
}

func TestBTDownloadHybrid(t *testing.T) {
	fixture := test.NewBTHybridFixture(t, test.BTPayload(80*1024, "hybrid-v2"))
	d := btDownloader(t, nil)
	dir := t.TempDir()
	btAwaitDone(t, d, func() (string, error) {
		return d.CreateDirect(&base.Request{URL: fixture.Magnet}, &base.Options{Path: dir})
	})
	fixture.AssertFiles(t, dir)
}

func TestBTRepairCorruptedExistingFile(t *testing.T) {
	fixture := test.NewBTFixture(t, map[string][]byte{"": test.BTPayload(128*1024, "repair")}, 0)
	dir := t.TempDir()
	want := fixture.Files["payload.bin"]
	damaged := append([]byte(nil), want...)
	damaged[20000] ^= 0xff
	if err := os.WriteFile(filepath.Join(dir, "payload.bin"), damaged, 0600); err != nil {
		t.Fatal(err)
	}
	d := btDownloader(t, nil)
	btAwaitDone(t, d, func() (string, error) {
		return d.CreateDirect(&base.Request{URL: fixture.Magnet}, &base.Options{Path: dir})
	})
	fixture.AssertFiles(t, dir)
}

func TestBTPatchSelectionWhilePaused(t *testing.T) {
	fixture := test.NewBTFixture(t, map[string][]byte{"a.bin": test.BTPayload(256*1024, "patch-a"), "b.bin": test.BTPayload(256*1024, "patch-b")}, 64*1024)
	d := btDownloader(t, nil)
	dir := t.TempDir()
	rr, err := d.Resolve(&base.Request{URL: fixture.Magnet})
	if err != nil {
		t.Fatal(err)
	}
	id, err := d.CreateWithOptions(rr.ID, &base.Options{Path: dir, SelectFiles: []int{0}})
	if err != nil {
		t.Fatal(err)
	}
	btAwaitPartial(t, d, id)
	if err := d.Pause(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatal(err)
	}
	if err := d.Patch(id, nil, &base.Options{SelectFiles: []int{1}}); err != nil {
		t.Fatal(err)
	}
	fixture.Limiter.SetLimit(rate.Inf)
	btAwaitDone(t, d, func() (string, error) { return id, d.Continue(&TaskFilter{IDs: []string{id}}) })
	fixture.AssertFiles(t, dir, filepath.Join("bundle", "b.bin"))
	for _, suffix := range []string{"", ".part"} {
		if _, err := os.Stat(filepath.Join(dir, "bundle", "a.bin") + suffix); !os.IsNotExist(err) {
			t.Fatalf("unselected output remains: %v", err)
		}
	}
}

func TestBTDownloadMultiFileWithEmptyFile(t *testing.T) {
	fixture := test.NewBTFixture(t, map[string][]byte{"empty.txt": {}, "nested/data.bin": test.BTPayload(48*1024, "empty-file")}, 0)
	d := btDownloader(t, nil)
	dir := t.TempDir()
	btAwaitDone(t, d, func() (string, error) {
		return d.CreateDirect(&base.Request{URL: fixture.Magnet}, &base.Options{Path: dir})
	})
	fixture.AssertFiles(t, dir)
}
