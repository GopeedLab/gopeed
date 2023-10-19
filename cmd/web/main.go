//go:build web
// +build web

package main

import (
	"embed"
	"fmt"
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

	args := parse()
	var webBasicAuth *model.WebBasicAuth
	if isNotBlank(args.Username) && isNotBlank(args.Password) {
		webBasicAuth = &model.WebBasicAuth{
			Username: *args.Username,
			Password: *args.Password,
		}
	}

	cfg := &model.StartConfig{
		Network:        "tcp",
		Address:        fmt.Sprintf("%s:%d", *args.Address, *args.Port),
		Storage:        model.StorageBolt,
		ApiToken:       *args.ApiToken,
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
