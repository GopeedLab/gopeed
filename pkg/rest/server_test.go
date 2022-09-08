package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"github.com/monkeyWie/gopeed-core/pkg/test"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
)

var taskReq = &base.Request{}

func TestResolve(t *testing.T) {
	doTestApi("/api/v1/resolve",
		func() any {
			return taskReq
		},
		func(resp *base.Resource) {
			var want = &base.Resource{
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
			if !reflect.DeepEqual(want, resp) {
				t.Errorf("Resolve() got = %v, want %v", resp, want)
			}
		})
}

func TestCreateTask(t *testing.T) {
	doTestApi("/api/v1/tasks",
		func() any {
			resource, err := Downloader.Resolve(taskReq)
			if err != nil {
				t.Fatal(err)
			}
			resource.Req = taskReq
			req := &model.CreateTaskReq{
				Resource: resource,
				Options: &base.Options{
					Path:        test.Dir,
					Name:        test.DownloadName,
					Connections: 4,
				},
			}
			return req
		},
		func(resp string) {
			if resp == "" {
				t.Fatal("create task failed")
			}
			var wg sync.WaitGroup
			wg.Add(1)
			Downloader.Listener(func(event *download.Event) {
				if event.Key == download.EventKeyFinally {
					wg.Done()
				}
			})
			wg.Wait()
			want := test.FileMd5(test.BuildFile)
			got := test.FileMd5(test.DownloadFile)
			if want != got {
				t.Errorf("Download() got = %v, want %v", got, want)
			}
		})
}

func doTestApi[T any](path string, buildReq func() any, handler func(resp T)) {
	port, listener := doStart()
	defer func() {
		listener.Close()
		Stop()
	}()
	taskReq.URL = "http://" + listener.Addr().String() + "/" + test.BuildName
	resp := httpPost[T](port, path, buildReq())
	handler(resp)
}

func doStart() (int, net.Listener) {
	port, err := Start("127.0.0.1", 0)
	if err != nil {
		panic(err)
	}
	return port, test.StartTestFileServer()
}

func httpPost[T any](port int, path string, req any) T {
	buf, _ := json.Marshal(req)
	result, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d%s", port, path), "application/json", bytes.NewBuffer(buf))
	if err != nil {
		panic(err)
	}
	if result.StatusCode != http.StatusOK {
		panic(result.Status)
	}
	var r model.Result[T]
	if err := json.NewDecoder(result.Body).Decode(&r); err != nil {
		panic(err)
	}
	if r.Code != 0 {
		panic(r.Msg)
	}
	return r.Data
}
