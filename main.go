package main

import (
	"fmt"
	"github.com/monkeyWie/gopeed/protocol/http"
)

func main() {
	request := &http.Request{
		Method: "get",
		URL:    "http://github.com/proxyee-down-org/proxyee-down/releases/download/3.4/proxyee-down-main.jar",
		Header: map[string]string{
			"Host":            "github.com",
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
			"Referer":         "http://github.com/proxyee-down-org/proxyee-down/releases",
			"Accept-Encoding": "gzip, deflate, br",
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		},
	}
	got, _ := http.Resolve(request)
	fmt.Println(got)
}
