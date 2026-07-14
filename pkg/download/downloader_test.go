package download

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internalblob "github.com/GopeedLab/gopeed/internal/blob"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
)

type generationTestManager struct {
	starts         atomic.Int32
	pauses         atomic.Int32
	holdOpen       bool
	resolveStarted chan struct{}
	resolveRelease <-chan struct{}
	resolveErr     error
	resolveOnce    sync.Once
	pauseStarted   chan struct{}
	pauseRelease   <-chan struct{}
	pauseOnce      sync.Once
}

func TestExtensionTaskControlMethodsOnlyExistOnError(t *testing.T) {
	taskType := reflect.TypeOf((*Task)(nil))
	if _, ok := taskType.MethodByName("Continue"); ok {
		t.Fatal("regular task unexpectedly exposes Continue")
	}
	if _, ok := taskType.MethodByName("Pause"); ok {
		t.Fatal("regular task unexpectedly exposes Pause")
	}

	errorTaskType := reflect.TypeOf((*ExtensionTask)(nil))
	if _, ok := errorTaskType.MethodByName("Continue"); !ok {
		t.Fatal("onError task does not expose Continue")
	}
	if _, ok := errorTaskType.MethodByName("Pause"); ok {
		t.Fatal("onError task unexpectedly exposes Pause")
	}

	for name, contextType := range map[string]reflect.Type{
		"onStart": reflect.TypeOf(OnStartContext{}),
		"onDone":  reflect.TypeOf(OnDoneContext{}),
	} {
		field, _ := contextType.FieldByName("Task")
		if field.Type != taskType {
			t.Fatalf("%s unexpectedly injects a control wrapper: %v", name, field.Type)
		}
	}
	errorTaskField, _ := reflect.TypeOf(OnErrorContext{}).FieldByName("Task")
	if errorTaskField.Type != errorTaskType {
		t.Fatalf("onError does not inject the Continue wrapper: %v", errorTaskField.Type)
	}
}

func (m *generationTestManager) Name() string { return "generation" }
func (m *generationTestManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{{Type: fetcher.FilterTypeUrl, Pattern: "generation"}}
}
func (m *generationTestManager) Build() fetcher.Fetcher {
	return &generationTestFetcher{manager: m, meta: &fetcher.FetcherMeta{}}
}
func (m *generationTestManager) ParseName(string) string            { return "generation.bin" }
func (m *generationTestManager) AutoRename() bool                   { return false }
func (m *generationTestManager) DefaultConfig() any                 { return map[string]any{} }
func (m *generationTestManager) Store(fetcher.Fetcher) (any, error) { return nil, nil }
func (m *generationTestManager) Restore() (any, func(*fetcher.FetcherMeta, any) fetcher.Fetcher) {
	return nil, func(meta *fetcher.FetcherMeta, _ any) fetcher.Fetcher {
		return &generationTestFetcher{manager: m, meta: meta}
	}
}
func (m *generationTestManager) Close() error { return nil }

type generationTestFetcher struct {
	manager *generationTestManager
	meta    *fetcher.FetcherMeta
	done    chan error
}

func (f *generationTestFetcher) Setup(*controller.Controller) {
	if f.done == nil {
		f.done = make(chan error, 2)
	}
}
func (f *generationTestFetcher) Resolve(req *base.Request, opts *base.Options) error {
	if f.manager.resolveStarted != nil {
		f.manager.resolveOnce.Do(func() { close(f.manager.resolveStarted) })
	}
	if f.manager.resolveRelease != nil {
		<-f.manager.resolveRelease
	}
	if f.manager.resolveErr != nil {
		return f.manager.resolveErr
	}
	f.meta.Req = req
	f.meta.Opts = opts
	f.meta.Res = &base.Resource{Files: []*base.FileInfo{{Name: "generation.bin", Size: 1}}}
	return nil
}
func (f *generationTestFetcher) Start() error {
	f.manager.starts.Add(1)
	if !f.manager.holdOpen {
		f.done <- nil
	}
	return nil
}
func (f *generationTestFetcher) Patch(req *base.Request, opts *base.Options) error {
	if req != nil && req.URL != "" {
		f.meta.Req.URL = req.URL
	}
	if opts != nil {
		f.meta.Opts = opts
	}
	return nil
}
func (f *generationTestFetcher) Pause() error {
	f.manager.pauses.Add(1)
	if f.manager.pauseStarted != nil {
		f.manager.pauseOnce.Do(func() { close(f.manager.pauseStarted) })
	}
	if f.manager.pauseRelease != nil {
		<-f.manager.pauseRelease
	}
	return nil
}
func (f *generationTestFetcher) Close() error               { return nil }
func (f *generationTestFetcher) Stats() any                 { return nil }
func (f *generationTestFetcher) Meta() *fetcher.FetcherMeta { return f.meta }
func (f *generationTestFetcher) Progress() fetcher.Progress { return fetcher.Progress{1} }
func (f *generationTestFetcher) Wait() error                { return <-f.done }

func newTestDownloadOpt(t *testing.T) *base.Options {
	t.Helper()
	return newTestDownloadOptAt(t.TempDir())
}

func newTestDownloadOptAt(path string) *base.Options {
	return &base.Options{
		Path: path,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	}
}

func TestDownloader_Resolve(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	rr, err := downloader.Resolve(req, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := &base.Resource{
		Size:  test.BuildSize,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Path: "",
				Size: test.BuildSize,
			},
		},
	}
	if !test.AssertResourceEqual(want, rr.Res) {
		t.Errorf("Resolve() got = %v, want %v", rr.Res, want)
	}
}

