package cmd

import (
	_ "embed"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/download"
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
	exist, _, err := rest.Downloader.GetConfig()
	if err != nil {
		panic(err)
	}
	if !exist {
		// Set default download dir to user download dir
		userDir, err := os.UserHomeDir()
		if err == nil {
			downloadDir := filepath.Join(userDir, "Downloads")
			rest.Downloader.PutConfig(&download.DownloaderStoreConfig{
				DownloadDir: downloadDir,
			})
		}

	}
	fmt.Printf("Server start success on http://%s\n", listener.Addr().String())
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
