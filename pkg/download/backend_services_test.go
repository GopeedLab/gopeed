package download

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	btProtocol "github.com/GopeedLab/gopeed/internal/protocol/bt"
	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
)

type testJobManager struct {
	httpProtocol.FetcherManager
	job fetcher.CronJob
}

func (m *testJobManager) Name() string { return "test" }
func (m *testJobManager) CronJobs(fetcher.ProtocolHost) []fetcher.CronJob {
	return []fetcher.CronJob{m.job}
}
func testBackendDownloader(t *testing.T, managers ...fetcher.FetcherManager) *Downloader {
	t.Helper()
	d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: managers})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}
func TestProtocolJobCoalescesAndCancelsOnClose(t *testing.T) {
	entered := make(chan struct{})
	var calls atomic.Int32
	manager := &testJobManager{job: fetcher.CronJob{Name: "refresh", Every: time.Hour, Timeout: time.Minute, Run: func(ctx context.Context, _ fetcher.TriggerSource) error {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-ctx.Done()
		return ctx.Err()
	}}}
	d := testBackendDownloader(t, manager)
	result := make(chan error, 2)
	go func() { result <- d.TriggerProtocolJob(context.Background(), "test", "refresh") }()
	<-entered
	waiterCtx, cancel := context.WithCancel(context.Background())
	// A canceled waiter joins the existing work without canceling it.
	cancel()
	if err := d.TriggerProtocolJob(waiterCtx, "test", "refresh"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("executed %d times", calls.Load())
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := d.TriggerProtocolJob(context.Background(), "test", "missing"); !errors.Is(err, ErrProtocolJobNotFound) {
		t.Fatal(err)
	}
}
func TestProtocolJobPersistsSuccess(t *testing.T) {
	manager := &testJobManager{job: fetcher.CronJob{Name: "refresh", Every: time.Hour, Timeout: time.Minute, Run: func(context.Context, fetcher.TriggerSource) error { return nil }}}
	d := testBackendDownloader(t, manager)
	if err := d.TriggerProtocolJob(context.Background(), "test", "refresh"); err != nil {
		t.Fatal(err)
	}
	var state protocolJobState
	exists, err := d.storage.Get(bucketProtocolJobs, "test/refresh", &state)
	if err != nil || !exists || state.LastSuccess.IsZero() {
		t.Fatalf("state %+v: %v", state, err)
	}
}
func TestRaceResourceValidatesResponsesAndCancelsLosers(t *testing.T) {
	loserCanceled := make(chan struct{})
	loserStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodConnect:
			http.Error(w, "official unavailable", http.StatusBadGateway)
		case strings.HasPrefix(r.URL.Path, "/bad/"):
			io.WriteString(w, "<html>proxy landing page</html>")
		case strings.HasPrefix(r.URL.Path, "/good/"):
			<-loserStarted
			io.WriteString(w, `{"ok":true}`)
		case strings.HasPrefix(r.URL.Path, "/slow/"):
			close(loserStarted)
			<-r.Context().Done()
			close(loserCanceled)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	d := testBackendDownloader(t, &httpProtocol.FetcherManager{})
	cfg, _ := d.GetConfig()
	cfg.Proxy = &base.DownloaderProxyConfig{Enable: true, Scheme: "http", Host: strings.TrimPrefix(server.URL, "http://")}
	cfg.GithubMirror = &base.GithubMirrorConfig{Enabled: true, Mirrors: []base.GithubMirror{
		{Type: "ghProxy", URL: server.URL + "/bad"},
		{Type: "ghProxy", URL: server.URL + "/slow"},
		{Type: "ghProxy", URL: server.URL + "/good"},
	}}
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
	body, err := d.fetchResource(context.Background(), "https://raw.githubusercontent.com/owner/repo/main/test.json", 1024, false, func(body []byte) error {
		if !json.Valid(body) {
			return errors.New("invalid JSON")
		}
		return nil
	})
	if err != nil || string(body) != `{"ok":true}` {
		t.Fatalf("%s %v", body, err)
	}
	select {
	case <-loserCanceled:
	case <-time.After(time.Second):
		t.Fatal("losing request was not canceled")
	}
}

func TestProtocolConfigInitializesCurrentSchema(t *testing.T) {
	d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&httpProtocol.FetcherManager{}}})
	d.cfg.DownloaderStoreConfig = &base.DownloaderStoreConfig{
		GithubMirror:   &base.GithubMirrorConfig{Enabled: false, Mirrors: []base.GithubMirror{}},
		ProtocolConfig: map[string]any{"http": map[string]any{"connections": 3}},
	}
	if err := d.storage.Setup([]string{bucketConfig}); err != nil {
		t.Fatal(err)
	}
	defer d.storage.Close()
	if err := d.initProtocolConfigs(); err != nil {
		t.Fatal(err)
	}
	if d.cfg.GithubMirror.Enabled || len(d.cfg.GithubMirror.Mirrors) != 0 {
		t.Fatal("explicit disabled/empty mirrors lost")
	}
	value, err := d.readProtocolConfig("http")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(value)
	var cfg struct {
		Connections int `json:"connections"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.Connections != 3 {
		t.Fatalf("current protocol config: %s %v", raw, err)
	}
	if err := d.initProtocolConfigs(); err != nil {
		t.Fatal(err)
	}
}

func TestProtocolConfigDoesNotImportOldExtraSettings(t *testing.T) {
	d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&btProtocol.FetcherManager{}}})
	d.cfg.DownloaderStoreConfig = &base.DownloaderStoreConfig{
		Extra: map[string]any{
			"githubMirror": "obsolete invalid value",
			"bt":           map[string]any{"customTrackers": []string{"udp://old.example:80/announce"}, "autoUpdateTrackers": true},
		},
		ProtocolConfig: map[string]any{"bt": map[string]any{"autoUpdateTrackers": false, "listenPort": 6881}},
	}
	if err := d.storage.Setup([]string{bucketConfig}); err != nil {
		t.Fatal(err)
	}
	defer d.storage.Close()
	if err := d.initProtocolConfigs(); err != nil {
		t.Fatal(err)
	}
	if !d.cfg.GithubMirror.Enabled || len(d.cfg.GithubMirror.Mirrors) != 4 {
		t.Fatal("missing root mirror config must use new defaults")
	}
	value, err := d.readProtocolConfig("bt")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(value)
	var cfg struct {
		ListenPort         int      `json:"listenPort"`
		AutoUpdateTrackers bool     `json:"autoUpdateTrackers"`
		CustomTrackers     []string `json:"customTrackers"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.ListenPort != 6881 || cfg.AutoUpdateTrackers || len(cfg.CustomTrackers) != 0 {
		t.Fatalf("old extra settings affected protocol config: %s %v", raw, err)
	}
}

