package main

import (
	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// only for local development
func main() {
	cfg := &model.StartConfig{
		Storage: model.StorageBolt,
		DownloadConfig: &base.DownloaderStoreConfig{
			Http: &base.DownloaderHttpConfig{
				Enable: true,
				Host:   "127.0.0.1",
				Port:   9999,
			},
		},
		WebEnable: true,
	}
	cmd.Start(cfg)
}
