package download

import (
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/protocol/bt"
	"github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"sync"
	"time"
)

type ResolveResult struct {
	ID  string         `json:"id"`
	Res *base.Resource `json:"res"`
}

type Task struct {
	ID        string               `json:"id"`
	Protocol  string               `json:"protocol"`
	Meta      *fetcher.FetcherMeta `json:"meta"`
	Status    base.Status          `json:"status"`
	Uploading bool                 `json:"uploading"`
	Progress  *Progress            `json:"progress"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`

	fetcherManager fetcher.FetcherManager
	fetcher        fetcher.Fetcher
	timer          *util.Timer
	statusLock     *sync.Mutex
	lock           *sync.Mutex
	speedArr       []int64
	uploadSpeedArr []int64
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
		UpdatedAt: time.Now(),
	}
}

func (t *Task) updateStatus(status base.Status) {
	t.UpdatedAt = time.Now()
	t.Status = status
}

func (t *Task) clone() *Task {
	return &Task{
		ID:        t.ID,
		Protocol:  t.Protocol,
		Meta:      t.Meta,
		Status:    t.Status,
		Uploading: t.Uploading,
		Progress:  t.Progress,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func (t *Task) calcSpeed(speedArr []int64, downloaded int64, usedTime float64) int64 {
	speedArr = append(speedArr, downloaded)
	if len(speedArr) > 6 {
		speedArr = speedArr[1:]
	}
	var total int64
	for _, v := range speedArr {
		total += v
	}
	return int64(float64(total) / float64(len(speedArr)) / usedTime)
}

type DownloaderConfig struct {
	Controller    *controller.Controller
	FetchManagers []fetcher.FetcherManager

	RefreshInterval int `json:"refreshInterval"` // RefreshInterval time duration to refresh task progress(ms)
	Storage         Storage
	StorageDir      string

	ProductionMode bool

	*base.DownloaderStoreConfig
}

func (cfg *DownloaderConfig) Init() *DownloaderConfig {
	if cfg.Controller == nil {
		cfg.Controller = controller.NewController()
	}
	if len(cfg.FetchManagers) == 0 {
		cfg.FetchManagers = []fetcher.FetcherManager{
			new(http.FetcherManager),
			new(bt.FetcherManager),
		}
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 350
	}
	if cfg.Storage == nil {
		cfg.Storage = NewMemStorage()
	}
	return cfg
}
