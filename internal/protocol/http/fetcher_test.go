package http

import (
	"encoding/json"
	"fmt"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
	"net"
	"testing"
	"time"
)

func TestFetcher_Resolve(t *testing.T) {
	testResolve(test.StartTestFileServer, test.BuildName, &base.Resource{
		Size:  test.BuildSize,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Size: test.BuildSize,
			},
		},
	}, t)
	testResolve(test.StartTestCustomServer, "disposition", &base.Resource{
		Size:  test.BuildSize,
		Range: false,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Size: test.BuildSize,
			},
		},
	}, t)
	testResolve(test.StartTestCustomServer, test.BuildName, &base.Resource{
		Size:  0,
		Range: false,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Size: 0,
			},
		},
	}, t)

	fetcher := buildFetcher()
	err := fetcher.Resolve(&base.Request{
		URL: "http://github.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fetcher.Meta().Res.Files[0].Name != "github.com" {
		t.Errorf("Resolve() got = %v, want %v", fetcher.Meta().Res, "github.com")
	}
}

func testResolve(startTestServer func() net.Listener, path string, want *base.Resource, t *testing.T) {
	listener := startTestServer()
	defer listener.Close()
	fetcher := buildFetcher()
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !test.AssertResourceEqual(want, fetcher.meta.Res) {
		t.Errorf("Resolve() got = %v, want %v", fetcher.meta.Res, want)
	}
}

func TestFetcher_DownloadNormal(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)
	downloadNormal(listener, 5, t)
	downloadNormal(listener, 8, t)
	downloadNormal(listener, 16, t)
}

func TestFetcher_DownloadContinue(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadContinue(listener, 1, t)
	downloadContinue(listener, 5, t)
	downloadContinue(listener, 8, t)
	downloadContinue(listener, 16, t)
}

func TestFetcher_DownloadChunked(t *testing.T) {
	listener := test.StartTestCustomServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)
	downloadNormal(listener, 2, t)
}

func TestFetcher_DownloadPost(t *testing.T) {
	listener := test.StartTestPostServer()
	defer listener.Close()

	downloadPost(listener, 1, t)
}

func TestFetcher_DownloadRetry(t *testing.T) {
	listener := test.StartTestRetryServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)
}

func TestFetcher_DownloadError(t *testing.T) {
	listener := test.StartTestErrorServer()
	defer listener.Close()

	downloadError(listener, 1, t)
}

func TestFetcher_DownloadLimit(t *testing.T) {
	listener := test.StartTestLimitServer(4, 0)
	defer listener.Close()

	downloadNormal(listener, 1, t)
	downloadNormal(listener, 2, t)
	downloadNormal(listener, 8, t)
}

func TestFetcher_DownloadResume(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadResume(listener, 1, t)
	downloadResume(listener, 5, t)
	downloadResume(listener, 8, t)
	downloadResume(listener, 16, t)
}

func TestFetcher_DownloadWithProxy(t *testing.T) {
	httpListener := test.StartTestFileServer()
	defer httpListener.Close()
	proxyListener := test.StartSocks5Server("", "")
	defer proxyListener.Close()

	downloadWithProxy(httpListener, proxyListener, t)
}

func TestFetcher_ConfigConnections(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	fetcher := doDownloadReady(buildConfigFetcher(config{
		Connections: 16,
	}), listener, 0, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func TestFetcher_ConfigUseServerCtime(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	fetcher := doDownloadReady(buildConfigFetcher(config{
		Connections:    16,
		UseServerCtime: true,
	}), listener, 0, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadReady(listener net.Listener, connections int, t *testing.T) fetcher.Fetcher {
	return doDownloadReady(buildFetcher(), listener, connections, t)
}

func doDownloadReady(f fetcher.Fetcher, listener net.Listener, connections int, t *testing.T) fetcher.Fetcher {
	err := f.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	})
	if err != nil {
		t.Fatal(err)
	}
	var extra any = nil
	if connections > 0 {
		extra = http.OptsExtra{
			Connections: connections,
		}
	}
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: extra,
	}
	err = f.Create(opts)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func downloadNormal(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadPost(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	fetcher.Meta().Req.Extra = &http.ReqExtra{
		Method: "POST",
		Header: map[string]string{
			"Authorization": "Bearer 123456",
		},
		Body: fmt.Sprintf(`{"name":"%s"}`, test.BuildName),
	}
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadContinue(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadError(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err == nil {
		t.Errorf("Download() got = %v, want %v", err, nil)
	}
}

func downloadResume(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}

	fb := new(FetcherManager)
	time.Sleep(time.Millisecond * 50)
	data, err := fb.Store(fetcher)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	fetcher.Pause()

	_, f := fb.Restore()
	f(fetcher.Meta(), data)
	if err != nil {
		t.Fatal(err)
	}
	fetcher.Setup(controller.NewController())
	fetcher.Start()

	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadWithProxy(httpListener net.Listener, proxyListener net.Listener, t *testing.T) {
	fetcher := downloadReady(httpListener, 4, t)
	ctl := controller.NewController()
	ctl.ProxyConfig = &base.DownloaderProxyConfig{
		Enable: true,
		Scheme: "socks5",
		Host:   proxyListener.Addr().String(),
	}
	fetcher.Setup(ctl)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func buildFetcher() *Fetcher {
	fb := new(FetcherManager)
	fetcher := fb.Build()
	newController := controller.NewController()
	newController.GetConfig = func(v any) {
		json.Unmarshal([]byte(test.ToJson(fb.DefaultConfig())), v)
	}
	fetcher.Setup(newController)
	return fetcher.(*Fetcher)
}

func buildConfigFetcher(cfg config) fetcher.Fetcher {
	fetcher := new(FetcherManager).Build()
	newController := controller.NewController()
	newController.GetConfig = func(v any) {
		json.Unmarshal([]byte(test.ToJson(cfg)), v)
	}
	fetcher.Setup(newController)
	return fetcher
}
