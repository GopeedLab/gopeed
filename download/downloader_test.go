package download

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/http"
	"testing"
)

func TestDownloader_Resolve(t *testing.T) {
	downloader := buildDownloader()
	res, err := downloader.Resolve(&base.Request{
		URL: "http://192.168.200.163:8088/docker-compose",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestDownloader_Create(t *testing.T) {
	downloader := buildDownloader()
	res, err := downloader.Resolve(&base.Request{
		URL: "http://192.168.200.163:8088/docker-compose",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = downloader.Create(res, &base.Options{
		Path:        "E:\\test\\gopeed\\http",
		Connections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func buildDownloader() *Downloader {
	return NewDownloader(http.FetcherBuilder)
}
