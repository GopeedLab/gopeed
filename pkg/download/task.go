package download

import (
	"github.com/google/uuid"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"sync"
)

type Task struct {
	ID       string
	Res      *base.Resource
	Opts     *base.Options
	Status   base.Status
	Files    map[string]*base.FileInfo
	Progress *Progress

	fetcher fetcher.Fetcher
	timer   *util.Timer
	locker  *sync.Mutex
}

func NewTask(fetcher fetcher.Fetcher) *Task {
	return &Task{
		ID:      uuid.New().String(),
		Status:  base.DownloadStatusPrepare,
		fetcher: fetcher,
	}
}
