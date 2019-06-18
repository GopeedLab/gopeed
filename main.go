package main

import (
	"fmt"

	httpdownload "gopeed/down/http"
)

func main() {
	Init()

	request := &httpdownload.Request{
		Method: "GET",
		URL:    config.HTTPDownloadAddress,
		Header: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
			"Referer":         "http://github.com/proxyee-down-org/proxyee-down/releases",
			"Accept-Encoding": "gzip, deflate, br",
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		},
	}
	got, err := httpdownload.Resolve(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	httpdownload.Down(request, config.ParrallelsNumber)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(got)
	// webview.Open("Minimal webview example", "https://www.baidu.com", 800, 600, true)
}