func TestDownloader_BlobRegistryDoesNotUseStorageDirForSpooling(t *testing.T) {
	storageDir := t.TempDir()
	downloader := NewDownloader(&DownloaderConfig{
		Storage:    NewMemStorage(),
		StorageDir: storageDir,
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	if downloader.blob == nil {
		t.Fatal("expected blob registry to be initialized")
	}
	if downloader.blob.Dir() != "" {
		t.Fatalf("expected blob registry not to use a spool dir, got %s", downloader.blob.Dir())
	}
	if _, err := os.Stat(filepath.Join(storageDir, "blob")); !os.IsNotExist(err) {
		t.Fatalf("expected blob spool dir not to be created, got err=%v", err)
	}
}

func TestDownloader_BlobTaskKeepsConfiguredHTTPConnections(t *testing.T) {
	downloader := NewDownloader(&DownloaderConfig{
		Storage:    NewMemStorage(),
		StorageDir: t.TempDir(),
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	url, err := downloader.blob.CreateBlob([]byte("hello"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	id, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{
		Path:  t.TempDir(),
		Name:  "blob-unmarked.txt",
		Extra: &http.OptsExtra{Connections: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(id)
	if task == nil {
		t.Fatal("task not found")
	}
	if task.Protocol != "http" {
		t.Fatalf("expected blob task protocol http, got %s", task.Protocol)
	}
	extra, ok := task.Meta.Opts.Extra.(*http.OptsExtra)
	if !ok {
		t.Fatalf("expected http extra, got %T", task.Meta.Opts.Extra)
	}
	if extra.Connections != 8 {
		t.Fatalf("expected blob task connections to stay 8, got %d", extra.Connections)
	}
	waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 5*time.Second)
}

func TestDownloader_ExternalBlobLikePathUsesNormalHTTP(t *testing.T) {
	payload := []byte("external blob-like path")
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if r.URL.Path != "/__blob/file.bin" {
			gohttp.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	dir := t.TempDir()
	id, err := downloader.CreateDirect(&base.Request{URL: server.URL + "/__blob/file.bin"}, &base.Options{
		Path: dir,
		Name: "external.bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 5*time.Second)
	got, err := os.ReadFile(filepath.Join(dir, "external.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("unexpected external payload %q", string(got))
	}
}

func TestDownloader_InternalBlobBypassesGlobalProxy(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloader.cfg.DownloaderStoreConfig.Proxy = &base.DownloaderProxyConfig{
		Enable: true,
		Scheme: "http",
		Host:   "127.0.0.1:1",
	}

	payload := []byte("direct blob")
	blobURL, err := downloader.blob.CreateBlob(payload, "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	id, err := downloader.CreateDirect(&base.Request{URL: blobURL}, &base.Options{
		Path: dir,
		Name: "direct.bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 5*time.Second)
	got, err := os.ReadFile(filepath.Join(dir, "direct.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("unexpected blob payload %q", string(got))
	}
}

func TestDownloader_RegistryMissUsesOrdinaryHTTPError(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	liveURL, err := downloader.blob.CreateBlob([]byte("unused"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	missingURL := liveURL[:len(liveURL)-1] + "x"
	if downloader.blob.IsURL(missingURL) {
		t.Fatal("test URL unexpectedly matched the Registry")
	}

	errorCh := make(chan error, 1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyError {
			select {
			case errorCh <- event.Err:
			default:
			}
		}
	})
	_, err = downloader.CreateDirect(&base.Request{URL: missingURL}, &base.Options{
		Path: t.TempDir(),
		Name: "missing.bin",
	})
	if err != nil {
		t.Fatalf("Registry miss was rejected before ordinary HTTP handling: %v", err)
	}
	select {
	case got := <-errorCh:
		if errors.Is(got, internalblob.ErrSourceNotFound) {
			t.Fatalf("Registry miss surfaced Blob error instead of HTTP failure: %v", got)
		}
		if got == nil || !strings.Contains(got.Error(), "404") {
			t.Fatalf("expected ordinary HTTP 404 failure, got %v", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Registry miss did not reach the ordinary HTTP error path")
	}
}

func TestDownloader_TasksShareBlobLease(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloader.cfg.MaxRunning = 1

	payload := bytes.Repeat([]byte("shared-blob-"), 8192)
	allowLaterOpens := make(chan struct{})
	var opens atomic.Int32
	blobURL, err := downloader.blob.CreateOpener(func(ctx context.Context, req internalblob.OpenRequest) (io.ReadCloser, error) {
		if opens.Add(1) > 1 {
			select {
			case <-allowLaterOpens:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		start := req.Offset
		end := int64(len(payload))
		if req.End >= start && req.End+1 < end {
			end = req.End + 1
		}
		return io.NopCloser(bytes.NewReader(payload[start:end])), nil
	}, &internalblob.CreateOptions{Size: int64(len(payload))})
	if err != nil {
		t.Fatal(err)
	}

	dirA, dirB := t.TempDir(), t.TempDir()
	idA, err := downloader.CreateDirect(&base.Request{URL: blobURL}, &base.Options{Path: dirA, Name: "a.bin"})
	if err != nil {
		t.Fatal(err)
	}
	idB, err := downloader.CreateDirect(&base.Request{URL: blobURL}, &base.Options{Path: dirB, Name: "b.bin"})
	if err != nil {
		t.Fatal(err)
	}

	waitForTaskStatus(t, downloader, idA, base.DownloadStatusDone, 5*time.Second)
	if err := downloader.Delete(&TaskFilter{IDs: []string{idA}}, false); err != nil {
		t.Fatal(err)
	}
	if !downloader.blob.IsURL(blobURL) {
		t.Fatal("deleting the first task revoked a Blob still leased by the second task")
	}
	close(allowLaterOpens)
	waitForTaskStatus(t, downloader, idB, base.DownloadStatusDone, 5*time.Second)
	if downloader.blob.IsURL(blobURL) {
		t.Fatal("final task completion did not release the shared Blob")
	}
	for _, file := range []string{filepath.Join(dirA, "a.bin"), filepath.Join(dirB, "b.bin")} {
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("unexpected shared Blob content in %s", file)
		}
	}
}

func TestDownloader_EmptyBlob(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	blobURL, err := downloader.blob.CreateBlob(nil, "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	id, err := downloader.CreateDirect(&base.Request{URL: blobURL}, &base.Options{Path: dir, Name: "empty.bin"})
	if err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 5*time.Second)
	info, err := os.Stat(filepath.Join(dir, "empty.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("unexpected empty Blob size: %d", info.Size())
	}
}

func TestDownloader_PatchTransfersBlobLease(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	// Keep the task waiting so Patch exercises only metadata and lease transfer.
	downloader.cfg.MaxRunning = 0

	urlA, err := downloader.blob.CreateBlob([]byte("a"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	urlB, err := downloader.blob.CreateBlob([]byte("b"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	id, err := downloader.CreateDirect(&base.Request{URL: urlA}, &base.Options{Path: t.TempDir(), Name: "patched.bin"})
	if err != nil {
		t.Fatal(err)
	}
	if err := downloader.Patch(id, &base.Request{URL: urlB}, nil); err != nil {
		t.Fatal(err)
	}
	if downloader.blob.IsURL(urlA) {
		t.Fatal("Patch retained the old Blob lease")
	}
	if !downloader.blob.IsURL(urlB) {
		t.Fatal("Patch did not acquire the new Blob lease")
	}
	if err := downloader.Delete(&TaskFilter{IDs: []string{id}}, false); err != nil {
		t.Fatal(err)
	}
	if downloader.blob.IsURL(urlB) {
		t.Fatal("deleting patched task did not release the new Blob lease")
	}
}

func TestDownloader_StaleStartGenerationCannotRestartTask(t *testing.T) {
	manager := &generationTestManager{}
	downloader := NewDownloader(&DownloaderConfig{FetchManagers: []fetcher.FetcherManager{manager}})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloader.cfg.MaxRunning = 0

	id, err := downloader.CreateDirect(&base.Request{URL: "generation://test"}, &base.Options{
		Path: t.TempDir(),
		Name: "generation.bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(id)
	if task == nil || task.Status != base.DownloadStatusWait {
		t.Fatalf("expected queued task, got %#v", task)
	}

	// Hold task.lock so the first Start, its Pause handler, and the second Start
	// are all queued. Only the newest start generation may reach Fetcher.Start.
	task.lock.Lock()
	downloader.cfg.MaxRunning = 1
	if err := downloader.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatal(err)
	}
	pauseDone := make(chan error, 1)
	go func() {
		pauseDone <- downloader.Pause(&TaskFilter{IDs: []string{id}})
	}()
	deadline := time.Now().Add(2 * time.Second)
	for downloader.taskStatus(task) != base.DownloadStatusPause && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if downloader.taskStatus(task) != base.DownloadStatusPause {
		t.Fatal("Pause did not update task status")
	}
	if err := downloader.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatal(err)
	}
	task.lock.Unlock()
	select {
	case err := <-pauseDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Pause did not finish")
	}

	waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 3*time.Second)
	if got := manager.starts.Load(); got != 1 {
		t.Fatalf("stale start generation reached Fetcher.Start: got %d starts, want 1", got)
	}
}

func TestDownloader_PauseWinsAgainstStaleResolveError(t *testing.T) {
	releaseResolve := make(chan struct{})
	manager := &generationTestManager{
		resolveStarted: make(chan struct{}),
		resolveRelease: releaseResolve,
		resolveErr:     errors.New("stale resolve failed"),
	}
	downloader := NewDownloader(&DownloaderConfig{FetchManagers: []fetcher.FetcherManager{manager}})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	errorEvent := make(chan error, 1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyError {
			errorEvent <- event.Err
		}
	})
	id, err := downloader.CreateDirect(&base.Request{URL: "generation://stale-resolve"}, &base.Options{
		Path: t.TempDir(),
		Name: "stale.bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-manager.resolveStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("Resolve did not start")
	}

	pauseDone := make(chan error, 1)
	go func() {
		pauseDone <- downloader.Pause(&TaskFilter{IDs: []string{id}})
	}()
	deadline := time.Now().Add(2 * time.Second)
	for downloader.taskStatus(downloader.GetTask(id)) != base.DownloadStatusPause && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := downloader.taskStatus(downloader.GetTask(id)); got != base.DownloadStatusPause {
		t.Fatalf("Pause did not win the status transition: %s", got)
	}
	close(releaseResolve)
	select {
	case err := <-pauseDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Pause did not finish after stale Resolve returned")
	}
	if got := downloader.taskStatus(downloader.GetTask(id)); got != base.DownloadStatusPause {
		t.Fatalf("stale Resolve error overwrote Pause: %s", got)
	}
	select {
	case err := <-errorEvent:
		t.Fatalf("stale Resolve emitted onError after Pause: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDownloader_SchedulingPauseCompletesBeforeReplacementStart(t *testing.T) {
	releasePause := make(chan struct{})
	manager := &generationTestManager{
		holdOpen:     true,
		pauseStarted: make(chan struct{}),
		pauseRelease: releasePause,
	}
	downloader := NewDownloader(&DownloaderConfig{FetchManagers: []fetcher.FetcherManager{manager}})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloader.cfg.MaxRunning = 1

	_, err := downloader.CreateDirect(&base.Request{URL: "generation://a"}, &base.Options{
		Path: t.TempDir(), Name: "a.bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for manager.starts.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if manager.starts.Load() != 1 {
		t.Fatal("first task did not physically start")
	}
	idB, err := downloader.CreateDirect(&base.Request{URL: "generation://b"}, &base.Options{
		Path: t.TempDir(), Name: "b.bin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if downloader.taskStatus(downloader.GetTask(idB)) != base.DownloadStatusWait {
		t.Fatal("second task was not queued")
	}

	continueDone := make(chan error, 1)
	go func() {
		continueDone <- downloader.Continue(&TaskFilter{IDs: []string{idB}})
	}()
	select {
	case <-manager.pauseStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("running task was not physically paused")
	}
	if got := manager.starts.Load(); got != 1 {
		t.Fatalf("replacement started before Pause completed: %d starts", got)
	}
	close(releasePause)
	select {
	case err := <-continueDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Continue did not return after Pause completed")
	}
	deadline = time.Now().Add(2 * time.Second)
	for manager.starts.Load() != 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := manager.starts.Load(); got != 2 {
		t.Fatalf("replacement task did not start: %d starts", got)
	}
	if got := manager.pauses.Load(); got != 1 {
		t.Fatalf("unexpected physical Pause count: %d", got)
	}
}

func TestDownloader_Create(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	opts := newTestDownloadOpt(t)
	rr, err := downloader.Resolve(req, opts)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})
	_, err = downloader.Create(rr.ID)
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(filepath.Join(opts.Path, opts.Name))
	if want != got {
		t.Errorf("Downloader_Create() got = %v, want %v", got, want)
	}
}

func TestDownloader_CreateNotInWhite(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(&DownloaderConfig{
		WhiteDownloadDirs: []string{"./downloads"},
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	// With new fetcher design, white list check happens during Resolve (not Create)
	// because Resolve now requires Options which includes the download path
	_, err := downloader.Resolve(req, newTestDownloadOpt(t))
	if err == nil {
		t.Error("TestDownloader_CreateNotInWhite() expected error but got nil")
	}
	if !strings.Contains(err.Error(), "white") {
		t.Errorf("TestDownloader_CreateNotInWhite() got = %v, want error containing 'white'", err.Error())
	}
}

func TestDownloader_CreateDirectBatch(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	reqs := make([]*base.CreateTaskBatchItem, 0)
	fileNames := make([]string, 0)
	for i := 0; i < 5; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		reqs = append(reqs, &base.CreateTaskBatchItem{
			Req: req,
		})
		if i == 0 {
			fileNames = append(fileNames, test.DownloadName)
		} else {
			arr := strings.Split(test.DownloadName, ".")
			fileNames = append(fileNames, arr[0]+" ("+strconv.Itoa(i)+")."+arr[1])
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(reqs))
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	opts := newTestDownloadOpt(t)
	_, err := downloader.CreateDirectBatch(&base.CreateTaskBatch{
		Reqs: reqs,
		Opts: opts,
	})
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	tasks := downloader.GetTasks()
	if len(tasks) != len(reqs) {
		t.Errorf("CreateDirectBatch() task got = %v, want %v", len(tasks), len(reqs))
	}

	// Collect all task names
	taskNames := make(map[string]bool)
	for _, task := range tasks {
		taskNames[task.Meta.Opts.Name] = true
	}

	// Check that we have the expected number of unique task names
	if len(taskNames) != len(reqs) {
		t.Errorf("CreateDirectBatch() unique task names got = %v, want %v, names: %v", len(taskNames), len(reqs), taskNames)
	}

	// Check that all task files exist
	for name := range taskNames {
		if _, err := os.Stat(filepath.Join(opts.Path, name)); os.IsNotExist(err) {
			t.Errorf("CreateDirectBatch() file not exist: %v", name)
		}
	}

}

func TestDownloader_CreateWithProxy(t *testing.T) {
	// No proxy
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		return nil
	}, nil)
	// Disable proxy
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.Enable = false
		return proxyCfg
	}, nil)
	// Enable system proxy but not set proxy environment variable
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.System = true
		return proxyCfg
	}, nil)
	// Enable proxy but error proxy environment variable
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		os.Setenv("HTTP_PROXY", "http://127.0.0.1:1234")
		os.Setenv("HTTPS_PROXY", "http://127.0.0.1:1234")
		proxyCfg.System = true
		return proxyCfg
	}, func(err error) {
		if err == nil {
			t.Fatal("doTestDownloaderCreateWithProxy() got = nil, want error")
		}
	})
	// Enable system proxy and set proxy environment variable
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		os.Setenv("HTTP_PROXY", proxyCfg.ToUrl().String())
		os.Setenv("HTTPS_PROXY", proxyCfg.ToUrl().String())
		proxyCfg.System = true
		return proxyCfg
	}, nil)
	// Invalid proxy scheme
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.Scheme = ""
		return proxyCfg
	}, nil)
	// Invalid proxy host
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.Host = ""
		return proxyCfg
	}, nil)
	// Use proxy without auth
	doTestDownloaderCreateWithProxy(t, false, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		return proxyCfg
	}, nil)
	// Use proxy with auth
	doTestDownloaderCreateWithProxy(t, true, nil, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		return proxyCfg
	}, nil)

	// Request proxy mode follow
	doTestDownloaderCreateWithProxy(t, false, func(reqProxy *base.RequestProxy) *base.RequestProxy {
		reqProxy.Mode = base.RequestProxyModeFollow
		return reqProxy
	}, nil, nil)

	// Request proxy mode none
	doTestDownloaderCreateWithProxy(t, false, func(reqProxy *base.RequestProxy) *base.RequestProxy {
		reqProxy.Mode = base.RequestProxyModeNone
		return reqProxy
	}, nil, nil)

	// Request proxy mode custom
	doTestDownloaderCreateWithProxy(t, false, func(reqProxy *base.RequestProxy) *base.RequestProxy {
		return reqProxy
	}, nil, nil)
}

func doTestDownloaderCreateWithProxy(t *testing.T, auth bool, buildReqProxy func(reqProxy *base.RequestProxy) *base.RequestProxy, buildProxyConfig func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig, errHandler func(err error)) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	usr, pwd := "", ""
	if auth {
		usr, pwd = "admin", "123"
	}
	proxyListener := test.StartSocks5Server(usr, pwd)
	defer proxyListener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	globalProxyCfg := &base.DownloaderProxyConfig{
		Enable: true,
		Scheme: "socks5",
		Host:   proxyListener.Addr().String(),
		Usr:    usr,
		Pwd:    pwd,
	}
	if buildProxyConfig != nil {
		globalProxyCfg = buildProxyConfig(globalProxyCfg)
	}
	downloader.cfg.DownloaderStoreConfig.Proxy = globalProxyCfg

	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	if buildReqProxy != nil {
		req.Proxy = buildReqProxy(&base.RequestProxy{
			Scheme: "socks5",
			Host:   proxyListener.Addr().String(),
			Usr:    usr,
			Pwd:    pwd,
		})
	}
	rr, err := downloader.Resolve(req, nil)
	if err != nil {
		if errHandler == nil {
			t.Fatal(err)
		}
		errHandler(err)
		return
	}
	want := &base.Resource{
		Size:  test.BuildSize,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Path: "",
				Size: test.BuildSize,
			},
		},
	}
	if !test.AssertResourceEqual(want, rr.Res) {
		t.Errorf("Resolve() got = %v, want %v", rr.Res, want)
	}
}

func TestDownloader_CreateRename(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	downloadDir := t.TempDir()
	var wg sync.WaitGroup
	wg.Add(2)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})
	for i := 0; i < 2; i++ {
		_, err := downloader.CreateDirect(req, &base.Options{
			Path: downloadDir,
			Name: test.DownloadName,
			Extra: http.OptsExtra{
				Connections: 4,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()

	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(filepath.Join(downloadDir, test.DownloadName))
	if want != got {
		t.Errorf("Downloader_CreateRename() got = %v, want %v", got, want)
	}
	got = test.FileMd5(filepath.Join(downloadDir, test.DownloadRename))
	if want != got {
		t.Errorf("Downloader_CreateRename() got = %v, want %v", got, want)
	}
}

func TestDownloader_StoreAndRestore(t *testing.T) {
	listener := test.StartTestSlowFileServer(time.Millisecond * 2000)
	defer listener.Close()

	storageDir := t.TempDir()
	downloader := NewDownloader(&DownloaderConfig{
		Storage: NewBoltStorage(storageDir),
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	opts := newTestDownloadOpt(t)
	rr, err := downloader.Resolve(req, opts)
	if err != nil {
		t.Fatal(err)
	}

	pauseResult := make(chan error, 1)
	var pauseOnce sync.Once
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyStart {
			pauseOnce.Do(func() {
				pauseResult <- downloader.Pause(&TaskFilter{IDs: []string{event.Task.ID}})
			})
		}
	})
	id, err := downloader.Create(rr.ID)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-pauseResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for task to pause")
	}
	downloader.Close()

	downloader = NewDownloader(&DownloaderConfig{
		Storage: NewBoltStorage(storageDir),
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(id)

	if task == nil {
		t.Fatal("task is nil")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})
	err = downloader.Continue(&TaskFilter{IDs: []string{id}})
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(filepath.Join(opts.Path, opts.Name))
	if want != got {
		t.Errorf("StoreAndResume() got = %v, want %v", got, want)
	}

	downloader.Clear()
}

func TestDownloader_Protocol_Config(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	var httpCfg map[string]any
	exits := downloader.getProtocolConfig("http", &httpCfg)
	if !exits {
		t.Errorf("getProtocolConfig() got = %v, want %v", exits, true)
	}

	storeCfg := &base.DownloaderStoreConfig{
		DownloadDir: "./downloads",
		ProtocolConfig: map[string]any{
			"http": map[string]any{
				"connections": 4,
			},
			"bt": map[string]any{
				"trackerSubscribeUrls": []string{
					"https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/best.txt",
				},
				"trackers": []string{
					"udp://tracker.coppersurfer.tk:6969/announce",
					"udp://tracker.leechers-paradise.org:6969/announce",
				},
			},
		},
		Extra: map[string]any{
			"theme": "dark",
		},
	}

	if err := downloader.PutConfig(storeCfg); err != nil {
		t.Fatal(err)
	}

	newStoreCfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}

	if !test.JsonEqual(storeCfg, newStoreCfg) {
		t.Errorf("GetConfig() got = %v, want %v", test.ToJson(storeCfg), test.ToJson(newStoreCfg))
	}
}

func TestDownloader_GetTasksByFilter(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	reqs := make([]*base.CreateTaskBatchItem, 0)
	fileNames := make([]string, 0)
	for i := 0; i < 10; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		reqs = append(reqs, &base.CreateTaskBatchItem{
			Req: req,
		})
		if i == 0 {
			fileNames = append(fileNames, test.DownloadName)
		} else {
			arr := strings.Split(test.DownloadName, ".")
			fileNames = append(fileNames, arr[0]+" ("+strconv.Itoa(i)+")."+arr[1])
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(reqs))
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	batchOpts := newTestDownloadOpt(t)
	taskIds, err := downloader.CreateDirectBatch(&base.CreateTaskBatch{
		Reqs: reqs,
		Opts: batchOpts,
	})
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	t.Run("GetTasksByFilter nil", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(nil)
		if len(tasks) != len(reqs) {
			t.Errorf("GetTasksByFilter nil task got = %v, want %v", len(tasks), len(reqs))
		}
	})

	t.Run("GetTasksByFilter empty", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{})
		if len(tasks) != len(reqs) {
			t.Errorf("GetTasksByFilter empty task got = %v, want %v", len(tasks), len(reqs))
		}
	})

	t.Run("GetTasksByFilter ids", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			IDs: taskIds,
		})
		if len(tasks) != len(reqs) {
			t.Errorf("GetTasksByFilter ids task got = %v, want %v", len(tasks), len(reqs))
		}
	})

	t.Run("GetTasksByFilter match ids", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			IDs: []string{taskIds[0]},
		})
		if len(tasks) != 1 {
			t.Errorf("GetTasksByFilter ids task got = %v, want %v", len(tasks), 1)
		}
	})

	t.Run("GetTasksByFilter not match ids", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			IDs: []string{"xxx"},
		})
		if len(tasks) != 0 {
			t.Errorf("GetTasksByFilter ids task got = %v, want %v", len(tasks), 0)
		}
	})

	t.Run("GetTasksByFilter status", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			Statuses: []base.Status{base.DownloadStatusDone},
		})
		if len(tasks) != len(reqs) {
			t.Errorf("GetTasksByFilter status task got = %v, want %v", len(tasks), len(reqs))
		}
	})

	t.Run("GetTasksByFilter not match status", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			Statuses: []base.Status{base.DownloadStatusError},
		})
		if len(tasks) != 0 {
			t.Errorf("GetTasksByFilter status task got = %v, want %v", len(tasks), 0)
		}
	})

	t.Run("GetTasksByFilter match notStatus", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			NotStatuses: []base.Status{base.DownloadStatusRunning, base.DownloadStatusPause},
		})
		if len(tasks) != len(reqs) {
			t.Errorf("GetTasksByFilter match notStatus task got = %v, want %v", len(tasks), len(reqs))
		}
	})

	t.Run("GetTasksByFilter not match notStatus", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			NotStatuses: []base.Status{base.DownloadStatusDone},
		})
		if len(tasks) != 0 {
			t.Errorf("GetTasksByFilter not match notStatus task got = %v, want %v", len(tasks), 0)
		}
	})

	t.Run("GetTasksByFilter match ids and status", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			IDs:      []string{taskIds[0]},
			Statuses: []base.Status{base.DownloadStatusDone},
		})
		if len(tasks) != 1 {
			t.Errorf("GetTasksByFilter match ids and status task got = %v, want %v", len(tasks), 1)
		}
	})

	t.Run("GetTasksByFilter not match ids and status", func(t *testing.T) {
		tasks := downloader.GetTasksByFilter(&TaskFilter{
			IDs:      []string{taskIds[0]},
			Statuses: []base.Status{base.DownloadStatusError},
		})
		if len(tasks) != 0 {
			t.Errorf("GetTasksByFilter not match ids and status task got = %v, want %v", len(tasks), 0)
		}
	})

}

