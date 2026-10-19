package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/download"
)

func TestFullConfigDispatchAndRetiredProtocolConfigRoutes(t *testing.T) {
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
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodPut} {
		if response := service.Dispatch(method, "/api/v1/protocols/http/config", nil, nil, nil); response.StatusCode != http.StatusNotFound {
			t.Fatalf("retired %s route still accessible: %+v", method, response)
		}
	}
	if response := service.Dispatch(http.MethodPost, "/api/v1/protocols/http/jobs/missing/trigger", nil, nil, nil); response.StatusCode != http.StatusNotFound {
		t.Fatalf("%+v", response)
	}
}

func TestNativePathParametersAreDecodedOnce(t *testing.T) {
	route := newRoute(http.MethodGet, "/api/v1/store/extensions/{id}", func(c *Context) *Response { return newOKResponse(c.PathParam("id")) })
	for _, test := range []struct{ path, want string }{
		{"/api/v1/store/extensions/%E5%BC%80%E5%8F%91%E8%80%85%40name", "开发者@name"},
		{"/api/v1/store/extensions/literal%252Fname", "literal%2Fname"},
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

func TestAppUpdateCandidatesDispatch(t *testing.T) {
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&httpProtocol.FetcherManager{}}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	status, raw := service.JSON(http.MethodPost, "/api/v1/app/updates/candidates", nil, []byte(`{"version":"1.2.3","asset":"Gopeed-v1.2.3-macos-arm64.dmg"}`), nil)
	var body struct {
		Data download.AppUpdateCandidates `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || len(body.Data.URLs) == 0 || body.Data.URLs[0] != "https://github.com/GopeedLab/gopeed/releases/download/v1.2.3/Gopeed-v1.2.3-macos-arm64.dmg" {
		t.Fatalf("%d %s", status, raw)
	}
	response := service.Dispatch(http.MethodPost, "/api/v1/app/updates/candidates", nil, []byte(`{"version":"1.2.3","asset":"https://example.com/anything"}`), nil)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("arbitrary resource accepted: %+v", response)
	}
	if response := service.Dispatch(http.MethodPost, "/api/v1/app/updates/prepare", nil, nil, nil); response.StatusCode != http.StatusNotFound {
		t.Fatalf("retired preparation route: %+v", response)
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
