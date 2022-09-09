package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"github.com/monkeyWie/gopeed-core/pkg/test"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
)

var (
	restPort int

	taskReq = &base.Request{}
	taskRes = &base.Resource{
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
	createReq = &model.CreateTaskReq{
		Resource: taskRes,
		Options: &base.Options{
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

func TestGetTasks(t *testing.T) {
	doTest(func() {
		var wg sync.WaitGroup
		wg.Add(1)
		Downloader.Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				wg.Done()
			}
		})

		httpRequestCheckOk[string](http.MethodPost, "/api/v1/tasks", createReq)
		code, _ := httpRequest[[]*download.Task](http.MethodGet, "/api/v1/tasks", nil)
		checkCode(code, 400)

		wg.Wait()
		r := httpRequestCheckOk[[]*download.Task](http.MethodGet, "/api/v1/tasks?status=done", nil)
		if r[0].Status != base.DownloadStatusDone {
			t.Errorf("GetTasks() got = %v, want %v", r[0].Status, base.DownloadStatusDone)
		}
	})
}

func doTest(handler func()) {
	fileListener := doStart()
	defer func() {
		fileListener.Close()
		Stop()
	}()
	taskReq.URL = "http://" + fileListener.Addr().String() + "/" + test.BuildName
	handler()
}

func doStart() net.Listener {
	port, err := Start("127.0.0.1", 0)
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
