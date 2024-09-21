package download

import (
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
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
		DownloadDirWhiteList: []string{"./downloads"},
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

	reqs := make([]*base.Request, 0)
	fileNames := make([]string, 0)
	for i := 0; i < 5; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		reqs = append(reqs, req)
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

	_, err := downloader.CreateDirectBatch(reqs, &base.Options{
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

	reqs := make([]*base.Request, 0)
	fileNames := make([]string, 0)
	for i := 0; i < 10; i++ {
		req := &base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}
		reqs = append(reqs, req)
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

	taskIds, err := downloader.CreateDirectBatch(reqs, &base.Options{
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