func TestDownloader_Stats(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Test Stats for non-existent task
	_, err := downloader.Stats("non-existent-id")
	if err != ErrTaskNotFound {
		t.Errorf("Stats() expected ErrTaskNotFound, got %v", err)
	}

	// Create a task
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	rr, err := downloader.Resolve(req, newTestDownloadOpt(t))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	taskId, err := downloader.Create(rr.ID)
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	// Test Stats for existing task
	stats, err := downloader.Stats(taskId)
	if err != nil {
		t.Errorf("Stats() unexpected error: %v", err)
	}
	if stats == nil {
		t.Error("Stats() returned nil stats")
	}
}

func TestDownloader_Delete(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Create multiple tasks
	var wg sync.WaitGroup
	taskCount := 3
	downloadDir := t.TempDir()
	wg.Add(taskCount)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	taskIds := make([]string, 0)
	for i := 0; i < taskCount; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		taskId, err := downloader.CreateDirect(req, newTestDownloadOptAt(downloadDir))
		if err != nil {
			t.Fatal(err)
		}
		taskIds = append(taskIds, taskId)
	}

	wg.Wait()

	// Test Delete with filter (single task)
	t.Run("Delete single task by ID", func(t *testing.T) {
		initialCount := len(downloader.GetTasks())
		err := downloader.Delete(&TaskFilter{IDs: []string{taskIds[0]}}, true)
		if err != nil {
			t.Errorf("Delete() unexpected error: %v", err)
		}
		newCount := len(downloader.GetTasks())
		if newCount != initialCount-1 {
			t.Errorf("Delete() task count got = %v, want %v", newCount, initialCount-1)
		}
	})

	// Test Delete with non-matching filter (should do nothing)
	t.Run("Delete with non-matching filter", func(t *testing.T) {
		initialCount := len(downloader.GetTasks())
		err := downloader.Delete(&TaskFilter{IDs: []string{"non-existent-id"}}, true)
		if err != nil {
			t.Errorf("Delete() unexpected error: %v", err)
		}
		newCount := len(downloader.GetTasks())
		if newCount != initialCount {
			t.Errorf("Delete() task count got = %v, want %v", newCount, initialCount)
		}
	})

	// Test Delete by status
	t.Run("Delete by status", func(t *testing.T) {
		initialCount := len(downloader.GetTasks())
		err := downloader.Delete(&TaskFilter{Statuses: []base.Status{base.DownloadStatusDone}}, false)
		if err != nil {
			t.Errorf("Delete() unexpected error: %v", err)
		}
		newCount := len(downloader.GetTasks())
		if newCount != 0 {
			t.Errorf("Delete() should have deleted all done tasks, got %v remaining", newCount)
		}
		_ = initialCount // suppress unused variable warning
	})
}

