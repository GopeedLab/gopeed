package download

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

// TestDownloader_HlsLiveSite is a manual acceptance gate against a real site.
// It is skipped unless HLS_LIVE_URL is set, e.g.:
//
//	HLS_LIVE_URL="https://host/path/index.m3u8" go test ./pkg/download/ -run TestDownloader_HlsLiveSite -v
func TestDownloader_HlsLiveSite(t *testing.T) {
	playlistURL := os.Getenv("HLS_LIVE_URL")
	if playlistURL == "" {
		t.Skip("set HLS_LIVE_URL to run the live-site acceptance test")
	}
	referer := os.Getenv("HLS_LIVE_REFERER")

	downloader := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloadDir := t.TempDir()
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DownloadDir = downloadDir
	if err := downloader.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}

	req := &base.Request{URL: playlistURL}
	if referer != "" {
		req.Extra = map[string]any{"header": map[string]string{"Referer": referer}}
	}
	rr, err := downloader.Resolve(req, nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	name := rr.Res.Files[0].Name
	size := rr.Res.Size
	t.Logf("resolved: name=%s size=%d", name, size)

	taskID, err := downloader.Create(rr.ID)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Minute)
	var lastLogged int64
	for {
		task := downloader.GetTask(taskID)
		if task == nil {
			t.Fatal("task not found")
		}
		if task.Status == base.DownloadStatusDone {
			break
		}
		if task.Status == base.DownloadStatusError {
			t.Fatalf("task entered error state after %d bytes", task.Progress.Downloaded)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout, downloaded %d bytes", task.Progress.Downloaded)
		}
		if tick := task.Progress.Downloaded; tick-lastLogged > 10<<20 {
			lastLogged = tick
			t.Logf("progress: %d bytes", tick)
		}
		time.Sleep(500 * time.Millisecond)
	}

	data, err := os.ReadFile(filepath.Join(downloadDir, name))
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) == 0 {
		t.Fatal("merged file is empty")
	}
	t.Logf("merged: %s (%d bytes)", name, len(data))
	if filepath.Ext(name) == ".ts" {
		// MPEG-TS packets are 188 bytes and start with the 0x47 sync byte.
		for i := 0; i < 188*4 && i < len(data); i += 188 {
			if data[i] != 0x47 {
				t.Fatalf("invalid TS sync byte at offset %d: %#x", i, data[i])
			}
		}
	} else if string(data[4:8]) != "ftyp" {
		t.Fatalf("invalid fMP4 header: %q", data[:8])
	}
	fmt.Printf("LIVE ACCEPTANCE OK: %s (%d bytes)\n", filepath.Join(downloadDir, name), len(data))
}
