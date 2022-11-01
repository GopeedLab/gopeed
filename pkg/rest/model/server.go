package model

import "io/fs"

type Storage string

const (
	StorageMem  Storage = "mem"
	StorageBolt Storage = "bolt"
)

type StartConfig struct {
	Network         string  `json:"network"`
	Address         string  `json:"address"`
	Storage         Storage `json:"storage"`
	StorageDir      string  `json:"storageDir"`
	RefreshInterval int     `json:"refreshInterval"`

	WebEnable bool
	WebFS     fs.FS
}

func (cfg *StartConfig) Init() *StartConfig {
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Storage == "" {
		cfg.Storage = StorageBolt
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./"
	}
	return cfg
}

// ServerConfig is present in the database
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	Connections int    `json:"connections"`
	DownloadDir string `json:"downloadDir"`

	Extra map[string]any `json:"extra"`
}

func (cfg *ServerConfig) Init() *ServerConfig {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port < 0 {
		cfg.Port = 0
	}
	if cfg.Connections <= 0 {
		cfg.Connections = 16
	}
	return cfg
}
