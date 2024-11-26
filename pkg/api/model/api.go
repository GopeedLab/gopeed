package model

import (
	"encoding/base64"
	"github.com/GopeedLab/gopeed/pkg/base"
	"io/fs"
)

type Storage string

const (
	StorageMem  Storage = "mem"
	StorageBolt Storage = "bolt"
)

type StartConfig struct {
	RefreshInterval   int                         `json:"refreshInterval"`
	Storage           Storage                     `json:"storage"`
	StorageDir        string                      `json:"storageDir"`
	WhiteDownloadDirs []string                    `json:"whiteDownloadDirs"`
	DownloadConfig    *base.DownloaderStoreConfig `json:"downloadConfig"`

	ProductionMode bool

	WebEnable    bool
	WebFS        fs.FS
	WebBasicAuth *WebBasicAuth
}

func (cfg *StartConfig) Init() *StartConfig {
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

type WebBasicAuth struct {
	Username string
	Password string
}

// Authorization returns the value of the Authorization header to be used in HTTP requests.
func (cfg *WebBasicAuth) Authorization() string {
	userId := cfg.Username + ":" + cfg.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(userId))
}

type StartResult struct {
	Host string `json:"host"`
	// Port is the http server real listen port.
	Port int `json:"port"`
}
