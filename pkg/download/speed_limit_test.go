package download

import (
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"

	"golang.org/x/time/rate"
)

func TestDownloader_SpeedLimitConfig(t *testing.T) {
	dir := t.TempDir()
	d := NewDownloader(&DownloaderConfig{Storage: NewBoltStorage(dir), StorageDir: dir})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	if d.speedLimits.Download.Limit() != rate.Inf || d.speedLimits.Upload.Limit() != rate.Inf {
		t.Fatal("new downloader is not unlimited")
	}
	cfg, _ := d.GetConfig()
	next := *cfg
	next.DownloadSpeedLimit = 1 << 20
	next.UploadSpeedLimit = 256 << 10
	if err := d.PutConfig(&next); err != nil {
		t.Fatal(err)
	}
	if d.speedLimits.Download.Limit() != rate.Limit(1<<20) || d.speedLimits.Upload.Limit() != rate.Limit(256<<10) {
		t.Fatalf("limits not applied: %v %v", d.speedLimits.Download.Limit(), d.speedLimits.Upload.Limit())
	}
	bad := next
	bad.UploadSpeedLimit = -1
	if err := d.PutConfig(&bad); err == nil {
		t.Fatal("negative limit accepted")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	// A restarted downloader reads the saved limits back.
	d2 := NewDownloader(&DownloaderConfig{Storage: NewBoltStorage(dir), StorageDir: dir})
	if err := d2.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	if d2.speedLimits.Download.Limit() != rate.Limit(1<<20) || d2.speedLimits.Upload.Limit() != rate.Limit(256<<10) {
		t.Fatal("saved limits not applied after restart")
	}
	if cfg, _ := d2.GetConfig(); cfg.DownloadSpeedLimit != 1<<20 || cfg.UploadSpeedLimit != 256<<10 {
		t.Fatalf("limit not persisted: %+v", cfg)
	}
}

func TestDownloader_HTTPDownloadRespectsSpeedLimit(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	dir := t.TempDir()
	d := NewDownloader(&DownloaderConfig{Storage: NewBoltStorage(dir), StorageDir: dir})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Clear()
	cfg, _ := d.GetConfig()
	next := *cfg
	// 200 MiB at 64 MiB/s needs a little over 3 s.
	next.DownloadSpeedLimit = 64 << 20
	if err := d.PutConfig(&next); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	d.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			close(done)
		}
	})
	rr, err := d.Resolve(&base.Request{URL: "http://" + listener.Addr().String() + "/" + test.BuildName})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := d.CreateWithOptions(rr.ID, newTestDownloadOpt(t)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("download did not finish")
	}
	if elapsed := time.Since(start); elapsed < 2500*time.Millisecond {
		t.Fatalf("200 MiB finished in %v, limit not applied", elapsed)
	}
}
