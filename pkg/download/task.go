package download

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/util"
	"sync"
	"time"
)

type Task struct {
	ID        string         `json:"id"`
	Res       *base.Resource `json:"res"`
	Opts      *base.Options  `json:"opts"`
	Status    base.Status    `json:"status"`
	Progress  *Progress      `json:"progress"`
	Size      int64          `json:"size"`
	CreatedAt time.Time      `json:"createdAt"`

	fetcherBuilder fetcher.FetcherBuilder
	fetcher        fetcher.Fetcher
	timer          *util.Timer
	lock           *sync.Mutex
}

func NewTask() *Task {
	id, err := gonanoid.New()
	if err != nil {
		panic(err)
	}
	return &Task{
		ID:        id,
		Status:    base.DownloadStatusReady,
		CreatedAt: time.Now(),
	}
}

func (t *Task) clone() *Task {
	return &Task{
		ID:        t.ID,
		Res:       t.Res,
		Opts:      t.Opts,
		Status:    t.Status,
		Progress:  t.Progress,
		Size:      t.Size,
		CreatedAt: t.CreatedAt,
	}
}