func TestDownloader_PauseAndContinue(t *testing.T) {
	listener := test.StartTestSlowFileServer(time.Millisecond * 2000)
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Create a single task
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	rr, err := downloader.Resolve(req, newTestDownloadOpt(t))
	if err != nil {
		t.Fatal(err)
	}
	taskId, err := downloader.Create(rr.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for task to start
	time.Sleep(time.Millisecond * 100)

	// Pause with specific taskId
	err = downloader.Pause(&TaskFilter{IDs: []string{taskId}})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	// Verify task is paused
	task := downloader.GetTask(taskId)
	if task.Status != base.DownloadStatusPause {
		t.Errorf("Task should be paused, got %s", task.Status)
	}

	// Continue with specific taskId
	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	err = downloader.Continue(&TaskFilter{IDs: []string{taskId}})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for task to complete
	wg.Wait()

	// Verify task is done
	task = downloader.GetTask(taskId)
	if task.Status != base.DownloadStatusDone {
		t.Errorf("Task should be done, got %s", task.Status)
	}

	// Clean up
	downloader.Delete(nil, true)
}

func TestDownloader_PauseAllAndContinueAll(t *testing.T) {
	listener := test.StartTestSlowFileServer(time.Millisecond * 2000)
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Create multiple tasks
	taskCount := 3
	taskIds := make([]string, 0)

	for i := 0; i < taskCount; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		rr, err := downloader.Resolve(req, newTestDownloadOpt(t))
		if err != nil {
			t.Fatal(err)
		}
		taskId, err := downloader.Create(rr.ID)
		if err != nil {
			t.Fatal(err)
		}
		taskIds = append(taskIds, taskId)
	}

	// Wait for tasks to start
	time.Sleep(time.Millisecond * 100)

	// Pause all tasks with nil filter
	err := downloader.Pause(nil)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

	// Verify all tasks are paused
	pausedCount := 0
	for _, taskId := range taskIds {
		task := downloader.GetTask(taskId)
		if task.Status == base.DownloadStatusPause {
			pausedCount++
		}
	}
	if pausedCount != taskCount {
		t.Errorf("Expected %d paused tasks, got %d", taskCount, pausedCount)
	}

	// Continue all tasks with nil filter
	var wg sync.WaitGroup
	wg.Add(taskCount)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	err = downloader.Continue(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Verify all tasks are done
	doneCount := 0
	for _, taskId := range taskIds {
		task := downloader.GetTask(taskId)
		if task.Status == base.DownloadStatusDone {
			doneCount++
		}
	}
	if doneCount != taskCount {
		t.Errorf("Expected %d done tasks, got %d", taskCount, doneCount)
	}

	// Clean up
	downloader.Delete(nil, true)
}

func TestDownloader_GetTask(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Test GetTask for non-existent task
	task := downloader.GetTask("non-existent-id")
	if task != nil {
		t.Errorf("GetTask() expected nil for non-existent task, got %v", task)
	}
}

func TestDownloader_Emit(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Test emit with no listener (should not panic)
	downloader.emit(EventKeyDone, nil)

	// Test emit with listener
	eventReceived := false
	downloader.Listener(func(event *Event) {
		eventReceived = true
	})
	downloader.emit(EventKeyDone, nil)
	if !eventReceived {
		t.Error("Event should have been received by listener")
	}
}

func TestDownloader_AutoExtract(t *testing.T) {
	// Create a temporary directory for extraction tests
	tempDir, err := os.MkdirTemp("", "downloader_extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip file
	zipPath := tempDir + "/test_archive.zip"
	if err := createTestArchive(zipPath); err != nil {
		t.Fatal(err)
	}

	// Verify isArchiveFile works correctly
	t.Run("isArchiveFile", func(t *testing.T) {
		if !isArchiveFile(zipPath) {
			t.Error("isArchiveFile should return true for .zip file")
		}
		if isArchiveFile(tempDir + "/test.txt") {
			t.Error("isArchiveFile should return false for .txt file")
		}
	})
}

// TestDownloader_AutoExtractWithProgress tests the auto-extract functionality with progress tracking
// This test exercises the ExtractStatus and ExtractProgress fields in the Progress struct
func TestDownloader_AutoExtractWithProgress(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "auto_extract_progress_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip file to serve
	zipPath := tempDir + "/archive.zip"
	if err := createTestArchiveWithMultipleFiles(zipPath, 3); err != nil {
		t.Fatal(err)
	}

	// Start a simple HTTP server to serve the zip file
	server := startTestArchiveServer(zipPath)
	defer server.Close()

	// Create downloader
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Track extraction status changes
	var extractStatusChanges []ExtractStatus
	var extractProgressValues []int
	var statusMutex sync.Mutex
	extractDoneCh := make(chan struct{})
	var extractDoneOnce sync.Once

	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyProgress && event.Task != nil && event.Task.Progress != nil {
			statusMutex.Lock()
			status := event.Task.Progress.ExtractStatus
			progress := event.Task.Progress.ExtractProgress
			// Record status changes
			if status != ExtractStatusNone {
				if len(extractStatusChanges) == 0 || extractStatusChanges[len(extractStatusChanges)-1] != status {
					extractStatusChanges = append(extractStatusChanges, status)
				}
				extractProgressValues = append(extractProgressValues, progress)
			}
			statusMutex.Unlock()
			// Signal when extraction is done or errored
			if status == ExtractStatusDone || status == ExtractStatusError {
				extractDoneOnce.Do(func() {
					close(extractDoneCh)
				})
			}
		}
	})

	// Create request to download the zip file
	req := &base.Request{
		URL: "http://" + server.Addr().String() + "/archive.zip",
	}

	// Create task with AutoExtract enabled
	downloadDir := tempDir + "/downloads"
	taskId, err := downloader.CreateDirect(req, &base.Options{
		Path: downloadDir,
		Name: "archive.zip",
		Extra: http.OptsExtra{
			Connections: 1,
			AutoExtract: util.BoolPtr(true),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for extraction to complete (with timeout)
	select {
	case <-extractDoneCh:
		// Extraction completed
	case <-time.After(30 * time.Second):
		t.Log("Extraction timed out, checking results anyway")
	}

	// Give a small buffer for final events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify task exists
	task := downloader.GetTask(taskId)
	if task == nil {
		t.Fatal("Task should exist")
	}

	// Verify extraction status changes occurred
	statusMutex.Lock()
	defer statusMutex.Unlock()

	t.Logf("Recorded extract status changes: %v", extractStatusChanges)
	t.Logf("Recorded extract progress values: %v", extractProgressValues)

	// Verify that we went through ExtractStatusExtracting
	foundExtracting := false
	for _, status := range extractStatusChanges {
		if status == ExtractStatusExtracting {
			foundExtracting = true
			break
		}
	}
	if !foundExtracting {
		t.Error("Expected ExtractStatusExtracting in status changes")
	}

	// Verify that we reached ExtractStatusDone
	foundDone := false
	for _, status := range extractStatusChanges {
		if status == ExtractStatusDone {
			foundDone = true
			break
		}
	}
	if !foundDone {
		t.Error("Expected ExtractStatusDone in status changes")
	}

	// Verify progress values include 100 (final)
	found100 := false
	for _, p := range extractProgressValues {
		if p == 100 {
			found100 = true
			break
		}
	}
	if !found100 {
		t.Error("Expected progress to reach 100")
	}

	// Verify extracted files exist
	extractedFile := downloadDir + "/test_0.txt"
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Error("Expected extracted file to exist")
	}
}

// TestDownloader_AutoExtractWithDeleteAfterExtract tests the auto-extract with DeleteAfterExtract option
func TestDownloader_AutoExtractWithDeleteAfterExtract(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "auto_extract_delete_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip file to serve
	zipPath := tempDir + "/archive.zip"
	if err := createTestArchiveWithMultipleFiles(zipPath, 2); err != nil {
		t.Fatal(err)
	}

	// Start a simple HTTP server to serve the zip file
	server := startTestArchiveServer(zipPath)
	defer server.Close()

	// Create downloader
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Track extraction status changes
	extractDoneCh := make(chan struct{})
	var extractDoneOnce sync.Once

	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyProgress && event.Task != nil && event.Task.Progress != nil {
			status := event.Task.Progress.ExtractStatus
			if status == ExtractStatusDone || status == ExtractStatusError {
				extractDoneOnce.Do(func() {
					close(extractDoneCh)
				})
			}
		}
	})

	// Create request to download the zip file
	req := &base.Request{
		URL: "http://" + server.Addr().String() + "/archive.zip",
	}

	// Create task with AutoExtract and DeleteAfterExtract enabled
	downloadDir := tempDir + "/downloads"
	_, err = downloader.CreateDirect(req, &base.Options{
		Path: downloadDir,
		Name: "archive.zip",
		Extra: http.OptsExtra{
			Connections:        1,
			AutoExtract:        util.BoolPtr(true),
			DeleteAfterExtract: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for extraction to complete (with timeout)
	select {
	case <-extractDoneCh:
		// Extraction completed
	case <-time.After(10 * time.Second):
		t.Log("Extraction timed out")
	}

	// Give time for file deletion
	time.Sleep(200 * time.Millisecond)

	// Verify archive was deleted
	archivePath := downloadDir + "/archive.zip"
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("Expected archive to be deleted after extraction")
	}

	// Verify extracted files exist
	extractedFile := downloadDir + "/test_0.txt"
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Error("Expected extracted file to exist")
	}
}

// TestDownloader_AutoExtractError tests the auto-extract error handling path
func TestDownloader_AutoExtractError(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "auto_extract_error_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a corrupt zip file (just invalid data with .zip extension)
	corruptZipPath := tempDir + "/corrupt.zip"
	if err := os.WriteFile(corruptZipPath, []byte("this is not a valid zip file"), 0644); err != nil {
		t.Fatal(err)
	}

	// Start a simple HTTP server to serve the corrupt zip file
	server := startTestArchiveServer(corruptZipPath)
	defer server.Close()

	// Create downloader
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Track extraction status changes
	var extractStatusChanges []ExtractStatus
	var statusMutex sync.Mutex
	extractDoneCh := make(chan struct{})
	var extractDoneOnce sync.Once

	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyProgress && event.Task != nil && event.Task.Progress != nil {
			statusMutex.Lock()
			status := event.Task.Progress.ExtractStatus
			if status != ExtractStatusNone {
				if len(extractStatusChanges) == 0 || extractStatusChanges[len(extractStatusChanges)-1] != status {
					extractStatusChanges = append(extractStatusChanges, status)
				}
			}
			statusMutex.Unlock()
			if status == ExtractStatusDone || status == ExtractStatusError {
				extractDoneOnce.Do(func() {
					close(extractDoneCh)
				})
			}
		}
	})

	// Create request to download the corrupt zip file
	req := &base.Request{
		URL: "http://" + server.Addr().String() + "/corrupt.zip",
	}

	// Create task with AutoExtract enabled
	downloadDir := tempDir + "/downloads"
	_, err = downloader.CreateDirect(req, &base.Options{
		Path: downloadDir,
		Name: "corrupt.zip",
		Extra: http.OptsExtra{
			Connections: 1,
			AutoExtract: util.BoolPtr(true),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for extraction to complete (with timeout)
	select {
	case <-extractDoneCh:
		// Extraction completed (should be error)
	case <-time.After(10 * time.Second):
		t.Log("Extraction timed out")
	}

	// Give a small buffer for final events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify extraction status changes include error
	statusMutex.Lock()
	defer statusMutex.Unlock()

	t.Logf("Recorded extract status changes: %v", extractStatusChanges)

	// Verify that we went through ExtractStatusExtracting
	foundExtracting := false
	for _, status := range extractStatusChanges {
		if status == ExtractStatusExtracting {
			foundExtracting = true
			break
		}
	}
	if !foundExtracting {
		t.Error("Expected ExtractStatusExtracting in status changes")
	}

	// Verify that we reached ExtractStatusError
	foundError := false
	for _, status := range extractStatusChanges {
		if status == ExtractStatusError {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("Expected ExtractStatusError in status changes")
	}
}

// TestExtractStatus tests the ExtractStatus constants
func TestExtractStatus(t *testing.T) {
	tests := []struct {
		status   ExtractStatus
		expected string
	}{
		{ExtractStatusNone, ""},
		{ExtractStatusQueued, "queued"},
		{ExtractStatusExtracting, "extracting"},
		{ExtractStatusDone, "done"},
		{ExtractStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("ExtractStatus %v = %q, want %q", tt.status, string(tt.status), tt.expected)
			}
		})
	}
}

// TestProgress_ExtractFields tests the ExtractStatus and ExtractProgress fields in Progress struct
func TestProgress_ExtractFields(t *testing.T) {
	progress := &Progress{
		ExtractStatus:   ExtractStatusExtracting,
		ExtractProgress: 50,
	}

	if progress.ExtractStatus != ExtractStatusExtracting {
		t.Errorf("ExtractStatus = %v, want %v", progress.ExtractStatus, ExtractStatusExtracting)
	}
	if progress.ExtractProgress != 50 {
		t.Errorf("ExtractProgress = %v, want %v", progress.ExtractProgress, 50)
	}

	// Test status transitions
	progress.ExtractStatus = ExtractStatusDone
	progress.ExtractProgress = 100
	if progress.ExtractStatus != ExtractStatusDone {
		t.Errorf("ExtractStatus after update = %v, want %v", progress.ExtractStatus, ExtractStatusDone)
	}
	if progress.ExtractProgress != 100 {
		t.Errorf("ExtractProgress after update = %v, want %v", progress.ExtractProgress, 100)
	}
}

// TestProgress_MultiPartFields tests the multi-part archive fields in Progress struct
func TestProgress_MultiPartFields(t *testing.T) {
	progress := &Progress{
		ExtractStatus:     ExtractStatusWaitingParts,
		MultiPartBaseName: "/path/to/archive.7z",
		MultiPartNumber:   1,
		MultiPartIsFirst:  true,
	}

	if progress.ExtractStatus != ExtractStatusWaitingParts {
		t.Errorf("ExtractStatus = %v, want %v", progress.ExtractStatus, ExtractStatusWaitingParts)
	}
	if progress.MultiPartBaseName != "/path/to/archive.7z" {
		t.Errorf("MultiPartBaseName = %v, want %v", progress.MultiPartBaseName, "/path/to/archive.7z")
	}
	if progress.MultiPartNumber != 1 {
		t.Errorf("MultiPartNumber = %v, want %v", progress.MultiPartNumber, 1)
	}
	if !progress.MultiPartIsFirst {
		t.Error("MultiPartIsFirst should be true")
	}

	// Test second part
	progress2 := &Progress{
		ExtractStatus:     ExtractStatusWaitingParts,
		MultiPartBaseName: "/path/to/archive.7z",
		MultiPartNumber:   2,
		MultiPartIsFirst:  false,
	}

	if progress2.MultiPartNumber != 2 {
		t.Errorf("MultiPartNumber = %v, want %v", progress2.MultiPartNumber, 2)
	}
	if progress2.MultiPartIsFirst {
		t.Error("MultiPartIsFirst should be false")
	}
}

// TestExtractStatus_WaitingParts tests the new ExtractStatusWaitingParts status
func TestExtractStatus_WaitingParts(t *testing.T) {
	if ExtractStatusWaitingParts != "waitingParts" {
		t.Errorf("ExtractStatusWaitingParts = %v, want %v", ExtractStatusWaitingParts, "waitingParts")
	}
}

// createTestArchiveWithMultipleFiles creates a test zip file with multiple files
func createTestArchiveWithMultipleFiles(path string, count int) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for i := 0; i < count; i++ {
		w, err := zipWriter.Create("test_" + strconv.Itoa(i) + ".txt")
		if err != nil {
			return err
		}
		_, err = w.Write([]byte("test content " + strconv.Itoa(i)))
		if err != nil {
			return err
		}
	}
	return nil
}

