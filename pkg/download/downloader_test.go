package download

import (
	"github.com/monkeyWie/gopeed-core/internal/protocol/http"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/test"
	"reflect"
	"sync"
	"testing"
)

func TestDownloader_Resolve(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(http.FetcherBuilder)
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	res, err := downloader.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	want := &base.Resource{
		Req:       req,
		TotalSize: test.BuildSize,
		Range:     true,
		Files: []*base.FileInfo{
			{
				Name: test.BuildName,
				Path: "",
				Size: test.BuildSize,
			},
		},
	}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("Resolve error = %s, want %s", test.ToJson(res), test.ToJson(want))
	}
}

func TestDownloader_Create(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(http.FetcherBuilder)
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	res, err := downloader.Resolve(req)
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
	err = downloader.Create(res, &base.Options{
		Path:        test.Dir,
		Name:        test.DownloadName,
		Connections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
	}
}
