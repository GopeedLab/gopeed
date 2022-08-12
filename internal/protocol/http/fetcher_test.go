package http

import (
	"github.com/monkeyWie/gopeed-core/internal/controller"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/test"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestFetcher_Resolve(t *testing.T) {
	testResolve(test.StartTestFileServer, &base.Resource{
		Length: test.BuildSize,
		Range:  true,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Size: test.BuildSize,
			},
		},
	}, t)
	testResolve(test.StartTestChunkedServer, &base.Resource{
		Length: 0,
		Range:  false,
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
		t.Errorf("Resolve error = %v, want %v", res, want)
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
	//downloadContinue(listener, 1, t)
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
		t.Errorf("Download error = %v, want %v", got, want)
	}
}

func downloadContinue(listener net.Listener, connections int, t *testing.T) {
	fetcher, _, _ := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 200)
	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 200)
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
		t.Errorf("Download error = %v, want %v", got, want)
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
		t.Errorf("Download error = %v, want %v", err, nil)
	}
}

func downloadResume(listener net.Listener, connections int, t *testing.T) {
	fetcher, res, opts := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}

	fb := new(FetcherBuilder)
	time.Sleep(time.Millisecond * 200)
	data := fb.Store(fetcher)
	time.Sleep(time.Millisecond * 200)
	fetcher.Pause()

	fetcher = fb.Resume(res, opts, data)
	fetcher.Setup(controller.NewController())
	fetcher.Continue()

	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
	}
}

func buildFetcher() *Fetcher {
	fetcher := new(FetcherBuilder).Build()
	fetcher.Setup(controller.NewController())
	return fetcher.(*Fetcher)
}
