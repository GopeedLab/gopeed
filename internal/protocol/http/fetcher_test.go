package http

import (
	"fmt"
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/internal/test"
	"github.com/monkeyWie/gopeed/pkg/base"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestFetcher_Resolve(t *testing.T) {
	testResolve(test.StartTestFileServer, &base.Resource{
		Name:  test.BuildName,
		Size:  test.BuildSize,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Size: test.BuildSize,
			},
		},
	}, t)
	testResolve(test.StartTestChunkedServer, &base.Resource{
		Name:  test.BuildName,
		Size:  0,
		Range: false,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Size: 0,
			},
		},
	}, t)
}

func testResolve(startTestServer func() net.Listener, want *base.Resource, t *testing.T) {
	listener := startTestServer()
	defer listener.Close()
	fetcher := buildFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Req = nil
	if !reflect.DeepEqual(want, res) {
		t.Errorf("Resolve() got = %v, want %v", res, want)
	}
}

func TestFetcher_DownloadNormal(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	// 正常下载
	downloadNormal(listener, 1, t)
	downloadNormal(listener, 5, t)
	downloadNormal(listener, 8, t)
	downloadNormal(listener, 16, t)
}

func TestFetcher_DownloadContinue(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	// 暂停继续
	downloadContinue(listener, 1, t)
	downloadContinue(listener, 5, t)
	downloadContinue(listener, 8, t)
	downloadContinue(listener, 16, t)
}

func TestFetcher_DownloadChunked(t *testing.T) {
	listener := test.StartTestChunkedServer()
	defer listener.Close()
	// chunked编码下载
	downloadNormal(listener, 1, t)
}

func TestFetcher_DownloadPost(t *testing.T) {
	listener := test.StartTestPostServer()
	defer listener.Close()
	// post下载
	downloadPost(listener, 1, t)
}

func TestFetcher_DownloadRetry(t *testing.T) {
	listener := test.StartTestRetryServer()
	defer listener.Close()
	// chunked编码下载
	downloadNormal(listener, 1, t)
}

func TestFetcher_DownloadError(t *testing.T) {
	listener := test.StartTestErrorServer()
	defer listener.Close()
	// chunked编码下载
	downloadError(listener, 1, t)
}

func TestFetcher_DownloadResume(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadResume(listener, 1, t)
	downloadResume(listener, 5, t)
	downloadResume(listener, 8, t)
	downloadResume(listener, 16, t)
}

func downloadReady(listener net.Listener, connections int, t *testing.T) (fetcher fetcher.Fetcher, res *base.Resource, opts *base.Options) {
	fetcher = buildFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	})
	if err != nil {
		t.Fatal(err)
	}
	opts = &base.Options{
		Name:        test.DownloadName,
		Path:        test.Dir,
		Connections: connections,
	}
	err = fetcher.Create(res, opts)
	if err != nil {
		t.Fatal(err)
	}
	return
}

func downloadNormal(listener net.Listener, connections int, t *testing.T) {
	fetcher, _, _ := downloadReady(listener, connections, t)
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
	fetcher, _, _ := downloadReady(listener, connections, t)
	fetcher.(*Fetcher).res.Req.Extra = extra{
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
	fetcher, _, _ := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	if err := fetcher.Continue(); err != nil {
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
	fetcher, _, _ := downloadReady(listener, connections, t)
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
	fetcher, res, opts := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}

	fb := new(FetcherBuilder)
	time.Sleep(time.Millisecond * 50)
	data, err := fb.Store(fetcher)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	fetcher.Pause()

	_, f := fb.Restore()
	f(res, opts, data)
	if err != nil {
		t.Fatal(err)
	}
	fetcher.Setup(controller.NewController())
	fetcher.Continue()

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
	fetcher := new(FetcherBuilder).Build()
	fetcher.Setup(controller.NewController())
	return fetcher.(*Fetcher)
}
