package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	restmodel "github.com/GopeedLab/gopeed/pkg/rest/model"
)

func TestResolveThenCreateWithDownloadOptions(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="payload.txt"`)
		w.Header().Set("Content-Length", "4")
		_, _ = w.Write([]byte("test"))
	}))
	t.Cleanup(upstream.Close)
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	// Resolving must work even when the default download directory is unusable.
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	current, err := d.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	config := *current
	config.DownloadDir = blocked
	config.MaxRunning = 0 // Inspect the created task before it starts downloading.
	if err := d.PutConfig(&config); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	resolved := serviceTestRequest[*download.ResolveResult](t, service, http.MethodPost, "/api/v1/resolve", &restmodel.ResolveTask{
		Req: &base.Request{URL: upstream.URL},
	})
	if resolved.ID == "" || resolved.Res.Size != 4 || resolved.Res.Files[0].Name != "payload.txt" {
		t.Fatalf("unexpected resolved resource: %+v", resolved)
	}
	dir := filepath.Join(t.TempDir(), "chosen-after-resolve")
	opts := &base.Options{Path: dir, Name: "renamed.txt", SelectFiles: []int{0}}
	id := serviceTestRequest[string](t, service, http.MethodPost, "/api/v1/tasks", &restmodel.CreateTask{Rid: resolved.ID, Opts: opts})
	task := serviceTestRequest[*download.Task](t, service, http.MethodGet, "/api/v1/tasks/"+id, nil)
	if task.Meta.Opts.Path != dir || task.Meta.Opts.Name != opts.Name || !reflect.DeepEqual(task.Meta.Opts.SelectFiles, opts.SelectFiles) {
		t.Fatalf("creation lost submitted download options: %+v", task.Meta.Opts)
	}
}

func serviceTestRequest[T any](t *testing.T, service *Service, method, path string, body any) T {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	status, response := service.JSON(method, path, nil, encoded, nil)
	var result restmodel.Result[T]
	if err := json.Unmarshal(response, &result); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || result.Code != restmodel.CodeOk {
		t.Fatalf("%s %s failed: status=%d response=%s", method, path, status, response)
	}
	return result.Data
}

func TestFullConfigDispatch(t *testing.T) {
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&httpProtocol.FetcherManager{}}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	response := service.Dispatch(http.MethodPut, "/api/v1/config", nil, []byte(`{"protocolConfig":{"http":{"connections":3,"useServerCtime":false}}}`), nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%+v", response)
	}
	status, body := service.JSON(http.MethodGet, "/api/v1/config", nil, nil, nil)
	var result struct {
		Data struct {
			ProtocolConfig map[string]struct {
				Connections int `json:"connections"`
			} `json:"protocolConfig"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || result.Data.ProtocolConfig["http"].Connections != 3 {
		t.Fatalf("%d %s", status, body)
	}
	if response := service.Dispatch(http.MethodPost, "/api/v1/protocols/http/jobs/missing/trigger", nil, nil, nil); response.StatusCode != http.StatusNotFound {
		t.Fatalf("%+v", response)
	}
}

