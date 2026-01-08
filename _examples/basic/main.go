package main

import (
	"fmt"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func main() {
	finallyCh := make(chan error)
	_, err := download.Boot().
		URL("https://www.baidu.com/index.html").
		Listener(func(event *download.Event) {
			if event.Key == download.EventKeyFinally {
				finallyCh <- event.Err
			}
		}).
		Create(&base.Options{
			Extra: http.OptsExtra{
				Connections: 8,
			},
		})
	if err != nil {
		panic(err)
	}
	err = <-finallyCh
	if err != nil {
		fmt.Printf("download fail:%v\n", err)
	} else {
		fmt.Println("download success")
	}
}
