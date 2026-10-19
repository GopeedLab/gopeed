package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
