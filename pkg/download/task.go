package download

import (
	"github.com/google/uuid"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"sync"
	"time"
)

type Task struct {
	ID         string                    `json:"id"`
	Res        *base.Resource            `json:"res"`
	Opts       *base.Options             `json:"opts"`
	Status     base.Status               `json:"status"`
	Files      map[string]*base.FileInfo `json:"files"`
	Progress   *Progress                 `json:"progress"`
	CreateTime time.Time                 `json:"create_time"`

	fetcher fetcher.Fetcher
	timer   *util.Timer
	locker  *sync.Mutex
}

func NewTask() *Task {
	return &Task{
		ID:         uuid.New().String(),
		Status:     base.DownloadStatusReady,
		CreateTime: time.Now(),
	}
}