func TestProtocolJobsRunAutomaticallyAndRestoreSuccess(t *testing.T) {
	dir := t.TempDir()
	executed := make(chan fetcher.TriggerSource, 4)
	manager := func() *testJobManager {
		return &testJobManager{job: fetcher.CronJob{Name: "refresh", Every: time.Hour, Timeout: time.Second, Run: func(_ context.Context, source fetcher.TriggerSource) error { executed <- source; return nil }}}
	}
	makeDownloader := func() *Downloader {
		d := NewDownloader(&DownloaderConfig{Storage: NewBoltStorage(dir), StorageDir: dir, TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{manager()}})
		if err := d.Setup(); err != nil {
			t.Fatal(err)
		}
		return d
	}
	d := makeDownloader()
	select {
	case source := <-executed:
		if source != fetcher.TriggerScheduled {
			t.Fatal(source)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("overdue job was not run")
	}
	entry := d.protocolJobs.jobs["test/refresh"]
	entry.mu.Lock()
	run := entry.active
	entry.mu.Unlock()
	if run != nil {
		<-run.done
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	d = makeDownloader()
	defer d.Close()
	state := d.protocolJobs.jobs["test/refresh"].state
	if state.LastSuccess.IsZero() {
		t.Fatal("successful execution not restored")
	}
	select {
	case <-executed:
		t.Fatal("recently successful job ran again on restart")
	case <-time.After(30 * time.Millisecond):
	}
}
