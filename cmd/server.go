package cmd

import (
	_ "embed"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/rest"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed banner.txt
var banner string

func Start(cfg *model.StartConfig) {
	fmt.Println(banner)
	srv, listener, err := rest.BuildServer(cfg)
	if err != nil {
		panic(err)
	}
	downloadCfg, err := rest.Downloader.GetConfig()
	if err != nil {
		panic(err)
	}
	if downloadCfg.FirstLoad {
		// Set default download dir to user download dir
		userDir, err := os.UserHomeDir()
		if err == nil {
			downloadDir := filepath.Join(userDir, "Downloads")
			downloadCfg.DownloadDir = downloadDir
			rest.Downloader.PutConfig(downloadCfg)
		}

	}
	fmt.Printf("Server start success on http://%s\n", listener.Addr().String())
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
