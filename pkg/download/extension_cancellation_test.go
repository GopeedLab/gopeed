package download

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func installCancellationExtension(t *testing.T, d *Downloader, event, script string) {
	t.Helper()
	dir := t.TempDir()
	manifest := fmt.Sprintf(`{"name":"cancel-%s","author":"test","title":"Cancel","version":"1.0.0","scripts":[{"event":%q,"match":{"urls":["*://example.com/*"]},"entry":"index.js"}]}`, event, event)
	for name, body := range map[string]string{"manifest.json": manifest, "index.js": script} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.InstallExtensionByFolder(dir, true); err != nil {
		t.Fatal(err)
	}
}
func newCancellationDownloader(t *testing.T) *Downloader {
	t.Helper()
	d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&generationTestManager{holdOpen: true}}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	return d
}
func TestResolveExtensionStopsOnShutdown(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	d := newCancellationDownloader(t)
	defer d.Close()
	installCancellationExtension(t, d, "onResolve", fmt.Sprintf(`gopeed.events.onResolve(async ctx=>{await fetch(%q);});`, server.URL))
	done := make(chan error, 1)
	go func() { _, err := d.Resolve(&base.Request{URL: "https://example.com/cancel"}); done <- err }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not enter")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("got %v, want cancellation", err)
		}
	case <-time.After(300 * time.Millisecond):
		once.Do(func() { close(release) })
		t.Fatal("shutdown did not cancel extension Resolve")
	}
}
func TestOnErrorCancellationReleasesTaskAPIs(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	d := newCancellationDownloader(t)
	defer d.Close()
	parent, cancel := context.WithCancel(d.backgroundCtx)
	d.backgroundCtx = parent
	defer cancel()
	installCancellationExtension(t, d, "onError", fmt.Sprintf(`gopeed.events.onError(async ctx=>{await fetch(%q);});`, server.URL))
	d.cfg.MaxRunning = 0
	id, err := d.CreateDirect(&base.Request{URL: "generation://example.com/task"}, &base.Options{Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	task := d.GetTask(id)
	task.statusLock.Lock()
	task.Status = base.DownloadStatusRunning
	task.statusLock.Unlock()
	handled := make(chan struct{})
	go func() { d.doOnError(task, errors.New("source failed")); close(handled) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("onError did not enter")
	}
	status, stats := make(chan struct{}), make(chan struct{})
	go func() { d.RuntimeStatus(id); close(status) }()
	go func() { d.Stats(id); close(stats) }()
	cancel()
	for name, ch := range map[string]<-chan struct{}{"onError": handled, "RuntimeStatus": status, "Stats": stats} {
		select {
		case <-ch:
		case <-time.After(300 * time.Millisecond):
			once.Do(func() { close(release) })
			<-handled
			t.Fatalf("%s stayed blocked after cancellation", name)
		}
	}
}
