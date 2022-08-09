package download

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/internal/protocol/bt"
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

	downloader := NewDownloader(new(http.FetcherBuilder))
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	_, res, err := downloader.Resolve(req)
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

	downloader := NewDownloader(new(http.FetcherBuilder))
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}
	taskID, res, err := downloader.Resolve(req)
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
	err = downloader.Create(taskID, res, &base.Options{
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

func TestDownloader_CreateBT(t *testing.T) {
	downloader := NewDownloader(new(bt.FetcherBuilder))
	req := &base.Request{
		URL: "magnet:?xt=urn:btih:A213E40628B27B58E70F0DADAC5350CA05403AC4&dn=Microsoft%20Office%20PRO%20Plus%202021%20Retail%20Version%202108%20Build%2014326.2&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.openbittorrent.com%3A6969%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2710%2Fannounce&tr=udp%3A%2F%2F9.rarbg.me%3A2780%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2730%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=http%3A%2F%2Fp4p.arenabg.com%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce&tr=udp%3A%2F%2Ftracker.tiny-vps.com%3A6969%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce",
	}
	taskID, res, err := downloader.Resolve(req)
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
	err = downloader.Create(taskID, res, nil)
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()

	fmt.Println("done")
}
