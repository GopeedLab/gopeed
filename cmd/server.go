package cmd

import (
	_ "embed"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/rest"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
		// Set default download config
		if cfg.DownloadConfig != nil {
			cfg.DownloadConfig.Merge(downloadCfg)
			// TODO Use PatchConfig
			rest.Downloader.PutConfig(cfg.DownloadConfig)
			downloadCfg = cfg.DownloadConfig
		}

		downloadDir := downloadCfg.DownloadDir
		// Set default download dir, in docker, it will be ${exe}/Downloads, else it will be ${user}/Downloads
		if downloadDir == "" {
			if base.InDocker == "true" {
				downloadDir = filepath.Join(filepath.Dir(cfg.StorageDir), "Downloads")
			} else {
				userDir, err := os.UserHomeDir()
				if err == nil {
					downloadDir = filepath.Join(userDir, "Downloads")
				}
			}
			if downloadDir != "" {
				downloadCfg.DownloadDir = downloadDir
				rest.Downloader.PutConfig(downloadCfg)
			}
		}
	}
	watchExit()

	fmt.Printf("Server start success on http://%s\n", listener.Addr().String())
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func watchExit() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Server is shutting down due to signal: %s\n", sig)
		rest.Downloader.Close()
		os.Exit(0)
	}()
}
