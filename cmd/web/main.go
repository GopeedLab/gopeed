//go:build web
// +build web

package main

import (
	"embed"
	"fmt"
	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed dist/*
var dist embed.FS

func main() {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}

	args := parse()
	var webBasicAuth *model.WebBasicAuth
	if isNotBlank(args.Username) && isNotBlank(args.Password) {
		webBasicAuth = &model.WebBasicAuth{
			Username: *args.Username,
			Password: *args.Password,
		}
	}

	var dir string
	if args.StorageDir != nil && *args.StorageDir != "" {
		dir = *args.StorageDir
	} else {
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}
		dir = filepath.Dir(exe)
	}

	cfg := &model.StartConfig{
		Network:        "tcp",
		Address:        fmt.Sprintf("%s:%d", *args.Address, *args.Port),
		Storage:        model.StorageBolt,
		StorageDir:     filepath.Join(dir, "storage"),
		ApiToken:       *args.ApiToken,
		DownloadConfig: args.DownloadConfig,
		ProductionMode: true,
		WebEnable:      true,
		WebFS:          sub,
		WebBasicAuth:   webBasicAuth,
	}
	cmd.Start(cfg)
}

func isNotBlank(str *string) bool {
	return str != nil && *str != ""
}
