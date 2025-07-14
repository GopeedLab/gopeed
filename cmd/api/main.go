package main

import (
	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

// only for local development
func main() {
	cfg := &model.StartConfig{
		Network:   "tcp",
		Address:   "127.0.0.1:9999",
		Storage:   model.StorageBolt,
		WebEnable: true,
	}
	cmd.Start(cfg)
}
