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
	ApiToken        string  `json:"apiToken"`

	WebEnable bool
	WebFS     fs.FS
}

func (cfg *StartConfig) Init() *StartConfig {
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:0"
	}
	if cfg.Storage == "" {
		cfg.Storage = StorageBolt
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./"
	}
	return cfg
}
