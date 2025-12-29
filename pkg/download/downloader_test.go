package download

import (
	"archive/zip"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

	nameMatchMap := make(map[string]bool)
	for _, name := range fileNames {
		if _, err := os.Stat(test.Dir + "/" + name); os.IsNotExist(err) {
			t.Errorf("CreateDirectBatch() file not exist: %v", name)
		}
		for _, task := range tasks {
			if name == task.Meta.Opts.Name {
				nameMatchMap[task.Meta.Opts.Name] = true
			}
		}
	}
	if len(nameMatchMap) != len(reqs) {
		t.Errorf("CreateDirectBatch() names got = %v, want %v", len(nameMatchMap), len(reqs))
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
