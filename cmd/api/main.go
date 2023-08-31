package main

import (
	_ "embed"
	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func main() {
	cfg := &model.StartConfig{
		Network: "tcp",
		Address: "127.0.0.1:9999",
		Storage: model.StorageBolt,
	}
	cmd.Start(cfg)
}
