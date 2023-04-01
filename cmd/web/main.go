//go:build web
// +build web

package main

import (
	"embed"
	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"io/fs"
)

//go:embed dist/*
var dist embed.FS

func main() {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	cfg := &model.StartConfig{
		Network:   "tcp",
		Address:   "0.0.0.0:9999",
		Storage:   model.StorageBolt,
		WebEnable: true,
		WebFS:     sub,
	}
	cmd.Start(cfg)
}