// startTestArchiveServer starts a simple HTTP server that serves a zip file
func startTestArchiveServer(zipPath string) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	go func() {
		gohttp.Serve(listener, gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
			file, err := os.Open(zipPath)
			if err != nil {
				gohttp.Error(w, err.Error(), gohttp.StatusInternalServerError)
				return
			}
			defer file.Close()

			stat, _ := file.Stat()
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
			io.Copy(w, file)
		}))
	}()

	return listener
}

// createTestArchive creates a simple test zip file for testing
func createTestArchive(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a simple zip archive
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Add a test file
	w, err := zipWriter.Create("test.txt")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("test content"))
	return err
}

func TestDownloader_Close(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}

	// Close should not error
	err := downloader.Close()
	if err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}

	// Calling Close again should not panic
	err = downloader.Close()
	if err != nil {
		t.Errorf("Close() second call unexpected error: %v", err)
	}
}

func TestDownloader_DeleteAll(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Create multiple tasks
	var wg sync.WaitGroup
	taskCount := 3
	downloadDir := t.TempDir()
	wg.Add(taskCount)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	for i := 0; i < taskCount; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		_, err := downloader.CreateDirect(req, &base.Options{
			Path: downloadDir,
			Name: test.DownloadName,
			Extra: http.OptsExtra{
				Connections: 4,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	wg.Wait()

	// Verify tasks were created
	if len(downloader.GetTasks()) != taskCount {
		t.Errorf("Expected %d tasks, got %d", taskCount, len(downloader.GetTasks()))
	}

	// Delete all tasks with nil filter
	err := downloader.Delete(nil, true)
	if err != nil {
		t.Errorf("Delete(nil) unexpected error: %v", err)
	}

	// Verify all tasks are deleted
	if len(downloader.GetTasks()) != 0 {
		t.Errorf("All tasks should be deleted, got %d remaining", len(downloader.GetTasks()))
	}
}

func TestDownloader_ProtocolConfigNotExist(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Test getting a protocol config that doesn't exist
	var unknownCfg map[string]any
	exists := downloader.getProtocolConfig("unknown-protocol", &unknownCfg)
	if exists {
		t.Errorf("getProtocolConfig() for unknown protocol should return false")
	}
}

func TestTaskFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   *TaskFilter
		expected bool
	}{
		{
			name:     "nil IDs, Statuses, NotStatuses",
			filter:   &TaskFilter{},
			expected: true,
		},
		{
			name:     "empty IDs only",
			filter:   &TaskFilter{IDs: []string{}},
			expected: true,
		},
		{
			name:     "non-empty IDs",
			filter:   &TaskFilter{IDs: []string{"id1"}},
			expected: false,
		},
		{
			name:     "non-empty Statuses",
			filter:   &TaskFilter{Statuses: []base.Status{base.DownloadStatusDone}},
			expected: false,
		},
		{
			name:     "non-empty NotStatuses",
			filter:   &TaskFilter{NotStatuses: []base.Status{base.DownloadStatusError}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsEmpty()
			if result != tt.expected {
				t.Errorf("TaskFilter.IsEmpty() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Tests for multi-part archive collection functions
func TestDownloader_CollectSequentialFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_sequential_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files for 7z multi-part pattern (archive.7z.001, .002, .003)
	for i := 1; i <= 3; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("archive.7z.%03d", i))
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	downloader := NewDownloader(nil)
	files := downloader.collectSequentialFiles(tempDir, "archive.7z", ".%03d")

	if len(files) != 3 {
		t.Errorf("collectSequentialFiles() = %d files, want 3", len(files))
	}

	// Verify files are in order
	for i, file := range files {
		expected := filepath.Join(tempDir, fmt.Sprintf("archive.7z.%03d", i+1))
		if file != expected {
			t.Errorf("files[%d] = %q, want %q", i, file, expected)
		}
	}
}

func TestDownloader_CollectSequentialFiles_NoFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_sequential_empty_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	downloader := NewDownloader(nil)
	files := downloader.collectSequentialFiles(tempDir, "archive.7z", ".%03d")

	if len(files) != 0 {
		t.Errorf("collectSequentialFiles() = %d files, want 0", len(files))
	}
}

func TestDownloader_CollectRarNewStyleFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_rar_new_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with double-digit format
	for i := 1; i <= 3; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("archive.part%02d.rar", i))
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	downloader := NewDownloader(nil)
	files := downloader.collectRarNewStyleFiles(tempDir, "archive")

	if len(files) != 3 {
		t.Errorf("collectRarNewStyleFiles() = %d files, want 3", len(files))
	}
}

func TestDownloader_CollectRarNewStyleFiles_SingleDigit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_rar_single_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with single-digit format
	for i := 1; i <= 2; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("archive.part%d.rar", i))
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	downloader := NewDownloader(nil)
	files := downloader.collectRarNewStyleFiles(tempDir, "archive")

	if len(files) != 2 {
		t.Errorf("collectRarNewStyleFiles() = %d files, want 2", len(files))
	}
}

