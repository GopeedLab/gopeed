package main

import (
	"github.com/monkeyWie/gopeed-core/download"
	"github.com/monkeyWie/gopeed-core/download/common"
	"github.com/monkeyWie/gopeed-core/download/http"
)

func main() {
	downloader := buildDownloader()
	res, err := downloader.Resolve(&common.Request{
		URL: "http://192.168.200.163:8088/docker-compose",
	})
	if err != nil {
		panic(err)
	}
	err = downloader.Create(res, &common.Options{
		Path:        "E:\\test\\gopeed\\http",
		Connections: 4,
	})
	if err != nil {
		panic(err)
	}
}

func buildDownloader() *download.Downloader {
	return download.NewDownloader(&http.Fetcher{BaseFetcher: &common.BaseFetcher{}})
}
