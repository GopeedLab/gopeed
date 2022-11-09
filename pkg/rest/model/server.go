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
