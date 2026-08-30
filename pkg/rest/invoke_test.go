package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func TestNativeInvokeWithoutRESTListener(t *testing.T) {
	cfg := &model.StartConfig{
		NativeMode: true,
		Storage:    model.StorageMem,
	}
	port, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if port != 0 {
		t.Fatalf("native runtime opened REST port %d by default", port)
	}

	var result model.Result[json.RawMessage]
	if err := json.Unmarshal([]byte(Invoke(http.MethodGet, "/api/v1/info", "", "")), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != model.CodeOk {
		t.Fatalf("invoke failed: %#v", result)
	}
}

func TestAPIServerLifecycleUsesPersistedConfigWithoutRestartingCore(t *testing.T) {
	port, err := Start(&model.StartConfig{NativeMode: true, Storage: model.StorageMem})
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	if port != 0 {
		t.Fatalf("native runtime opened REST port %d by default", port)
	}

	putAPIServerConfig(t, &base.APIServerConfig{
		Enable:  true,
		Network: "tcp",
		Address: "127.0.0.1:0",
		Token:   "secret",
	})
	state, err := StartAPIServer()
	if err != nil {
		t.Fatal(err)
	}
	port = state.RunningPort
	if port == 0 {
		t.Fatal("enabling the REST API did not open a TCP listener")
	}

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/info", port), nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Api-Token", "secret")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("REST API returned status %d", response.StatusCode)
	}

	putAPIServerConfig(t, &base.APIServerConfig{Enable: false})
	if state, err := StopAPIServer(); err != nil {
		t.Fatal(err)
	} else if state.Running || state.RunningPort != 0 || state.PendingApply {
		t.Fatalf("unexpected stopped API server state: %#v", state)
	}

	var result model.Result[json.RawMessage]
	if err := json.Unmarshal([]byte(Invoke(http.MethodGet, "/api/v1/info", "", "")), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != model.CodeOk {
		t.Fatalf("in-process API stopped with REST listener: %#v", result)
	}

	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/info", port)); err == nil {
		t.Fatal("REST listener still accepted connections after it was disabled")
	}
}

func TestRestartAPIServerFailsFastAndLeavesListenerStopped(t *testing.T) {
	if _, err := Start(&model.StartConfig{NativeMode: true, Storage: model.StorageMem}); err != nil {
		t.Fatal(err)
	}
	defer Stop()

	putAPIServerConfig(t, &base.APIServerConfig{
		Enable:  true,
		Network: "tcp",
		Address: "127.0.0.1:0",
		Token:   "secret",
	})
	startedState, err := StartAPIServer()
	if err != nil {
		t.Fatal(err)
	}
	previousPort := startedState.RunningPort
	putAPIServerConfig(t, &base.APIServerConfig{
		Enable:  true,
		Network: "unsupported-network",
		Address: "invalid",
	})
	state, err := RestartAPIServer()
	if err == nil {
		t.Fatal("invalid listener configuration unexpectedly succeeded")
	}
	if state.Running || state.RunningPort != 0 || !state.PendingApply || state.LastError == "" {
		t.Fatalf("failed restart did not leave REST listener stopped: %#v", state)
	}

	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/info", previousPort)); err == nil {
		t.Fatal("previous REST listener still accepted connections after failed restart")
	}

	var result model.Result[json.RawMessage]
	if err := json.Unmarshal([]byte(Invoke(http.MethodGet, "/api/v1/info", "", "")), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != model.CodeOk {
		t.Fatalf("in-process API unavailable after failed REST restart: %#v", result)
	}
}

func TestNativeRESTSettingPersistsInGoStorage(t *testing.T) {
	storageDir := filepath.Join(t.TempDir(), "storage")
	start := func() int {
		port, err := Start(&model.StartConfig{
			NativeMode: true,
			Storage:    model.StorageBolt,
			StorageDir: storageDir,
		})
		if err != nil {
			t.Fatal(err)
		}
		return port
	}

	if port := start(); port != 0 {
		t.Fatalf("first native start opened REST port %d", port)
	}
	config, err := Downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	config.API = newAPIServerConfigForTest()
	if err := Downloader.PutConfig(config); err != nil {
		t.Fatal(err)
	}
	Stop()

	if port := start(); port == 0 {
		Stop()
		t.Fatal("persisted REST setting did not open a listener")
	}
	Stop()
}

func TestNativeAPIServerAutoStartFailureKeepsCoreAvailable(t *testing.T) {
	storageDir := filepath.Join(t.TempDir(), "storage")
	startConfig := &model.StartConfig{
		NativeMode: true,
		Storage:    model.StorageBolt,
		StorageDir: storageDir,
	}
	if _, err := Start(startConfig); err != nil {
		t.Fatal(err)
	}
	putAPIServerConfig(t, &base.APIServerConfig{
		Enable:  true,
		Network: "unsupported-network",
		Address: "invalid",
	})
	Stop()

	port, err := Start(startConfig)
	if err != nil {
		t.Fatalf("optional REST listener failure stopped native core startup: %v", err)
	}
	defer Stop()
	if port != 0 {
		t.Fatalf("failed REST listener reported running port %d", port)
	}
	state, err := GetAPIServerState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.Enabled || state.Running || !state.PendingApply || state.LastError == "" {
		t.Fatalf("unexpected state after optional REST listener failure: %#v", state)
	}

	var result model.Result[json.RawMessage]
	if err := json.Unmarshal([]byte(Invoke(http.MethodGet, "/api/v1/info", "", "")), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != model.CodeOk {
		t.Fatalf("in-process API unavailable after REST listener failure: %#v", result)
	}
}

func newAPIServerConfigForTest() *base.APIServerConfig {
	return &base.APIServerConfig{Enable: true, Network: "tcp", Address: "127.0.0.1:0"}
}

func putAPIServerConfig(t *testing.T, apiConfig *base.APIServerConfig) {
	t.Helper()
	config, err := Downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	config.API = apiConfig
	if err := Downloader.PutConfig(config); err != nil {
		t.Fatal(err)
	}
}
