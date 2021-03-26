package main

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"path/filepath"
	"sync"
)

const progressWidth = 20

func main() {
	args := parse()

	var wg sync.WaitGroup
	err := download.Boot().
		URL(args.url).
		Listener(func(event *download.Event) {
			if event.Key == download.EventKeyProgress {
				printProgress(event.Task, "downloading...")
			}
			if event.Key == download.EventKeyFinally {
				var title string
				if event.Err != nil {
					title = "fail"
				} else {
					title = "complete"
				}
				printProgress(event.Task, title)
				fmt.Println()
				if event.Err != nil {
					gPrint("reason: " + event.Err.Error())
				} else {
					gPrint("saving file " + filepath.Join(*args.dir, event.Task.Res.Files[0].Name))
				}
				wg.Done()
			}
		}).
		Create(&base.Options{
			Path:        *args.dir,
			Connections: *args.connections,
		})
	if err != nil {
		panic(err)
	}
	wg.Add(1)
	wg.Wait()
}

func printProgress(task *download.TaskInfo, title string) {
	rate := float64(task.Progress.Downloaded) / float64(task.Res.TotalSize)
	completeWidth := int(progressWidth * rate)
	speed := util.ByteFmt(task.Progress.Speed)
	totalSize := util.ByteFmt(task.Res.TotalSize)
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