func TestDownloader_CollectRarOldStyleFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_rar_old_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .rar file
	rarPath := filepath.Join(tempDir, "archive.rar")
	if err := os.WriteFile(rarPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .r00, .r01, .r02 files
	for i := 0; i <= 2; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("archive.r%02d", i))
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	downloader := NewDownloader(nil)
	files := downloader.collectRarOldStyleFiles(tempDir, "archive")

	// Should have 4 files: .rar + .r00 + .r01 + .r02
	if len(files) != 4 {
		t.Errorf("collectRarOldStyleFiles() = %d files, want 4", len(files))
	}

	// First file should be .rar
	if files[0] != rarPath {
		t.Errorf("First file should be .rar, got %q", files[0])
	}
}

func TestDownloader_CollectZipSplitFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_zip_split_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .z01, .z02 files
	for i := 1; i <= 2; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("archive.z%02d", i))
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create .zip file
	zipPath := filepath.Join(tempDir, "archive.zip")
	if err := os.WriteFile(zipPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	downloader := NewDownloader(nil)
	files := downloader.collectZipSplitFiles(tempDir, "archive")

	// Should have 3 files: .z01 + .z02 + .zip
	if len(files) != 3 {
		t.Errorf("collectZipSplitFiles() = %d files, want 3", len(files))
	}

	// Last file should be .zip
	if files[len(files)-1] != zipPath {
		t.Errorf("Last file should be .zip, got %q", files[len(files)-1])
	}
}

func TestDownloader_CollectMultiPartFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collect_multipart_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test with 7z pattern
	t.Run("7z pattern", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "7z")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		for i := 1; i <= 2; i++ {
			path := filepath.Join(subDir, fmt.Sprintf("archive.7z.%03d", i))
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		downloader := NewDownloader(nil)
		firstPart := filepath.Join(subDir, "archive.7z.001")
		files := downloader.collectMultiPartFiles(firstPart)

		if len(files) != 2 {
			t.Errorf("collectMultiPartFiles(7z) = %d files, want 2", len(files))
		}
	})

	// Test with RAR new style pattern
	t.Run("RAR new style pattern", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "rar_new")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		for i := 1; i <= 2; i++ {
			path := filepath.Join(subDir, fmt.Sprintf("archive.part%02d.rar", i))
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		downloader := NewDownloader(nil)
		firstPart := filepath.Join(subDir, "archive.part01.rar")
		files := downloader.collectMultiPartFiles(firstPart)

		if len(files) != 2 {
			t.Errorf("collectMultiPartFiles(RAR new) = %d files, want 2", len(files))
		}
	})

	// Test with ZIP multi-part pattern
	t.Run("ZIP multi-part pattern", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "zip")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		for i := 1; i <= 3; i++ {
			path := filepath.Join(subDir, fmt.Sprintf("archive.zip.%03d", i))
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		downloader := NewDownloader(nil)
		firstPart := filepath.Join(subDir, "archive.zip.001")
		files := downloader.collectMultiPartFiles(firstPart)

		if len(files) != 3 {
			t.Errorf("collectMultiPartFiles(ZIP) = %d files, want 3", len(files))
		}
	})
}

