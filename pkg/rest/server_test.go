package rest

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

var (
	restPort int

	taskReq = &base.Request{
		Extra: map[string]any{
			"method": "",
			"header": map[string]string{
				"Usr-Agent": "gopeed",
			},
			"body": "",
		},
	}
	taskRes = &base.Resource{
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
	createOpts = &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: map[string]any{
			"connections": 2,
		},
	}
	createReq = &model.CreateTask{
		Req: taskReq,
		Opt: createOpts,
	}
	createResoledReq = &model.CreateTask{
		Req: taskReq,
		Opt: createOpts,
	}
	installExtensionReq = &model.InstallExtension{
		URL: "https://github.com/GopeedLab/gopeed-extension-samples#github-contributor-avatars-sample",
	}
)

func TestInfo(t *testing.T) {
	matchKeys := []string{"version", "runtime", "os", "arch", "inDocker"}
	doTest(func() {
		resp := httpRequestCheckOk[map[string]any](http.MethodGet, "/api/v1/info", nil)
		for _, key := range matchKeys {
			if _, ok := resp[key]; !ok {
				t.Errorf("Info() missing key = %v", key)
			}
		}
	})
}

func TestResolve(t *testing.T) {
	doTest(func() {
		resp := httpRequestCheckOk[*download.ResolveResult](http.MethodPost, "/api/v1/resolve", taskReq)
		if !test.AssertResourceEqual(taskRes, resp.Res) {
			t.Errorf("Resolve() got = %v, want %v", test.ToJson(resp.Res), test.ToJson(taskRes))
		}
	})
}

func TestCreateTask(t *testing.T) {
	doTest(func() {
		resp := httpRequestCheckOk[*download.ResolveResult](http.MethodPost, "/api/v1/resolve", taskReq)

		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", &model.CreateTask{
			Rid: resp.ID,
			Opt: createOpts,
		})
		if taskId == "" {
			t.Fatal("create task failed")
		}

		wg.Wait()
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("CreateTask() got = %v, want %v", got, want)
		}
	})
}

func TestCreateDirectTask(t *testing.T) {
	doTest(func() {
		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		if taskId == "" {
			t.Fatal("create task failed")
		}

		wg.Wait()
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("CreateDirectTask() got = %v, want %v", got, want)
		}
	})
}

func TestCreateDirectTaskBatch(t *testing.T) {
	doTest(func() {
		reqs := make([]*base.Request, 0)
		for i := 0; i < 5; i++ {
			reqs = append(reqs, createReq.Req)
		}
		taskIds := httpRequestCheckOk[[]string](http.MethodPost, "/api/v1/tasks/batch", &model.CreateTaskBatch{
			Reqs: reqs,
		})
		if len(taskIds) != len(reqs) {
			t.Errorf("CreateDirectTaskBatch() got = %v, want %v", len(taskIds), len(reqs))
		}
	})
}

func TestCreateDirectTaskWithResoled(t *testing.T) {
	doTest(func() {
		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createResoledReq)
		if taskId == "" {
			t.Fatal("create task failed")
		}

		wg.Wait()
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("CreateDirectTaskWithResoled() got = %v, want %v", got, want)
		}
	})
}

func TestPauseAndContinueTask(t *testing.T) {
	doTest(func() {
		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			switch event.Key {
			case download.EventKeyFinally:
				wg.Done()
			}
		})

		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		t1 := httpRequestCheckOk[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		if t1.Status != base.DownloadStatusRunning {
			t.Errorf("CreateTask() got = %v, want %v", t1.Status, base.DownloadStatusRunning)
		}
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/tasks/"+taskId+"/pause", nil)
		t2 := httpRequestCheckOk[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		if t2.Status != base.DownloadStatusPause {
			t.Errorf("PauseTask() got = %v, want %v", t2.Status, base.DownloadStatusPause)
		}
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/tasks/"+taskId+"/continue", nil)
		t3 := httpRequestCheckOk[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		if t3.Status != base.DownloadStatusRunning {
			t.Errorf("ContinueTask() got = %v, want %v", t3.Status, base.DownloadStatusRunning)
		}

		wg.Wait()
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("PauseAndContinueTask() got = %v, want %v", got, want)
		}
	})
}

