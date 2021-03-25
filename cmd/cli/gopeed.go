package main

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/download"
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/http"
	"github.com/monkeyWie/gopeed-core/util"
	"path/filepath"
	"sync"
)

const progressWidth = 20

func main() {
	args := parse()

	downloader := download.NewDownloader(http.FetcherBuilder)
	res, err := downloader.Resolve(&base.Request{
		URL: args.url,
	})
	if err != nil {
		panic(err)
	}

	err = downloader.Create(res, &base.Options{
		Path:        *args.dir,
		Connections: *args.connections,
	})
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	downloader.Listener(func(taskInfo *download.TaskInfo, eventKey base.EventKey) {
		if eventKey == base.EventKeyProgress {
			printProgress(taskInfo, "downloading...")
		}
		if eventKey == base.EventKeyDone {
			printProgress(taskInfo, "complete")
			wg.Done()
		}
	})
	wg.Wait()
	fmt.Println()
	gPrint("saving file " + filepath.Join(*args.dir, res.Files[0].Name))
}

func printProgress(taskInfo *download.TaskInfo, title string) {
	rate := float64(taskInfo.Progress.Downloaded) / float64(taskInfo.Res.TotalSize)
	completeWidth := int(progressWidth * rate)
	speed := util.ByteFmt(taskInfo.Progress.Speed)
	totalSize := util.ByteFmt(taskInfo.Res.TotalSize)
	fmt.Printf("\r%s [", title)
	for i := 0; i < progressWidth; i++ {
		if i < completeWidth {
			fmt.Print("■")
		} else {
			fmt.Print("□")
		}
	}
	fmt.Printf("] %.1f%%    %s/s    %s", rate*100, speed, totalSize)
}
