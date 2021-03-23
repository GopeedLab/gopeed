package download

import (
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/dltest"
	"github.com/monkeyWie/gopeed-core/download/http"
	"reflect"
	"sync"
	"testing"
)

func TestDownloader_Resolve(t *testing.T) {
	listener := dltest.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(http.FetcherBuilder)
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + dltest.BuildName,
	}
	res, err := downloader.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	want := &base.Resource{
		Req:       req,
		TotalSize: dltest.BuildSize,
		Range:     true,
		Files: []*base.FileInfo{
			{
				Name: dltest.BuildName,
				Path: "",
				Size: dltest.BuildSize,
			},
		},
	}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("Resolve error = %s, want %s", dltest.ToJson(res), dltest.ToJson(want))
	}
}

func TestDownloader_Create(t *testing.T) {
	listener := dltest.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(http.FetcherBuilder)
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + dltest.BuildName,
	}
	res, err := downloader.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(taskInfo *TaskInfo, eventKey base.EventKey) {
		if eventKey == base.EventKeyDone {
			wg.Done()
		}
	})
	err = downloader.Create(res, &base.Options{
		Path:        dltest.TestDir,
		Name:        dltest.DownloadName,
		Connections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	want := dltest.FileMd5(dltest.BuildFile)
	got := dltest.FileMd5(dltest.DownloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
	}
}
