package download

import (
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/protocol/bt"
	"github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"net/url"
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
	Protocol  string               `json:"protocol"`
	Progress  *Progress            `json:"progress"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`

	fetcherBuilder fetcher.FetcherBuilder
	fetcher        fetcher.Fetcher
	timer          *util.Timer
	statusLock     *sync.Mutex
	lock           *sync.Mutex
	speedArr       []int64
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
		Meta:      t.Meta,
		Status:    t.Status,
		Progress:  t.Progress,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func (t *Task) calcSpeed(downloaded int64, usedTime float64) int64 {
	t.speedArr = append(t.speedArr, downloaded)
	if len(t.speedArr) > 6 {
		t.speedArr = t.speedArr[1:]
	}
	var total int64
	for _, v := range t.speedArr {
		total += v
	}
	return int64(float64(total) / float64(len(t.speedArr)) / usedTime)
}

type DownloaderConfig struct {
	Controller    *controller.Controller
	FetchBuilders []fetcher.FetcherBuilder

	RefreshInterval int `json:"refreshInterval"` // RefreshInterval time duration to refresh task progress(ms)
	Storage         Storage
	StorageDir      string

	ProductionMode bool

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
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 350
	}
	if cfg.Storage == nil {
		cfg.Storage = NewMemStorage()
	}
	return cfg
}

// DownloaderStoreConfig is the config that can restore the downloader.
type DownloaderStoreConfig struct {
	FirstLoad bool `json:"-"` // FirstLoad is the flag that the config is first time init and not from store

	DownloadDir    string                 `json:"downloadDir"`    // DownloadDir is the default directory to save the downloaded files
	MaxRunning     int                    `json:"maxRunning"`     // MaxRunning is the max running download count
	ProtocolConfig map[string]any         `json:"protocolConfig"` // ProtocolConfig is special config for each protocol
	Extra          map[string]any         `json:"extra"`
	Proxy          *DownloaderProxyConfig `json:"proxy"`
}

func (cfg *DownloaderStoreConfig) Init() *DownloaderStoreConfig {
	if cfg.MaxRunning == 0 {
		cfg.MaxRunning = 5
	}
	if cfg.Proxy == nil {
		cfg.Proxy = &DownloaderProxyConfig{}
	}
	return cfg
}

func (cfg *DownloaderStoreConfig) ProxyUrl() *url.URL {
	if cfg.Proxy == nil {
		return nil
	}
	if cfg.Proxy.Enable == false || cfg.Proxy.Scheme == "" || cfg.Proxy.Host == "" {
		return nil
	}
	return util.BuildProxyUrl(cfg.Proxy.Scheme, cfg.Proxy.Host, cfg.Proxy.Usr, cfg.Proxy.Pwd)
}

type DownloaderProxyConfig struct {
	Enable bool   `json:"enable"`
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Usr    string `json:"usr"`
	Pwd    string `json:"pwd"`
}
