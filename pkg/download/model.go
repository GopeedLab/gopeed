package download

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/internal/protocol/bt"
	"github.com/monkeyWie/gopeed/internal/protocol/http"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/util"
	"sync"
	"time"
)

type ResolveResult struct {
	ID  string         `json:"id"`
	Res *base.Resource `json:"res"`
}

type Task struct {
	ID        string               `json:"id"`
	Meta      *fetcher.FetcherMeta `json:"meta"`
	Status    base.Status          `json:"status"`
	Progress  *Progress            `json:"progress"`
	Size      int64                `json:"size"`
	CreatedAt time.Time            `json:"createdAt"`

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
		Meta:      t.Meta,
		Status:    t.Status,
		Progress:  t.Progress,
		Size:      t.Size,
		CreatedAt: t.CreatedAt,
	}
}

type DownloaderConfig struct {
	Controller    *controller.Controller
	FetchBuilders []fetcher.FetcherBuilder
	Storage       Storage

	*DownloaderStoreConfig
}

func (cfg *DownloaderConfig) Init() *DownloaderConfig {
	if cfg.Controller == nil {
		cfg.Controller = controller.NewController()
	}
	if len(cfg.FetchBuilders) == 0 {
		cfg.FetchBuilders = []fetcher.FetcherBuilder{
			new(http.FetcherBuilder),
			new(bt.FetcherBuilder),
		}
	}
	if cfg.Storage == nil {
		cfg.Storage = NewMemStorage()
	}

	if cfg.DownloaderStoreConfig == nil {
		cfg.DownloaderStoreConfig = &DownloaderStoreConfig{}
	}

	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 350
	}
	return cfg
}

// DownloaderStoreConfig is the config that can restore the downloader.
type DownloaderStoreConfig struct {
	RefreshInterval int            `json:"refreshInterval"` // RefreshInterval time duration to refresh task progress(ms)
	DownloadDir     string         `json:"downloadDir"`     // DownloadDir is the directory to save the downloaded files
	ProtocolConfig  map[string]any `json:"protocolConfig"`  // ProtocolConfig is special config for each protocol
	Extra           map[string]any `json:"extra"`           // Extra is the extra config
}
