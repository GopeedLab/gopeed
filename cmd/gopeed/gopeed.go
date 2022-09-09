package main

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"strings"
	"sync"
)

const progressWidth = 20

func main() {
	args := parse()

	var wg sync.WaitGroup
	_, err := download.Boot().
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
					fmt.Printf("reason: %s", event.Err.Error())
				} else {
					fmt.Printf("saving path: %s", *args.dir)
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

var (
	lastLineLen = 0
	sb          = new(strings.Builder)
)

func printProgress(task *download.Task, title string) {
	rate := float64(task.Progress.Downloaded) / float64(task.Res.Size)
	completeWidth := int(progressWidth * rate)
	speed := util.ByteFmt(task.Progress.Speed)
	totalSize := util.ByteFmt(task.Res.Size)
	sb.WriteString(fmt.Sprintf("\r%s [", title))
	for i := 0; i < progressWidth; i++ {
		if i < completeWidth {
			sb.WriteString("■")
		} else {
			sb.WriteString("□")
		}
	}
	sb.WriteString(fmt.Sprintf("] %.1f%%    %s/s    %s", rate*100, speed, totalSize))
	if lastLineLen != 0 {
		paddingLen := lastLineLen - sb.Len()
		if paddingLen > 0 {
			sb.WriteString(strings.Repeat(" ", paddingLen))
		}
	}
	lastLineLen = sb.Len()
	fmt.Print(sb.String())
	sb.Reset()
}
