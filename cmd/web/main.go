//go:build web
// +build web

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

//go:embed dist/*
var dist embed.FS

func main() {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}

	args := parse()
	var webAuth *model.WebAuth
	if isNotBlank(args.Username) && isNotBlank(args.Password) {
		webAuth = &model.WebAuth{
			Username: *args.Username,
			Password: *args.Password,
		}
	}

	var storageDir string
	if args.StorageDir != nil && *args.StorageDir != "" {
		storageDir = *args.StorageDir
	} else {
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}
		storageDir = filepath.Join(filepath.Dir(exe), "storage")
	}

	cfg := &model.StartConfig{
		Network:           "tcp",
		Address:           fmt.Sprintf("%s:%d", *args.Address, *args.Port),
		Storage:           model.StorageBolt,
		StorageDir:        storageDir,
		WhiteDownloadDirs: args.WhiteDownloadDirs,
		ApiToken:          *args.ApiToken,
		DownloadConfig:    args.DownloadConfig,
		ProductionMode:    true,
		WebEnable:         true,
		WebFS:             sub,
		WebAuth:           webAuth,
	}
	cmd.Start(cfg)
}

func isNotBlank(str *string) bool {
	return str != nil && *str != ""
}