func TestPauseAllAndContinueALLTasks(t *testing.T) {
	doTest(func() {
		cfg, err := Downloader.GetConfig()
		if err != nil {
			t.Fatal(err)
		}

		createAndPause := func() {
			taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
			httpRequestCheckOk[*download.Task](http.MethodPut, "/api/v1/tasks/"+taskId+"/pause", nil)
		}

		total := cfg.MaxRunning + 2
		for i := 0; i < total; i++ {
			createAndPause()
		}

		// continue all
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/tasks/continue", nil)
		time.Sleep(time.Millisecond * 100)
		tasks := httpRequestCheckOk[[]*download.Task](http.MethodGet, fmt.Sprintf("/api/v1/tasks?status=%s", base.DownloadStatusRunning), nil)
		if len(tasks) != cfg.MaxRunning {
			t.Errorf("ContinueAllTasks() got = %v, want %v", len(tasks), cfg.MaxRunning)
		}
		// pause all
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/tasks/pause", nil)
		time.Sleep(time.Millisecond * 100)
		tasks = httpRequestCheckOk[[]*download.Task](http.MethodGet, fmt.Sprintf("/api/v1/tasks?status=%s", base.DownloadStatusPause), nil)
		if len(tasks) != total {
			t.Errorf("PauseAllTasks() got = %v, want %v", len(tasks), total)
		}
	})
}