func TestDownloader_CheckAllMultiPartTasksDone(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "check_multipart_done_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create tasks with multi-part base name
	baseName := "archive.7z"

	// Create task 1 - done
	task1 := &Task{
		ID:     "task1",
		Status: base.DownloadStatusDone,
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: baseName + ".001"}},
			},
		},
		Progress: &Progress{},
	}
	initTask(task1)

	// Create task 2 - done
	task2 := &Task{
		ID:     "task2",
		Status: base.DownloadStatusDone,
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: baseName + ".002"}},
			},
		},
		Progress: &Progress{},
	}
	initTask(task2)

	downloader.tasks = []*Task{task1, task2}

	// All tasks are done
	basePath := filepath.Join(tempDir, baseName)
	allDone, missing := downloader.checkAllMultiPartTasksDone(basePath)
	if !allDone {
		t.Errorf("checkAllMultiPartTasksDone() = false, want true; missing: %v", missing)
	}

	// Set task2 to running
	task2.Status = base.DownloadStatusRunning
	allDone, missing = downloader.checkAllMultiPartTasksDone(basePath)
	if allDone {
		t.Error("checkAllMultiPartTasksDone() = true, want false")
	}
	if len(missing) == 0 {
		t.Error("Expected missing parts to be reported")
	}
}

func TestDownloader_CheckAllMultiPartTasksDone_NoRelatedTasks(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// No tasks exist
	allDone, missing := downloader.checkAllMultiPartTasksDone("/some/path/archive.7z")
	if allDone {
		t.Error("checkAllMultiPartTasksDone() = true, want false with no related tasks")
	}
	if len(missing) == 0 {
		t.Error("Expected missing message")
	}
}

func TestDownloader_TryClaimMultiPartExtraction(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "extraction_progress_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	baseName := "archive.7z"
	// GetMultiPartArchiveBaseName returns filepath.Join(dir, baseName)
	fullBaseName := filepath.Join(tempDir, baseName)

	// Create tasks
	task1 := &Task{
		ID:     "task1",
		Status: base.DownloadStatusDone,
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: baseName + ".001", Path: ""}},
			},
		},
		Progress: &Progress{ExtractStatus: ExtractStatusNone},
	}
	initTask(task1)

	task2 := &Task{
		ID:     "task2",
		Status: base.DownloadStatusDone,
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: baseName + ".002", Path: ""}},
			},
		},
		Progress: &Progress{ExtractStatus: ExtractStatusNone},
	}
	initTask(task2)

	downloader.tasks = []*Task{task1, task2}

	// Task1 should be able to claim extraction (no one has claimed yet)
	if !downloader.tryClaimMultiPartExtraction(task1, fullBaseName) {
		t.Error("tryClaimMultiPartExtraction() = false, want true (first claim)")
	}
	// task1's status should now be Queued
	if task1.Progress.ExtractStatus != ExtractStatusQueued {
		t.Errorf("task1.ExtractStatus = %v, want %v", task1.Progress.ExtractStatus, ExtractStatusQueued)
	}

	// Task2 should NOT be able to claim (task1 already claimed via sync.Map)
	if downloader.tryClaimMultiPartExtraction(task2, fullBaseName) {
		t.Error("tryClaimMultiPartExtraction() = true, want false (already claimed)")
	}

	// Release the claim
	downloader.releaseMultiPartExtractionClaim(fullBaseName)

	// Now task2 CAN claim (claim was released)
	if !downloader.tryClaimMultiPartExtraction(task2, fullBaseName) {
		t.Error("tryClaimMultiPartExtraction() = false, want true (claim was released)")
	}
}

func TestDownloader_HandleExtractionResult_Success(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "extraction_result_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test archive file
	archivePath := filepath.Join(tempDir, "test.zip")
	if err := os.WriteFile(archivePath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	task := &Task{
		ID: "test-task",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "test.zip"}},
			},
		},
		Progress: &Progress{},
	}
	initTask(task)

	// Test successful extraction
	downloader.handleExtractionResult(task, nil, []string{archivePath}, false)

	if task.Progress.ExtractStatus != ExtractStatusDone {
		t.Errorf("ExtractStatus = %v, want %v", task.Progress.ExtractStatus, ExtractStatusDone)
	}
	if task.Progress.ExtractProgress != 100 {
		t.Errorf("ExtractProgress = %d, want 100", task.Progress.ExtractProgress)
	}

	// Archive should still exist (deleteAfterExtract=false)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive should still exist when deleteAfterExtract=false")
	}
}

func TestDownloader_HandleExtractionResult_WithDelete(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "extraction_delete_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test archive files
	archivePath1 := filepath.Join(tempDir, "test.7z.001")
	archivePath2 := filepath.Join(tempDir, "test.7z.002")
	if err := os.WriteFile(archivePath1, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath2, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	task := &Task{
		ID: "test-task",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "test.7z.001"}},
			},
		},
		Progress: &Progress{},
	}
	initTask(task)

	// Test successful extraction with delete
	downloader.handleExtractionResult(task, nil, []string{archivePath1, archivePath2}, true)

	if task.Progress.ExtractStatus != ExtractStatusDone {
		t.Errorf("ExtractStatus = %v, want %v", task.Progress.ExtractStatus, ExtractStatusDone)
	}

	// Archives should be deleted
	if _, err := os.Stat(archivePath1); !os.IsNotExist(err) {
		t.Error("Archive 1 should be deleted when deleteAfterExtract=true")
	}
	if _, err := os.Stat(archivePath2); !os.IsNotExist(err) {
		t.Error("Archive 2 should be deleted when deleteAfterExtract=true")
	}
}

func TestDownloader_HandleExtractionResult_Error(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "extraction_error_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	task := &Task{
		ID: "test-task",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "test.zip"}},
			},
		},
		Progress: &Progress{},
	}
	initTask(task)

	// Test failed extraction
	extractErr := fmt.Errorf("extraction failed")
	downloader.handleExtractionResult(task, extractErr, nil, false)

	if task.Progress.ExtractStatus != ExtractStatusError {
		t.Errorf("ExtractStatus = %v, want %v", task.Progress.ExtractStatus, ExtractStatusError)
	}
}

func TestDownloader_UpdateMultiPartTasksStatus(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "update_status_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create source task with multi-part base name
	sourceTask := &Task{
		ID: "source",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "archive.7z.001"}},
			},
		},
		Progress: &Progress{MultiPartBaseName: "archive.7z"},
	}
	initTask(sourceTask)

	// Create related task
	relatedTask := &Task{
		ID: "related",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "archive.7z.002"}},
			},
		},
		Progress: &Progress{MultiPartBaseName: "archive.7z"},
	}
	initTask(relatedTask)

	// Create unrelated task
	unrelatedTask := &Task{
		ID: "unrelated",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "other.7z.001"}},
			},
		},
		Progress: &Progress{MultiPartBaseName: "other.7z"},
	}
	initTask(unrelatedTask)

	downloader.tasks = []*Task{sourceTask, relatedTask, unrelatedTask}

	// Test successful extraction
	downloader.updateMultiPartTasksStatus(sourceTask, nil)

	if relatedTask.Progress.ExtractStatus != ExtractStatusDone {
		t.Errorf("Related task ExtractStatus = %v, want %v", relatedTask.Progress.ExtractStatus, ExtractStatusDone)
	}
	if relatedTask.Progress.ExtractProgress != 100 {
		t.Errorf("Related task ExtractProgress = %d, want 100", relatedTask.Progress.ExtractProgress)
	}
	if unrelatedTask.Progress.ExtractStatus == ExtractStatusDone {
		t.Error("Unrelated task should not be updated")
	}
}

func TestDownloader_UpdateMultiPartTasksStatus_WithError(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "update_error_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create source task with multi-part base name
	sourceTask := &Task{
		ID: "source",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "archive.7z.001"}},
			},
		},
		Progress: &Progress{MultiPartBaseName: "archive.7z"},
	}
	initTask(sourceTask)

	// Create related task
	relatedTask := &Task{
		ID: "related",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "archive.7z.002"}},
			},
		},
		Progress: &Progress{MultiPartBaseName: "archive.7z"},
	}
	initTask(relatedTask)

	downloader.tasks = []*Task{sourceTask, relatedTask}

	// Test failed extraction
	downloader.updateMultiPartTasksStatus(sourceTask, fmt.Errorf("extraction failed"))

	if relatedTask.Progress.ExtractStatus != ExtractStatusError {
		t.Errorf("Related task ExtractStatus = %v, want %v", relatedTask.Progress.ExtractStatus, ExtractStatusError)
	}
	if relatedTask.Progress.ExtractProgress != 0 {
		t.Errorf("Related task ExtractProgress = %d, want 0", relatedTask.Progress.ExtractProgress)
	}
}

func TestDownloader_UpdateMultiPartTasksStatus_NoBaseName(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "update_no_base_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create task without multi-part base name
	task := &Task{
		ID: "single",
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: "single.zip"}},
			},
		},
		Progress: &Progress{MultiPartBaseName: ""},
	}
	initTask(task)

	downloader.tasks = []*Task{task}

	// Should not panic or error
	downloader.updateMultiPartTasksStatus(task, nil)
}

func TestDownloader_CheckMultiPartArchiveReady(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	tempDir, err := os.MkdirTemp("", "check_ready_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	baseName := "archive.7z"
	fileName := baseName + ".001"
	filePath := filepath.Join(tempDir, fileName)

	// Create tasks
	task := &Task{
		ID:     "task1",
		Status: base.DownloadStatusDone,
		Meta: &fetcher.FetcherMeta{
			Opts: &base.Options{Path: tempDir},
			Res: &base.Resource{
				Files: []*base.FileInfo{{Name: fileName}},
			},
		},
		Progress: &Progress{},
	}
	initTask(task)
	downloader.tasks = []*Task{task}

	partInfo := getArchivePartInfo(filePath)
	ready, missing := downloader.checkMultiPartArchiveReady(filePath, tempDir, partInfo)

	if !ready {
		t.Errorf("checkMultiPartArchiveReady() = false, want true; missing: %v", missing)
	}
}

func TestDownloader_CheckMultiPartArchiveReady_EmptyBaseName(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Use a non-multi-part file path
	partInfo := getArchivePartInfo("/some/regular.zip")
	ready, _ := downloader.checkMultiPartArchiveReady("/some/regular.zip", "/dest", partInfo)

	// Should return true for non-multi-part files
	if !ready {
		t.Error("checkMultiPartArchiveReady() should return true for non-multi-part files")
	}
}

// startTestTorrentServer starts a simple HTTP server that serves a torrent file
func startTestTorrentServer(torrentPath string) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	server := &gohttp.Server{
		Handler: gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
			data, err := os.ReadFile(torrentPath)
			if err != nil {
				w.WriteHeader(gohttp.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/x-bittorrent")
			w.Header().Set("Content-Disposition", "attachment; filename=ubuntu.torrent")
			w.Write(data)
		}),
	}
	go server.Serve(listener)
	return listener
}

