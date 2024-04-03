package main

import (
	"fmt"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
	"strings"
	"sync"
)

const progressWidth = 20

func main() {
	args := parse()

	var wg sync.WaitGroup
	wg.Add(1)
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
			Path:  *args.dir,
			Extra: http.OptsExtra{Connections: *args.connections},
		})
	if err != nil {
		panic(err)
	}
	printProgress(emptyTask, "downloading...")
	wg.Wait()
}

var (
	lastLineLen = 0
	sb          = new(strings.Builder)
	emptyTask   = &download.Task{
		Progress: &download.Progress{},
		Meta: &fetcher.FetcherMeta{
			Res: &base.Resource{},
		},
	}
)

func printProgress(task *download.Task, title string) {
	var rate float64
	if task.Meta.Res == nil {
		task = emptyTask
	}
	if task.Meta.Res.Size <= 0 {
		rate = 0
	} else {
		rate = float64(task.Progress.Downloaded) / float64(task.Meta.Res.Size)
	}
	completeWidth := int(progressWidth * rate)
	speed := util.ByteFmt(task.Progress.Speed)
	totalSize := util.ByteFmt(task.Meta.Res.Size)
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
