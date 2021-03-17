package http

import (
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/dltest"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestFetcher_Resolve(t *testing.T) {
	testResolve(dltest.StartTestFileServer, &base.Resource{
		TotalSize: dltest.BuildSize,
		Range:     true,
		Files: []*base.FileInfo{
			{
				Name: dltest.BuildName,
				Size: dltest.BuildSize,
			},
		},
	}, t)
	testResolve(dltest.StartTestChunkedServer, &base.Resource{
		TotalSize: 0,
		Range:     false,
		Files: []*base.FileInfo{
			{
				Name: dltest.BuildName,
				Size: 0,
			},
		},
	}, t)
}

func testResolve(startTestServer func() net.Listener, want *base.Resource, t *testing.T) {
	listener := startTestServer()
	defer listener.Close()
	fetcher := NewFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + dltest.BuildName,
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
	listener := dltest.StartTestFileServer()
	defer listener.Close()
	// 正常下载
	downloadNormal(listener, 1, t)
	downloadNormal(listener, 5, t)
	downloadNormal(listener, 8, t)
	downloadNormal(listener, 16, t)
}

func TestFetcher_DownloadContinue(t *testing.T) {
	listener := dltest.StartTestFileServer()
	defer listener.Close()
	// 暂停继续
	downloadContinue(listener, 1, t)
	downloadContinue(listener, 5, t)
	downloadContinue(listener, 8, t)
	downloadContinue(listener, 16, t)
}

func TestFetcher_DownloadChunked(t *testing.T) {
	listener := dltest.StartTestChunkedServer()
	defer listener.Close()
	// chunked编码下载
	downloadNormal(listener, 1, t)
	downloadContinue(listener, 1, t)
}

func TestFetcher_DownloadRetry(t *testing.T) {
	listener := dltest.StartTestRetryServer()
	defer listener.Close()
	// chunked编码下载
	downloadNormal(listener, 1, t)
}

func TestFetcher_DownloadError(t *testing.T) {
	listener := dltest.StartTestErrorServer()
	defer listener.Close()
	// chunked编码下载
	downloadError(listener, 1, t)
}

func downloadReady(listener net.Listener, connections int, t *testing.T) base.Fetcher {
	fetcher := NewFetcher()
	fetcher.Setup(base.NewController())
	res, err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + dltest.BuildName,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Create(res, &base.Options{
		Name:        dltest.DownloadName,
		Path:        dltest.TestDir,
		Connections: connections,
	})
	if err != nil {
		t.Fatal(err)
	}
	return fetcher

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
	want := dltest.FileMd5(dltest.BuildFile)
	got := dltest.FileMd5(dltest.DownloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
	}
}

func downloadContinue(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
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
	want := dltest.FileMd5(dltest.BuildFile)
	got := dltest.FileMd5(dltest.DownloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
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
		t.Errorf("Download error = %v, want %v", err, nil)
	}
}