// TestDownloader_AutoTorrent tests the auto-torrent functionality
// When a .torrent file is downloaded with AutoTorrent enabled, it should automatically create a BT task
func TestDownloader_AutoTorrent(t *testing.T) {
	// Path to the test torrent file
	torrentPath := "../../internal/protocol/bt/testdata/ubuntu-22.04-live-server-amd64.iso.torrent"
	if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
		t.Skip("Test torrent file not found, skipping test")
	}

	// Start a simple HTTP server to serve the torrent file
	server := startTestTorrentServer(torrentPath)
	defer server.Close()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "auto_torrent_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create downloader
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Delete all tasks before clearing to avoid panic from BT tasks trying to access deleted resources
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	// Track created tasks
	btTaskCreated := make(chan struct{}, 1)
	var originalTaskId string

	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyStart {
			// A new task started - if it's not the original, it's the BT task
			if event.Task != nil && event.Task.ID != originalTaskId && originalTaskId != "" {
				select {
				case btTaskCreated <- struct{}{}:
				default:
				}
			}
		}
	})

	// Create request to download the torrent file
	req := &base.Request{
		URL: "http://" + server.Addr().String() + "/ubuntu.torrent",
	}

	// Create task with AutoTorrent enabled
	downloadDir := tempDir + "/downloads"
	originalTaskId, err = downloader.CreateDirect(req, &base.Options{
		Path: downloadDir,
		Name: "ubuntu.torrent",
		Extra: http.OptsExtra{
			Connections: 1,
			AutoTorrent: util.BoolPtr(true),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Original task ID: %s", originalTaskId)

	// Wait for BT task to be created (with timeout)
	select {
	case <-btTaskCreated:
		t.Log("BT task created")
	case <-time.After(10 * time.Second):
		t.Log("Timeout waiting for BT task creation")
	}

	// Give a small buffer for task creation to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that a BT task was created
	tasks := downloader.GetTasks()

	// At minimum, we should have 2 tasks: the original torrent download and the BT task
	if len(tasks) < 2 {
		t.Errorf("Expected at least 2 tasks (torrent download + BT task), got %d", len(tasks))
	} else {
		t.Logf("Successfully created %d tasks", len(tasks))
	}
}

// TestDownloader_AutoTorrentWithDelete tests the auto-torrent with DeleteTorrentAfterDownload option
func TestDownloader_AutoTorrentWithDelete(t *testing.T) {
	// Path to the test torrent file
	torrentPath := "../../internal/protocol/bt/testdata/ubuntu-22.04-live-server-amd64.iso.torrent"
	if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
		t.Skip("Test torrent file not found, skipping test")
	}

	// Start a simple HTTP server to serve the torrent file
	server := startTestTorrentServer(torrentPath)
	defer server.Close()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "auto_torrent_delete_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create downloader
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Delete all tasks before clearing to avoid panic from BT tasks trying to access deleted resources
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	// Track task events
	var originalTaskId string
	originalTaskDeleted := make(chan struct{}, 1)
	btTaskCreated := make(chan struct{}, 1)

	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyStart {
			// BT task was created and started
			if event.Task != nil && event.Task.ID != originalTaskId && originalTaskId != "" {
				select {
				case btTaskCreated <- struct{}{}:
				default:
				}
			}
		}
		if event.Key == EventKeyDelete {
			// Check if the deleted task is the original torrent task
			if event.Task != nil && event.Task.ID == originalTaskId {
				select {
				case originalTaskDeleted <- struct{}{}:
				default:
				}
			}
		}
	})

	// Create request to download the torrent file
	req := &base.Request{
		URL: "http://" + server.Addr().String() + "/ubuntu.torrent",
	}

	// Create task with AutoTorrent and DeleteTorrentAfterDownload enabled
	downloadDir := tempDir + "/downloads"
	originalTaskId, err = downloader.CreateDirect(req, &base.Options{
		Path: downloadDir,
		Name: "ubuntu.torrent",
		Extra: http.OptsExtra{
			Connections:                1,
			AutoTorrent:                util.BoolPtr(true),
			DeleteTorrentAfterDownload: util.BoolPtr(true),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Original task ID: %s", originalTaskId)

	// Wait for original task to be deleted (this happens after BT task is created)
	select {
	case <-originalTaskDeleted:
		t.Log("Original torrent task was deleted as expected")
	case <-time.After(10 * time.Second):
		// Check manually if the task still exists
		originalTask := downloader.GetTask(originalTaskId)
		if originalTask != nil {
			t.Error("Original torrent task should have been deleted but still exists")
		} else {
			t.Log("Original torrent task was deleted (detected via GetTask)")
		}
	}

	// Give a moment for BT task creation
	time.Sleep(200 * time.Millisecond)

	// Verify that original task is deleted and BT task exists
	originalTask := downloader.GetTask(originalTaskId)
	if originalTask != nil {
		t.Error("Original torrent task should have been deleted")
	}

	// Verify remaining tasks (should have at least the BT task)
	tasks := downloader.GetTasks()
	t.Logf("Remaining tasks: %d", len(tasks))

	// At least one task should remain (the BT task)
	if len(tasks) == 0 {
		t.Error("Expected at least one task (BT task) to remain")
	}

	// None of the remaining tasks should be the original torrent task
	for _, task := range tasks {
		if task.ID == originalTaskId {
			t.Error("Original torrent task should have been deleted")
		}
	}
}

// TestDownloader_AutoTorrentDisabled tests that auto-torrent does not create BT task when disabled
func TestDownloader_AutoTorrentDisabled(t *testing.T) {
	// Path to the test torrent file
	torrentPath := "../../internal/protocol/bt/testdata/ubuntu-22.04-live-server-amd64.iso.torrent"
	if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
		t.Skip("Test torrent file not found, skipping test")
	}

	// Start a simple HTTP server to serve the torrent file
	server := startTestTorrentServer(torrentPath)
	defer server.Close()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "auto_torrent_disabled_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create downloader
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	// Track task completion
	taskDone := make(chan struct{}, 1)

	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			select {
			case taskDone <- struct{}{}:
			default:
			}
		}
	})

	// Create request to download the torrent file
	req := &base.Request{
		URL: "http://" + server.Addr().String() + "/ubuntu.torrent",
	}

	// Create task with AutoTorrent explicitly disabled
	downloadDir := tempDir + "/downloads"
	_, err = downloader.CreateDirect(req, &base.Options{
		Path: downloadDir,
		Name: "ubuntu.torrent",
		Extra: http.OptsExtra{
			Connections: 1,
			AutoTorrent: util.BoolPtr(false),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for task to complete
	select {
	case <-taskDone:
		// Task completed
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for task to complete")
	}

	// Give a small buffer
	time.Sleep(200 * time.Millisecond)

	// Verify that only 1 task exists (no BT task was created)
	tasks := downloader.GetTasks()
	if len(tasks) != 1 {
		t.Errorf("Expected exactly 1 task (torrent download only), got %d", len(tasks))
	}

	// Verify the torrent file was downloaded
	torrentFilePath := downloadDir + "/ubuntu.torrent"
	if _, err := os.Stat(torrentFilePath); os.IsNotExist(err) {
		t.Error("Torrent file should have been downloaded")
	}
}

func TestDownloader_PatchTask_HTTP(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		Extra: &http.OptsExtra{
			Connections: 2,
		},
		Labels: map[string]string{
			"test": "value1",
		},
	}
	opts := &base.Options{
		Path: t.TempDir(),
		Name: test.DownloadName,
	}

	// Create task but don't start it yet
	taskId, err := downloader.CreateDirect(req, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Pause the task immediately
	if err := downloader.Pause(&TaskFilter{IDs: []string{taskId}}); err != nil {
		t.Fatal(err)
	}

	// Patch the task with new labels
	patchReq := &base.Request{
		Labels: map[string]string{
			"test":   "value2",
			"newKey": "newValue",
		},
	}

	if err := downloader.Patch(taskId, patchReq, nil); err != nil {
		t.Fatal(err)
	}

	// Verify the patch was applied
	task := downloader.GetTask(taskId)
	if task == nil {
		t.Fatal("task not found")
	}

	if task.Meta.Req.Labels["test"] != "value2" {
		t.Errorf("PatchTask() label 'test' = %v, want %v", task.Meta.Req.Labels["test"], "value2")
	}
	if task.Meta.Req.Labels["newKey"] != "newValue" {
		t.Errorf("PatchTask() label 'newKey' = %v, want %v", task.Meta.Req.Labels["newKey"], "newValue")
	}

	// Clean up
	downloader.Delete(&TaskFilter{IDs: []string{taskId}}, true)
}

func TestDownloader_PatchTask_NotFound(t *testing.T) {
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Try to patch a non-existent task
	patchReq := &base.Request{
		Labels: map[string]string{
			"test": "value",
		},
	}

	err := downloader.Patch("non-existent-id", patchReq, nil)
	if err != ErrTaskNotFound {
		t.Errorf("Patch() error = %v, want %v", err, ErrTaskNotFound)
	}
}