func TestDeleteTask(t *testing.T) {
	doTest(func() {
		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		time.Sleep(time.Millisecond * 200)
		httpRequestCheckOk[any](http.MethodDelete, "/api/v1/tasks/"+taskId, nil)
		code, _ := httpRequest[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		checkCode(code, model.CodeTaskNotFound)
	})
}

func TestDeleteTaskForce(t *testing.T) {
	doTest(func() {
		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		time.Sleep(time.Millisecond * 200)
		httpRequestCheckOk[any](http.MethodDelete, "/api/v1/tasks/"+taskId+"?force=true", nil)
		code, _ := httpRequest[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		checkCode(code, model.CodeTaskNotFound)
		if _, err := os.Stat(test.DownloadFile); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("DeleteTaskForce() got = %v, want %v", err, os.ErrNotExist)
		}
	})
}

func TestDeleteAllTasks(t *testing.T) {
	doTest(func() {
		taskCount := 3

		var wg sync.WaitGroup
		wg.Add(taskCount)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		for i := 0; i < taskCount; i++ {
			httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		}

		wg.Wait()

		httpRequestCheckOk[any](http.MethodDelete, "/api/v1/tasks?force=true", nil)
		tasks := httpRequestCheckOk[[]*download.Task](http.MethodGet, "/api/v1/tasks", nil)
		if len(tasks) != 0 {
			t.Errorf("DeleteTasks() got = %v, want %v", len(tasks), 0)
		}
	})
}

func TestDeleteTasksByStatues(t *testing.T) {
	doTest(func() {
		taskCount := 3

		var wg sync.WaitGroup
		wg.Add(taskCount)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		for i := 0; i < taskCount; i++ {
			httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		}

		wg.Wait()

		httpRequestCheckOk[any](http.MethodDelete, fmt.Sprintf("/api/v1/tasks?status=%s&force=true", base.DownloadStatusDone), nil)
		tasks := httpRequestCheckOk[[]*download.Task](http.MethodGet, "/api/v1/tasks", nil)
		if len(tasks) != 0 {
			t.Errorf("DeleteTasks() got = %v, want %v", len(tasks), 0)
		}
	})
}

func TestGetTasks(t *testing.T) {
	doTest(func() {
		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		httpRequestCheckOk[string](http.MethodPost, fmt.Sprintf("/api/v1/tasks?status=%s&status=%s",
			base.DownloadStatusReady, base.DownloadStatusRunning), createReq)
		httpRequestCheckOk[[]*download.Task](http.MethodGet, "/api/v1/tasks", nil)

		wg.Wait()
		r := httpRequestCheckOk[[]*download.Task](http.MethodGet, fmt.Sprintf("/api/v1/tasks?status=%s",
			base.DownloadStatusDone), nil)
		if r[0].Status != base.DownloadStatusDone {
			t.Errorf("GetTasks() got = %v, want %v", r[0].Status, base.DownloadStatusDone)
		}
		r = httpRequestCheckOk[[]*download.Task](http.MethodGet, fmt.Sprintf("/api/v1/tasks?status=%s,%s",
			base.DownloadStatusReady, base.DownloadStatusRunning), nil)
		if len(r) > 0 {
			t.Errorf("GetTasks() got = %v, want %v", len(r), 0)
		}
	})
}

func TestGetAndPutConfig(t *testing.T) {
	doTest(func() {
		cfg := httpRequestCheckOk[*base.DownloaderStoreConfig](http.MethodGet, "/api/v1/config", nil)
		cfg.DownloadDir = "./download"
		cfg.Extra = map[string]any{
			"serverConfig": &Config{
				Host: "127.0.0.1",
				Port: 8080,
			},
			"theme": "dark",
		}
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/config", cfg)

		newCfg := httpRequestCheckOk[*base.DownloaderStoreConfig](http.MethodGet, "/api/v1/config", nil)
		if !test.JsonEqual(cfg, newCfg) {
			t.Errorf("GetAndPutConfig() got = %v, want %v", test.ToJson(newCfg), test.ToJson(cfg))
		}
	})
}

func TestInstallExtension(t *testing.T) {
	doTest(func() {
		identity := httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)
		if identity == "" {
			t.Errorf("InstallExtension() got = %v, want %v", identity, "not empty")
		}

		// not a valid extension repository
		code, _ := httpRequest[string](http.MethodPost, "/api/v1/extensions", &model.InstallExtension{
			URL: "https://github.com/GopeedLab/gopeed",
		})
		checkCode(code, model.CodeError)

		// not a git repository
		code, _ = httpRequest[string](http.MethodPost, "/api/v1/extensions", &model.InstallExtension{
			URL: "https://github.com",
		})
		checkCode(code, model.CodeError)
	})
}

func TestGetExtensions(t *testing.T) {
	doTest(func() {
		httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)
		extensions := httpRequestCheckOk[[]*download.Extension](http.MethodGet, "/api/v1/extensions", nil)
		if len(extensions) == 0 {
			t.Errorf("GetExtensions() got = %v, want %v", len(extensions), "not empty")
		}
	})
}

func TestUpdateExtensionSettings(t *testing.T) {
	doTest(func() {
		identity := httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)

		httpRequestCheckOk[any](http.MethodPut, "/api/v1/extensions/"+identity+"/settings", &model.UpdateExtensionSettings{
			Settings: map[string]any{
				"undefined": "test",
				"ua":        "test",
			},
		})

		settings := httpRequestCheckOk[*download.Extension](http.MethodGet, "/api/v1/extensions/"+identity, nil).Settings
		if len(settings) != 1 {
			t.Errorf("UpdateExtensionSettings() got = %v, want %v", len(settings), 1)
		}

		if settings[0].Name != "ua" || settings[0].Value != "test" {
			t.Errorf("UpdateExtensionSettings() got = %v, want %v", settings[0].Value, "test")
		}
	})
}

func TestSwitchExtension(t *testing.T) {
	doTest(func() {
		identity := httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/extensions/"+identity+"/switch", &model.SwitchExtension{
			Status: false,
		})
		extensions := httpRequestCheckOk[[]*download.Extension](http.MethodGet, "/api/v1/extensions", nil)
		if !extensions[0].Disabled {
			t.Errorf("TestSwitchExtension() got = %v, want %v", extensions[0].Disabled, true)
		}
	})
}

