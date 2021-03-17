package main

import (
	"github.com/monkeyWie/gopeed-core/download"
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/http"
)

func main() {
	downloader := download.NewDownloader(http.FetcherBuilder)
	res, err := downloader.Resolve(&base.Request{
		URL: "http://192.168.200.163:8088/docker-compose",
	})
	if err != nil {
		panic(err)
	}
	err = downloader.Create(res, &base.Options{
		Path:        "E:\\test\\gopeed\\http",
		Connections: 4,
	})
	if err != nil {
		panic(err)
	}
}
