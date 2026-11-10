package download

import (
	"errors"
	"fmt"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestReleaseResolveCancelsPendingHTTPHeaders(t *testing.T) {
	entered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	defer server.Close()
	d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	done := make(chan error, 1)
	go func() {
		_, err := d.ResolveWithID("pending-http", &base.Request{URL: server.URL + "/file"})
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	if err := d.ReleaseResolve("pending-http"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled Resolve succeeded")
		}
	case <-time.After(300 * time.Millisecond):
		server.CloseClientConnections()
		t.Fatal("ReleaseResolve did not cancel the request waiting for headers")
	}
	if len(d.fetcherCache) != 0 || len(d.resolveStates) != 0 {
		t.Fatal("cancelled operation retained resources")
	}
}
func TestReleaseResolveAfterCreateDoesNotCloseTask(t *testing.T) {
	d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: nil})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.cfg.MaxRunning = 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer server.Close()
	rr, err := d.Resolve(&base.Request{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	id, err := d.CreateWithOptions(rr.ID, &base.Options{Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.ReleaseResolve(rr.ID); err != nil {
		t.Fatal(err)
	}
	task := d.GetTask(id)
	if task == nil {
		t.Fatal("late release removed task")
	}
	if err := d.doStart(task); err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, d, id, base.DownloadStatusDone, 3*time.Second)
}
func TestResolveRejectsClosedDownloader(t *testing.T) {
	d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	d.Close()
	if _, err := d.ResolveWithID("closed", &base.Request{URL: "http://127.0.0.1/"}); err == nil {
		t.Fatalf("closed downloader accepted Resolve: %v", err)
	}
}

func TestReleaseResolveCancelsPendingMagnetMetadata(t *testing.T) {
	d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	done := make(chan error, 1)
	go func() {
		_, err := d.ResolveWithID("pending-magnet", &base.Request{URL: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"})
		done <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		d.fetcherMapLock.Lock()
		state := d.resolveStates["pending-magnet"]
		ready := state != nil && state.fetcher != nil
		d.fetcherMapLock.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("magnet resolve did not start")
		}
		time.Sleep(time.Millisecond)
	}
	if err := d.ReleaseResolve("pending-magnet"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled magnet Resolve succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled magnet remained in metadata wait")
	}
	if len(d.fetcherCache) != 0 || len(d.resolveStates) != 0 {
		t.Fatal("cancelled magnet retained its resource")
	}
}
func TestCloseDrainsResolvedPreviewCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer server.Close()
	d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Resolve(&base.Request{URL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if len(d.fetcherCache) != 0 || len(d.resolveStates) != 0 {
		t.Fatal("shutdown retained preview resources")
	}
}

type cancelledCreationFetcher struct {
	fetcher.Fetcher
	entered chan struct{}
	release chan struct{}
	err     error
}

func (f *cancelledCreationFetcher) Patch(*base.Request, *base.Options) error {
	close(f.entered)
	<-f.release
	return f.err
}
func TestReleaseResolveDuringCreationTransfersOrClosesOwnership(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("fail=%v", fail), func(t *testing.T) {
			d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir()})
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			d.cfg.MaxRunning = 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
			defer server.Close()
			rr, err := d.Resolve(&base.Request{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			f := &cancelledCreationFetcher{Fetcher: d.fetcherCache[rr.ID], entered: make(chan struct{}), release: make(chan struct{})}
			if fail {
				f.err = errors.New("creation failed")
			}
			d.fetcherMapLock.Lock()
			d.fetcherCache[rr.ID] = f
			d.fetcherMapLock.Unlock()
			done := make(chan struct{})
			var id string
			var createErr error
			go func() {
				defer close(done)
				id, createErr = d.CreateWithOptions(rr.ID, &base.Options{Path: t.TempDir(), Size: 2})
			}()
			select {
			case <-f.entered:
			case <-time.After(time.Second):
				t.Fatal("creation did not claim resource")
			}
			if err := d.ReleaseResolve(rr.ID); err != nil {
				t.Fatal(err)
			}
			close(f.release)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("creation did not return")
			}
			if len(d.fetcherCache) != 0 || len(d.resolveStates) != 0 {
				t.Fatal("cancelled creation returned ownership to preview cache")
			}
			if fail {
				if createErr == nil {
					t.Fatal("expected creation failure")
				}
				return
			}
			if createErr != nil {
				t.Fatal(createErr)
			}
			// The protocol manager persists its concrete Fetcher type.
			task := d.GetTask(id)
			task.fetcher = f.Fetcher
			if err := d.doStart(task); err != nil {
				t.Fatal(err)
			}
			waitForTaskStatus(t, d, id, base.DownloadStatusDone, 3*time.Second)
		})
	}
}