func TestDeleteExtension(t *testing.T) {
	doTest(func() {
		identity := httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)
		httpRequestCheckOk[any](http.MethodDelete, "/api/v1/extensions/"+identity, nil)
		extensions := httpRequestCheckOk[[]*download.Extension](http.MethodGet, "/api/v1/extensions", nil)
		if len(extensions) != 0 {
			t.Errorf("TestDeleteExtension() got = %v, want %v", len(extensions), 0)
		}
	})
}

func TestUpdateCheckExtension(t *testing.T) {
	doTest(func() {
		identity := httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)
		resp := httpRequestCheckOk[*model.UpdateCheckExtensionResp](http.MethodGet, "/api/v1/extensions/"+identity+"/update", nil)
		// no new version
		if resp.NewVersion != "" {
			t.Errorf("UpdateCheckExtension() got = %v, want %v", resp.NewVersion, "")
		}
		// force update
		httpRequestCheckOk[any](http.MethodPost, "/api/v1/extensions/"+identity+"/update", nil)
	})
}

func TestFsExtension(t *testing.T) {
	doTest(func() {
		identity := httpRequestCheckOk[string](http.MethodPost, "/api/v1/extensions", installExtensionReq)
		statusCode, _ := doHttpRequest0(http.MethodGet, "/fs/extensions/"+identity+"/icon.png", nil, nil)
		if statusCode != http.StatusOK {
			t.Errorf("FsExtension() got = %v, want %v", statusCode, http.StatusOK)
		}
	})
}

func TestFsExtensionFail(t *testing.T) {
	doTest(func() {
		statusCode, _ := doHttpRequest0(http.MethodGet, "/fs/extensions/not_exist/icon.png", nil, nil)
		if statusCode != http.StatusNotFound {
			t.Errorf("TestFsExtensionFail() got = %v, want %v", statusCode, http.StatusNotFound)
		}
	})
}

func TestDoProxy(t *testing.T) {
	doTest(func() {
		code, respBody := doHttpRequest0(http.MethodGet, "/api/v1/proxy", map[string]string{
			"X-Target-Uri": "https://github.com/GopeedLab/gopeed/raw/695da7ea87d2b455552b709d3cb4d7879484d4d1/README.md",
		}, nil)
		if code != http.StatusOK {
			t.Errorf("DoProxy() got = %v, want %v", code, http.StatusOK)
		}
		want := "4ee193b676f1ebb2ad810e016350d52a"
		got := fmt.Sprintf("%x", md5.Sum(respBody))
		if got != want {
			t.Errorf("DoProxy() got = %v, want %v", got, want)
		}
	})

	doTest(func() {
		code, _ := doHttpRequest0(http.MethodGet, "/api/v1/proxy", map[string]string{
			"X-Target-Uri": "https://github.com/GopeedLab/gopeed/raw/695da7ea87d2b455552b709d3cb4d7879484d4d1/NOT_FOUND",
		}, nil)
		if code != http.StatusNotFound {
			t.Errorf("DoProxy() got = %v, want %v", code, http.StatusNotFound)
		}
	})
}

func TestApiToken(t *testing.T) {
	var cfg = &model.StartConfig{}
	cfg.Init()
	cfg.ApiToken = "123456"
	fileListener := doStart(cfg)
	defer func() {
		if err := fileListener.Close(); err != nil {
			panic(err)
		}
		Stop()
	}()

	status, _ := doHttpRequest0(http.MethodGet, "/api/v1/config", nil, nil)
	if status != http.StatusUnauthorized {
		t.Errorf("TestApiToken() got = %v, want %v", status, http.StatusUnauthorized)
	}

	status, _ = doHttpRequest0(http.MethodGet, "/api/v1/config", map[string]string{
		"X-Api-Token": cfg.ApiToken,
	}, nil)
	if status != http.StatusOK {
		t.Errorf("TestApiToken() got = %v, want %v", status, http.StatusOK)
	}

}

