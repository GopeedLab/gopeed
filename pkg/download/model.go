package download

import (
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/internal/protocol/bt"
	"github.com/monkeyWie/gopeed/internal/protocol/http"
)

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
		cfg.RefreshInterval = 1000
	}
	return cfg
}

// DownloaderStoreConfig is the config that can restore the downloader.
type DownloaderStoreConfig struct {
	RefreshInterval int            `json:"refreshInterval"` // RefreshInterval time duration to refresh task progress(ms)
	DownloadDir     string         `json:"downloadDir"`     // DownloadDir is the directory to save the downloaded files
	ProtocolExtra   map[string]any `json:"protocolExtra"`   // ProtocolExtra is special config for each protocol
	Extra           map[string]any `json:"extra"`           // Extra is the extra config
}
