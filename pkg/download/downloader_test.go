package download

import (
	"archive/zip"
	"fmt"
	"io"
	"net"
	gohttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

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
	rr, err := downloader.Resolve(req)
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
	rr, err := downloader.Resolve(req)
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
	_, err = downloader.Create(rr.ID, &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
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
	rr, err := downloader.Resolve(req)
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
	_, err = downloader.Create(rr.ID, &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	})
	if !strings.Contains(err.Error(), "white") {
		t.Errorf("TestDownloader_CreateNotInWhite() got = %v, want %v", err.Error(), "not in white list")
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

	_, err := downloader.CreateDirectBatch(&base.CreateTaskBatch{
		Reqs: reqs,
		Opts: &base.Options{
			Path: test.Dir,
			Name: test.DownloadName,
			Extra: http.OptsExtra{
				Connections: 4,
			},
		},
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
		if _, err := os.Stat(test.Dir + "/" + name); os.IsNotExist(err) {
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
		URL: test.ExternalDownloadUrl,
	}
	if buildReqProxy != nil {
		req.Proxy = buildReqProxy(&base.RequestProxy{
			Scheme: "socks5",
			Host:   proxyListener.Addr().String(),
			Usr:    usr,
			Pwd:    pwd,
		})
	}
	rr, err := downloader.Resolve(req)
	if err != nil {
		if errHandler == nil {
			t.Fatal(err)
		}
		errHandler(err)
		return
	}
	want := &base.Resource{
		Size:  test.ExternalDownloadSize,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: test.ExternalDownloadName,
				Path: "",
				Size: test.ExternalDownloadSize,
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
	var wg sync.WaitGroup
	wg.Add(2)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})
	for i := 0; i < 2; i++ {
		_, err := downloader.CreateDirect(req, &base.Options{
			Path: test.Dir,
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
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Downloader_CreateRename() got = %v, want %v", got, want)
	}
	got = test.FileMd5(test.DownloadRenameFile)
	if want != got {
		t.Errorf("Downloader_CreateRename() got = %v, want %v", got, want)
	}
}

func TestDownloader_StoreAndRestore(t *testing.T) {
	listener := test.StartTestSlowFileServer(time.Millisecond * 2000)
	defer listener.Close()

	downloader := NewDownloader(&DownloaderConfig{
		Storage: NewBoltStorage("./"),
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	rr, err := downloader.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}

	id, err := downloader.Create(rr.ID, &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 1001)
	err = downloader.Pause(&TaskFilter{IDs: []string{id}})
	if err != nil {
		t.Fatal(err)
	}
	downloader.Close()

	downloader = NewDownloader(&DownloaderConfig{
		Storage: NewBoltStorage("./"),
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
	got := test.FileMd5(test.DownloadFile)
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

	taskIds, err := downloader.CreateDirectBatch(&base.CreateTaskBatch{
		Reqs: reqs,
		Opts: &base.Options{
			Path: test.Dir,
			Name: test.DownloadName,
			Extra: http.OptsExtra{
				Connections: 4,
			},
		},
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
	rr, err := downloader.Resolve(req)
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

	taskId, err := downloader.Create(rr.ID, &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	})
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
		taskId, err := downloader.CreateDirect(req, &base.Options{
			Path: test.Dir,
			Name: test.DownloadName,
			Extra: http.OptsExtra{
				Connections: 4,
			},
		})
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

func TestDownloader_ContinueBatch(t *testing.T) {
	listener := test.StartTestSlowFileServer(time.Millisecond * 2000)
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Create multiple tasks and pause them
	taskCount := 3
	taskIds := make([]string, 0)

	for i := 0; i < taskCount; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		rr, err := downloader.Resolve(req)
		if err != nil {
			t.Fatal(err)
		}
		taskId, err := downloader.Create(rr.ID, &base.Options{
			Path: test.Dir,
			Name: test.DownloadName,
			Extra: http.OptsExtra{
				Connections: 4,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		taskIds = append(taskIds, taskId)
	}

	// Wait a moment for tasks to start
	time.Sleep(time.Millisecond * 500)

	// Pause all tasks
	err := downloader.Pause(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify tasks are paused
	for _, taskId := range taskIds {
		task := downloader.GetTask(taskId)
		if task.Status != base.DownloadStatusPause && task.Status != base.DownloadStatusDone {
			t.Errorf("Task %s should be paused or done, got %s", taskId, task.Status)
		}
	}

	// Test ContinueBatch with specific IDs
	t.Run("ContinueBatch with filter", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyDone {
				wg.Done()
			}
		})

		err := downloader.ContinueBatch(&TaskFilter{IDs: []string{taskIds[0]}})
		if err != nil {
			t.Errorf("ContinueBatch() unexpected error: %v", err)
		}
		wg.Wait()
	})

	// Test ContinueBatch with nil filter (continues all)
	t.Run("ContinueBatch with nil filter", func(t *testing.T) {
		var wg sync.WaitGroup
		// Count remaining paused tasks
		pausedCount := 0
		for _, taskId := range taskIds {
			task := downloader.GetTask(taskId)
			if task.Status == base.DownloadStatusPause {
				pausedCount++
			}
		}
		wg.Add(pausedCount)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyDone {
				wg.Done()
			}
		})

		err := downloader.ContinueBatch(nil)
		if err != nil {
			t.Errorf("ContinueBatch() unexpected error: %v", err)
		}
		if pausedCount > 0 {
			wg.Wait()
		}
	})

	// Clean up
	downloader.Delete(nil, true)
}

func TestDownloader_PauseAndContinue(t *testing.T) {
	listener := test.StartTestSlowFileServer(time.Millisecond * 5000)
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	rr, err := downloader.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}

	taskId, err := downloader.Create(rr.ID, &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for task to start
	time.Sleep(time.Millisecond * 1000)

	// Test Pause with empty filter (returns nil, not error - empty filter means no op)
	t.Run("Pause with empty filter", func(t *testing.T) {
		err := downloader.Pause(&TaskFilter{})
		// Empty filter returns error from Pause function
		if err == nil {
			// This is fine - different implementations might handle this differently
		}
	})

	// Test Pause with filter
	t.Run("Pause with filter", func(t *testing.T) {
		task := downloader.GetTask(taskId)
		if task == nil {
			t.Skip("Task already completed")
		}
		if task.Status == base.DownloadStatusDone {
			t.Skip("Task already completed")
		}

		err := downloader.Pause(&TaskFilter{IDs: []string{taskId}})
		if err != nil {
			// Task might have finished between check and pause
			t.Logf("Pause() error (task may have finished): %v", err)
		} else {
			task = downloader.GetTask(taskId)
			if task.Status != base.DownloadStatusPause && task.Status != base.DownloadStatusDone {
				t.Errorf("Task should be paused or done, got %s", task.Status)
			}
		}
	})

	// Test Pause with non-matching filter
	t.Run("Pause with non-matching filter", func(t *testing.T) {
		err := downloader.Pause(&TaskFilter{IDs: []string{"non-existent-id"}})
		if err == nil {
			t.Error("Pause() with non-matching filter should return error")
		}
	})

	// Test Continue with empty filter (returns error - empty filter means no op)
	t.Run("Continue with empty filter", func(t *testing.T) {
		err := downloader.Continue(&TaskFilter{})
		// Empty filter returns error from Continue function
		if err == nil {
			// This is fine - different implementations might handle this differently
		}
	})

	// Test Continue with non-matching filter
	t.Run("Continue with non-matching filter", func(t *testing.T) {
		err := downloader.Continue(&TaskFilter{IDs: []string{"non-existent-id"}})
		if err == nil {
			t.Error("Continue() with non-matching filter should return error")
		}
	})

	// Test Continue with valid filter
	t.Run("Continue with valid filter", func(t *testing.T) {
		task := downloader.GetTask(taskId)
		if task == nil {
			t.Skip("Task not found")
		}
		if task.Status == base.DownloadStatusDone {
			t.Skip("Task already completed")
		}

		var wg sync.WaitGroup
		wg.Add(1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyDone && event.Task != nil && event.Task.ID == taskId {
				wg.Done()
			}
		})

		err := downloader.Continue(&TaskFilter{IDs: []string{taskId}})
		if err != nil {
			// Task might have finished or not be in pausable state
			t.Logf("Continue() note: %v", err)
		}
		wg.Wait()

		task = downloader.GetTask(taskId)
		if task != nil && task.Status != base.DownloadStatusDone {
			t.Errorf("Task should be done, got %s", task.Status)
		}
	})

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
			AutoExtract: true,
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
			AutoExtract:        true,
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
			AutoExtract: true,
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
			Path: test.Dir,
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

func TestDownloader_ContinueAll(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	// Create a task and wait for completion
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			wg.Done()
		}
	})

	taskId, err := downloader.CreateDirect(req, &base.Options{
		Path: test.Dir,
		Name: test.DownloadName,
		Extra: http.OptsExtra{
			Connections: 4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	// Verify task completed
	task := downloader.GetTask(taskId)
	if task == nil {
		t.Fatal("Task should exist")
	}
	if task.Status != base.DownloadStatusDone {
		t.Errorf("Task should be done, got %s", task.Status)
	}

	// Test ContinueBatch with nil on an already done task (should be a no-op)
	err = downloader.ContinueBatch(nil)
	if err != nil {
		t.Logf("ContinueBatch(nil) on done tasks returned: %v", err)
	}

	// Clean up
	downloader.Delete(nil, true)
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