func TestNativePathParametersAreDecodedOnce(t *testing.T) {
	route := newRoute(http.MethodGet, "/api/v1/tasks/{id}/status", func(c *Context) *Response { return newOKResponse(c.PathParam("id")) })
	for _, test := range []struct{ path, want string }{
		{"/api/v1/tasks/%E5%BC%80%E5%8F%91%E8%80%85%40name/status", "开发者@name"},
		{"/api/v1/tasks/literal%252Fname/status", "literal%2Fname"},
	} {
		response := (&Service{routes: []RouteSpec{route}}).Dispatch(http.MethodGet, test.path, nil, nil, nil)
		raw, _ := json.Marshal(response.Body)
		var body struct {
			Data string `json:"data"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.Data != test.want {
			t.Fatalf("%s => %q", test.path, body.Data)
		}
	}
}

func TestMirrorCandidates(t *testing.T) {
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&httpProtocol.FetcherManager{}}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	cfg, _ := d.GetConfig()
	cfg.GithubMirror = &base.GithubMirrorConfig{Enabled: true, Mirrors: []base.GithubMirror{
		{Type: "jsdelivr", URL: "https://cdn.example/gh"},
		{Type: "ghProxy", URL: "https://proxy.example"},
		{Type: "ghProxy", URL: "https://proxy.example"},
	}}
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, official := range []string{
		"https://github.com/GopeedLab/gopeed/releases/download/v2.0.0/package.zip",
		"https://raw.githubusercontent.com/owner/repo/main/trackers.txt",
	} {
		request, _ := json.Marshal(map[string]string{"url": official})
		status, raw := service.JSON(http.MethodPost, "/api/v1/mirrors/github/candidates", nil, request, nil)
		var body struct {
			Data struct {
				URLs []string `json:"urls"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		want := 2
		if official == "https://raw.githubusercontent.com/owner/repo/main/trackers.txt" {
			want = 3
		}
		urls := body.Data.URLs
		if status != http.StatusOK || len(urls) != want || urls[0] != official || urls[len(urls)-1] != "https://proxy.example/"+official {
			t.Fatalf("%d %s", status, raw)
		}
	}
	for _, invalid := range []string{"https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=10", "", "https://example.com/x", "http://github.com/x", "https://github.com.evil.test/x", "https://user@github.com/x", "https://github.com:443/x", "https://github.com:/x", "https://github.com/x#fragment", "https://github.com/x#", "/relative", "https://github.com%2fevil.test/x"} {
		request, _ := json.Marshal(map[string]string{"url": invalid})
		if response := service.Dispatch(http.MethodPost, "/api/v1/mirrors/github/candidates", nil, request, nil); response.StatusCode != http.StatusBadRequest {
			t.Fatalf("accepted %s: %+v", invalid, response)
		}
	}
	cfg.GithubMirror.Enabled = false
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
	status, raw := service.JSON(http.MethodPost, "/api/v1/mirrors/github/candidates", nil, []byte(`{"url":"https://github.com/owner/repo"}`), nil)
	var body struct {
		Data struct {
			URLs []string `json:"urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || status != http.StatusOK || len(body.Data.URLs) != 1 || body.Data.URLs[0] != "https://github.com/owner/repo" {
		t.Fatalf("%d %s", status, raw)
	}

	for _, provider := range []string{"unknown", "GitHub"} {
		if response := service.Dispatch(http.MethodPost, "/api/v1/mirrors/"+provider+"/candidates", nil, []byte(`{"url":"https://github.com/owner/repo"}`), nil); response.StatusCode != http.StatusNotFound {
			t.Fatalf("unknown provider accepted: %+v", response)
		}
	}
}

type apiJobState struct {
	Runs int `json:"runs"`
}
type apiJobConfig struct {
	fetcher.ConfigState[apiJobState]
	Writable int `json:"writable"`
}
type apiStateJobManager struct{ httpProtocol.FetcherManager }

func (*apiStateJobManager) Name() string       { return "state-api" }
func (*apiStateJobManager) DefaultConfig() any { return &apiJobConfig{Writable: 7} }
func (*apiStateJobManager) CronJobs(host fetcher.ProtocolHost) []fetcher.CronJob {
	return []fetcher.CronJob{{Name: "refresh", Every: time.Hour, Timeout: time.Second, Run: func(_ context.Context, source fetcher.TriggerSource) error {
		if source == fetcher.TriggerScheduled {
			return fetcher.ErrJobSkipped
		}
		return fetcher.UpdateState[apiJobConfig, apiJobState](host, func(cfg *apiJobConfig, state *apiJobState) error { cfg.Writable = 999; state.Runs++; return nil })
	}}}
}

func TestFullConfigAndJobTriggerShareProtectedProtocolState(t *testing.T) {
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&apiStateJobManager{}}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	put := []byte(`{"protocolConfig":{"state-api":{"writable":9,"state":"ignore forged state"}}}`)
	if response := service.Dispatch(http.MethodPut, "/api/v1/config", nil, put, nil); response.StatusCode != http.StatusOK {
		t.Fatalf("%+v", response)
	}
	if response := service.Dispatch(http.MethodPost, "/api/v1/protocols/state-api/jobs/refresh/trigger", nil, nil, nil); response.StatusCode != http.StatusOK {
		t.Fatalf("%+v", response)
	}
	status, raw := service.JSON(http.MethodGet, "/api/v1/config", nil, nil, nil)
	var response struct {
		Data struct {
			ProtocolConfig map[string]apiJobConfig `json:"protocolConfig"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	cfg := response.Data.ProtocolConfig["state-api"]
	if status != http.StatusOK || cfg.Writable != 9 || cfg.State.Runs != 1 {
		t.Fatalf("%d %s", status, raw)
	}
	// Replaying an old full PUT must not undo the manual job's state change.
	if response := service.Dispatch(http.MethodPut, "/api/v1/config", nil, put, nil); response.StatusCode != http.StatusOK {
		t.Fatalf("%+v", response)
	}
	_, raw = service.JSON(http.MethodGet, "/api/v1/config", nil, nil, nil)
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.ProtocolConfig["state-api"].State.Runs != 1 {
		t.Fatalf("job state overwritten: %s", raw)
	}
}
