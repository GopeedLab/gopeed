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
		t.Errorf("Download() got = %v, want %v", got, want)
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
		downloader.DeleteByStatues(nil, true)
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
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		return nil
	}, nil)
	// Disable proxy
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.Enable = false
		return proxyCfg
	}, nil)
	// Enable system proxy but not set proxy environment variable
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.System = true
		return proxyCfg
	}, nil)
	// Enable proxy but error proxy environment variable
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
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
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		os.Setenv("HTTP_PROXY", proxyCfg.ToUrl().String())
		os.Setenv("HTTPS_PROXY", proxyCfg.ToUrl().String())
		proxyCfg.System = true
		return proxyCfg
	}, nil)
	// Invalid proxy scheme
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.Scheme = ""
		return proxyCfg
	}, nil)
	// Invalid proxy host
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		proxyCfg.Host = ""
		return proxyCfg
	}, nil)
	// Use proxy without auth
	doTestDownloaderCreateWithProxy(t, false, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		return proxyCfg
	}, nil)
	// Use proxy with auth
	doTestDownloaderCreateWithProxy(t, true, func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig {
		return proxyCfg
	}, nil)
}

func doTestDownloaderCreateWithProxy(t *testing.T, auth bool, buildProxyConfig func(proxyCfg *base.DownloaderProxyConfig) *base.DownloaderProxyConfig, errHandler func(err error)) {
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
	downloader.cfg.DownloaderStoreConfig.Proxy = buildProxyConfig(&base.DownloaderProxyConfig{
		Enable: true,
		Scheme: "socks5",
		Host:   proxyListener.Addr().String(),
		Usr:    usr,
		Pwd:    pwd,
	})

	req := &base.Request{
		URL: test.ExternalDownloadUrl,
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
	err = downloader.Pause(id)
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
	err = downloader.Continue(id)
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
