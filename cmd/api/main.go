package main

import (
	_ "embed"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/rest"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"net/http"
)

//go:embed banner.txt
var banner string

func main() {
	cfg := &model.StartConfig{
		Network: "tcp",
		Address: "127.0.0.1:9999",
		Storage: model.StorageBolt,
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
