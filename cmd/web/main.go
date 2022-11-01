package main

import (
	"embed"
	"fmt"
	"github.com/monkeyWie/gopeed/pkg/rest"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
	"io/fs"
	"net/http"
)

//go:embed banner.txt
var banner string

//go:embed dist/*
var dist embed.FS

func main() {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	cfg := &model.StartConfig{
		Network:   "tcp",
		Address:   "127.0.0.1:9999",
		Storage:   model.StorageMem,
		WebEnable: true,
		WebFS:     sub,
	}
	fmt.Println(banner)
	srv, listener, err := rest.BuildServer(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server start success on http://%s\n", listener.Addr().String())
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
