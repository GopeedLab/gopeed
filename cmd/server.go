package cmd

import (
	_ "embed"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"github.com/GopeedLab/gopeed/pkg/base"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

//go:embed banner.txt
var banner string

func Start(cfg *model.StartConfig) {
	fmt.Println(banner)
	instance, err := api.Create(cfg)
	if err != nil {
		panic(err)
	}

	downloadCfgResult := instance.GetConfig()
	if downloadCfgResult.HasError() {
		panic(downloadCfgResult.Msg)
	}
	downloadCfg := downloadCfgResult.Data
	if downloadCfg.FirstLoad {
		// Set default download config
		if cfg.DownloadConfig != nil {
			cfg.DownloadConfig.Merge(downloadCfg)
			// TODO Use PatchConfig
			instance.PutConfig(cfg.DownloadConfig)
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
				instance.PutConfig(downloadCfg)
			}
		}
	}

	watchExit(instance)

	port, start, err := api.ListenHttp(cfg.DownloadConfig.Http, instance)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server start success on http://%s:%d\n", cfg.DownloadConfig.Http.Host, port)
	start()
}

func watchExit(instance *api.Instance) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Server is shutting down due to signal: %s\n", sig)
		instance.Close()
		os.Exit(0)
	}()
}
