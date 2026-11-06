package download

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// setupRtorrentDownloader returns a downloader whose BitTorrent tasks go to
// the fake rtorrent.
func setupRtorrentDownloader(t *testing.T) (*Downloader, *test.FakeRtorrent, string) {
	t.Helper()
	rt := test.NewFakeRtorrent(t)
	endpoint := rt.ServeSCGI(t)
	d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		d.Delete(nil, true)
		d.Clear()
	})
	cfg, err := d.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	downloadDir := t.TempDir()
	cfg.DownloadDir = downloadDir
	raw, _ := json.Marshal(cfg.ProtocolConfig["bt"])
	var bt map[string]any
	if err := json.Unmarshal(raw, &bt); err != nil {
		t.Fatal(err)
	}
	bt["external"] = map[string]any{"enable": true, "url": endpoint}
	cfg.ProtocolConfig["bt"] = bt
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
	return d, rt, downloadDir
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRtorrentHandoffMagnetCreateDirect(t *testing.T) {
	d, rt, downloadDir := setupRtorrentDownloader(t)
	_, mi, hash := test.WriteTorrent(t, "season", map[string][]byte{"e01.mkv": make([]byte, 50000), "e02.mkv": make([]byte, 70000)})
	magnet := "magnet:?xt=urn:btih:" + strings.ToLower(hash) + "&dn=Season+One"

	id, err := d.CreateDirect(&base.Request{URL: magnet}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the metadata download in rtorrent", func() bool {
		dl := rt.Download(hash)
		return dl != nil && dl.Meta && dl.Started
	})
	if dl := rt.Download(hash); dl.Directory != downloadDir {
		t.Fatalf("rtorrent directory %q, want %q", dl.Directory, downloadDir)
	}
	task := d.GetTask(id)
	if task.Meta.Res.Name != "Season One" {
		t.Fatalf("name before metadata %q", task.Meta.Res.Name)
	}

	rt.FinishMetadata(hash, mi)
	// The fetcher starts the real download after it has taken the files.
	waitFor(t, "the real download to start in the download folder", func() bool {
		dl := rt.Download(hash)
		return dl != nil && !dl.Meta && dl.Started && dl.Directory == downloadDir
	})

	rt.Complete(hash)
	waitTaskStatus(t, d, task, base.DownloadStatusDone)

	// The name, size and files from rtorrent are saved with the task.
	var stored Task
	if ok, err := d.storage.Get(bucketTask, id, &stored); err != nil || !ok {
		t.Fatalf("stored task: %v %v", ok, err)
	}
	if stored.Meta.Res.Name != "season" || stored.Meta.Res.Size != 120000 || len(stored.Meta.Res.Files) != 2 {
		t.Fatalf("stored resource %+v", stored.Meta.Res)
	}
	if len(stored.Meta.Opts.SelectFiles) != 2 {
		t.Fatalf("stored selection %v", stored.Meta.Opts.SelectFiles)
	}
}

func TestRtorrentHandoffResolveThenCreateWithSelection(t *testing.T) {
	d, rt, downloadDir := setupRtorrentDownloader(t)
	torrentPath, _, hash := test.WriteTorrent(t, "pack", map[string][]byte{"a.bin": make([]byte, 30000), "b.bin": make([]byte, 30000)})

	rr, err := d.Resolve(&base.Request{URL: torrentPath})
	if err != nil {
		t.Fatal(err)
	}
	if rr.Res.Name != "pack" || len(rr.Res.Files) != 2 {
		t.Fatalf("resolved %+v", rr.Res)
	}
	if rt.Called("load.raw") != 0 {
		t.Fatal("resolve must not hand the torrent to rtorrent")
	}
	id, err := d.CreateWithOptions(rr.ID, &base.Options{Path: downloadDir, SelectFiles: []int{1}})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the download in rtorrent", func() bool {
		dl := rt.Download(hash)
		return dl != nil && dl.Started
	})
	dl := rt.Download(hash)
	if dl.Files[0].Priority != 0 || dl.Files[1].Priority != 1 {
		t.Fatalf("priorities %d %d, want 0 1", dl.Files[0].Priority, dl.Files[1].Priority)
	}
	rt.CompleteFile(hash, 1)
	waitTaskStatus(t, d, d.GetTask(id), base.DownloadStatusDone)
}

func TestRtorrentHandoffDeleteBeforeMetadata(t *testing.T) {
	d, rt, downloadDir := setupRtorrentDownloader(t)
	hash := "0123456789ABCDEF0123456789ABCDEF01234567"
	id, err := d.CreateDirect(&base.Request{URL: "magnet:?xt=urn:btih:" + hash + "&dn=Never+Seen"}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the metadata download in rtorrent", func() bool { return rt.Download(hash) != nil })
	folder := filepath.Join(downloadDir, "Never Seen")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := d.Delete(&TaskFilter{IDs: []string{id}}, true); err != nil {
		t.Fatal(err)
	}
	if rt.Download(hash) != nil {
		t.Fatal("deleting the task must remove the download from rtorrent")
	}
	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		t.Fatalf("delete with files must remove %s: %v", folder, err)
	}
	if d.GetTask(id) != nil {
		t.Fatal("the task is still listed")
	}
}
