package model

import (
	"github.com/GopeedLab/gopeed/pkg/base"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"io/fs"
)

type Storage string

const (
	StorageMem  Storage = "mem"
	StorageBolt Storage = "bolt"
)

type StartConfig struct {
	Network           string                      `json:"network"`
	Address           string                      `json:"address"`
	RefreshInterval   int                         `json:"refreshInterval"`
	Storage           Storage                     `json:"storage"`
	StorageDir        string                      `json:"storageDir"`
	WhiteDownloadDirs []string                    `json:"whiteDownloadDirs"`
	ApiToken          string                      `json:"apiToken"`
	DownloadConfig    *base.DownloaderStoreConfig `json:"downloadConfig"`
	WebViewRPCConfig  *enginewebview.RPCConfig    `json:"webViewRpcConfig,omitempty"`

	ProductionMode  bool
	WebViewProvider enginewebview.Provider `json:"-"`

	WebEnable bool
	WebFS     fs.FS
	WebAuth   *WebAuth
}

func (cfg *StartConfig) Init() *StartConfig {
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:0"
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 350
	}
	if cfg.Storage == "" {
		cfg.Storage = StorageBolt
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./"
	}
	return cfg
}

type WebAuth struct {
	Username string
	Password string
}