func TestAuthorization(t *testing.T) {
	var cfg = &model.StartConfig{}
	cfg.Init()
	cfg.ApiToken = "123456"
	cfg.WebEnable = true
	cfg.WebBasicAuth = &model.WebBasicAuth{
		Username: "admin",
		Password: "123456",
	}
	fileListener := doStart(cfg)
	defer func() {
		if err := fileListener.Close(); err != nil {
			panic(err)
		}
		Stop()
	}()

	status, _ := doHttpRequest0(http.MethodGet, "/api/v1/config", nil, nil)
	if status != http.StatusUnauthorized {
		t.Errorf("TestAuthorization() got = %v, want %v", status, http.StatusUnauthorized)
	}

	status, _ = doHttpRequest0(http.MethodGet, "/api/v1/config", map[string]string{
		"Authorization": cfg.WebBasicAuth.Authorization(),
	}, nil)
	if status != http.StatusOK {
		t.Errorf("TestAuthorization() got = %v, want %v", status, http.StatusOK)
	}

	status, _ = doHttpRequest0(http.MethodGet, "/api/v1/config", map[string]string{
		"X-Api-Token": cfg.ApiToken,
	}, nil)
	if status != http.StatusOK {
		t.Errorf("TestAuthorization() got = %v, want %v", status, http.StatusOK)
	}
}

func doTest(handler func()) {
	testFunc := func(storage model.Storage) {
		var cfg = &model.StartConfig{}
		cfg.Init()
		cfg.Storage = storage
		cfg.StorageDir = ".test_storage"
		cfg.WebEnable = true
		fileListener := doStart(cfg)
		defer func() {
			if err := fileListener.Close(); err != nil {
				panic(err)
			}
			Stop()
			Downloader.Clear()
		}()
		defer func() {
			Downloader.PauseAll()
			Downloader.DeleteByStatues(nil, true)
			os.RemoveAll(cfg.StorageDir)
		}()
		taskReq.URL = "http://" + fileListener.Addr().String() + "/" + test.BuildName
		handler()
	}
	testFunc(model.StorageMem)
	testFunc(model.StorageBolt)
}

func doStart(cfg *model.StartConfig) net.Listener {
	port, err := Start(cfg)
	if err != nil {
		panic(err)
	}
	restPort = port
	return test.StartTestFileServer()
}

func doHttpRequest0(method string, path string, headers map[string]string, body any) (int, []byte) {
	var reader io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		reader = bytes.NewBuffer(buf)
	}

	request, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", restPort, path), reader)
	if err != nil {
		panic(err)
	}
	if headers != nil {
		for k, v := range headers {
			request.Header.Set(k, v)
		}
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	return response.StatusCode, respBody
}

func doHttpRequest[T any](method string, path string, headers map[string]string, body any) (int, *model.Result[T]) {
	statusCode, respBody := doHttpRequest0(method, path, headers, body)
	if statusCode != http.StatusOK {
		panic(fmt.Sprintf("http request failed, status code: %d", statusCode))
	}

	var r model.Result[T]
	if err := json.Unmarshal(respBody, &r); err != nil {
		panic(err)
	}
	return int(r.Code), &r
}

func httpRequest[T any](method string, path string, body any) (int, *model.Result[T]) {
	return doHttpRequest[T](method, path, nil, body)
}

func httpRequestCheckOk[T any](method string, path string, body any) T {
	code, result := httpRequest[T](method, path, body)
	checkOk(code)
	return result.Data
}

func checkOk(code int) {
	checkCode(code, model.CodeOk)
}

func checkCode(code int, exceptCode model.RespCode) {
	if code != int(exceptCode) {
		panic(fmt.Sprintf("code got = %d, want %d", code, exceptCode))
	}
}
