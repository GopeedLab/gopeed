package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/monkeyWie/gopeed/internal/test"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/download"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
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

	taskReq = &base.Request{}
	taskRes = &base.Resource{
		Name:  test.BuildName,
		Req:   taskReq,
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
	createReq = &model.CreateTask{
		Res: taskRes,
		Opts: &base.Options{
			Path:        test.Dir,
			Name:        test.DownloadName,
			Connections: 2,
		},
	}
)

func TestResolve(t *testing.T) {
	doTest(func() {
		resp := httpRequestCheckOk[*base.Resource](http.MethodPost, "/api/v1/resolve", taskReq)
		if !reflect.DeepEqual(taskRes, resp) {
			t.Errorf("Resolve() got = %v, want %v", resp, taskRes)
		}
	})
}

func TestCreateTask(t *testing.T) {
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
			t.Errorf("CreateTask() got = %v, want %v", got, want)
		}
	})
}

func TestPauseAndContinueTask(t *testing.T) {
	doTest(func() {
		type result struct {
			pauseCount    int
			continueCount int
			md5           string
		}

		var got = result{
			pauseCount:    0,
			continueCount: 0,
			md5:           "",
		}
		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			switch event.Key {
			case download.EventKeyPause:
				got.pauseCount++
			case download.EventKeyContinue:
				got.continueCount++
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
		want := result{
			pauseCount:    1,
			continueCount: 1,
			md5:           test.FileMd5(test.BuildFile),
		}
		got.md5 = test.FileMd5(test.DownloadFile)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("PauseAndContinueTask() got = %v, want %v", got, want)
		}
	})
}

func TestDeleteTask(t *testing.T) {
	doTest(func() {
		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		httpRequestCheckOk[any](http.MethodDelete, "/api/v1/tasks/"+taskId, nil)
		code, _ := httpRequest[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		checkCode(code, http.StatusNotFound)
	})
}

func TestDeleteTaskForce(t *testing.T) {
	doTest(func() {
		taskId := httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		time.Sleep(time.Millisecond * 500)
		httpRequestCheckOk[any](http.MethodDelete, "/api/v1/tasks/"+taskId+"?force=true", nil)
		code, _ := httpRequest[*download.Task](http.MethodGet, "/api/v1/tasks/"+taskId, nil)
		checkCode(code, http.StatusNotFound)
		if _, err := os.Stat(test.DownloadFile); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("DeleteTaskForce() got = %v, want %v", err, os.ErrNotExist)
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

		httpRequestCheckOk[string](http.MethodPost, fmt.Sprintf("/api/v1/tasks?status=%s,%s",
			base.DownloadStatusReady, base.DownloadStatusRunning), createReq)
		code, _ := httpRequest[[]*download.Task](http.MethodGet, "/api/v1/tasks", nil)
		checkCode(code, http.StatusBadRequest)

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
		cfg := httpRequestCheckOk[*model.ServerConfig](http.MethodGet, "/api/v1/config", nil)

		cfg.Port = 8888
		cfg.Connections = 32
		cfg.DownloadDir = "./download"
		cfg.Extra = map[string]any{
			"theme": "dark",
		}
		httpRequestCheckOk[any](http.MethodPut, "/api/v1/config", cfg)

		newCfg := httpRequestCheckOk[*model.ServerConfig](http.MethodGet, "/api/v1/config", nil)
		if !reflect.DeepEqual(newCfg, cfg) {
			t.Errorf("GetAndPutConfig() got = %v, want %v", newCfg, cfg)
		}
	})
}

func doTest(handler func()) {
	testFunc := func(storage model.Storage) {
		fileListener := doStart(storage)
		defer func() {
			if err := fileListener.Close(); err != nil {
				panic(err)
			}
			Stop()
		}()
		defer Downloader.Clear()
		taskReq.URL = "http://" + fileListener.Addr().String() + "/" + test.BuildName
		handler()
	}
	testFunc(model.StorageMem)
	testFunc(model.StorageBolt)
}

func doStart(storage model.Storage) net.Listener {
	port, err := Start((&model.StartConfig{
		Storage: storage,
	}).Init())
	if err != nil {
		panic(err)
	}
	restPort = port
	return test.StartTestFileServer()
}

func httpRequest[T any](method string, path string, req any) (int, *model.Result[T]) {
	var body io.Reader
	if req != nil {
		buf, _ := json.Marshal(req)
		body = bytes.NewBuffer(buf)
	}

	request, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", restPort, path), body)
	if err != nil {
		panic(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var r model.Result[T]
	if err := json.NewDecoder(response.Body).Decode(&r); err != nil {
		panic(err)
	}
	return response.StatusCode, &r
}

func httpRequestCheckOk[T any](method string, path string, req any) T {
	code, result := httpRequest[T](method, path, req)
	checkOk(code)
	return result.Data
}

func checkOk(code int) {
	checkCode(code, 200)
}

func checkCode(code int, exceptCode int) {
	if code != exceptCode {
		panic(fmt.Sprintf("code got = %d, want %d", code, exceptCode))
	}
}
