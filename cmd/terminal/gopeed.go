package main

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/download"
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/http"
	"github.com/superhawk610/bar"
	"github.com/ttacon/chalk"
	"strconv"
	"sync"
)

func main() {
	b := bar.NewWithOpts(
		bar.WithDimensions(10000, 20),
		bar.WithDisplay("[", "■", "■", "□", "]"),
		bar.WithFormat(
			fmt.Sprintf(
				" %s:status%s :percent :bar%s :speedKB/s%s ",
				chalk.Green,
				chalk.Reset,
				chalk.Blue,
				chalk.Reset,
			),
		),
	)

	downloader := download.NewDownloader(http.FetcherBuilder)
	res, err := downloader.Resolve(&base.Request{
		URL: "https://dldir1.qq.com/weixin/Windows/WeChatSetup.exe",
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

	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(taskInfo *download.TaskInfo, eventKey base.EventKey) {
		if eventKey == base.EventKeyProgress {
			b.Update(int(10000*(float64(taskInfo.Progress.Downloaded)/float64(taskInfo.Res.TotalSize))), bar.Context{
				bar.Ctx("status", "downloading..."),
				bar.Ctx("speed", strconv.Itoa(int(taskInfo.Progress.Speed/1024))),
			})
		}
		if eventKey == base.EventKeyDone {
			b.Update(10000, bar.Context{
				bar.Ctx("status", "complete"),
				bar.Ctx("speed", strconv.Itoa(int(taskInfo.Progress.Speed/1024))),
			})
			wg.Done()
		}
	})
	wg.Wait()
	b.Done()
}
