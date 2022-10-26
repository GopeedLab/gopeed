package rest

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"github.com/monkeyWie/gopeed-core/pkg/util"
	"net"
	"net/http"
	"strconv"
)

var (
	srv *http.Server

	Downloader *download.Downloader
)

func Start(startCfg *model.StartConfig) (int, error) {
	if startCfg == nil {
		startCfg = &model.StartConfig{}
	}
	startCfg.Init()

	downloadCfg := &download.DownloaderConfig{}
	if startCfg.Storage == model.StorageBolt {
		downloadCfg.Storage = download.NewBoltStorage(startCfg.StorageDir)
	} else {
		downloadCfg.Storage = download.NewMemStorage()
	}
	downloadCfg.RefreshInterval = startCfg.RefreshInterval
	Downloader = download.NewDownloader(downloadCfg.Init())
	if err := Downloader.Setup(); err != nil {
		return 0, err
	}

	serverConfig := getAndPutServerConfig(startCfg)
	var address string
	if startCfg.Network == "unix" {
		address = startCfg.Address
		util.SafeRemove(address)
	} else {
		address = net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port))
	}

	listener, err := net.Listen(startCfg.Network, address)
	if err != nil {
		return 0, err
	}

	var r = mux.NewRouter()
	r.Methods(http.MethodPost).Path("/api/v1/resolve").HandlerFunc(Resolve)
	r.Methods(http.MethodPost).Path("/api/v1/tasks").HandlerFunc(CreateTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/{id}/pause").HandlerFunc(PauseTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/{id}/continue").HandlerFunc(ContinueTask)
	r.Methods(http.MethodDelete).Path("/api/v1/tasks/{id}").HandlerFunc(DeleteTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks/{id}").HandlerFunc(GetTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks").HandlerFunc(GetTasks)
	r.Methods(http.MethodGet).Path("/api/v1/config").HandlerFunc(GetConfig)
	r.Methods(http.MethodPut).Path("/api/v1/config").HandlerFunc(PutConfig)

	srv = &http.Server{Handler: r}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			// TODO log
			panic(err)
		}
	}()
	if addr, ok := listener.Addr().(*net.TCPAddr); ok {
		return addr.Port, nil
	}
	return 0, nil
}

func Stop() {
	if srv != nil {
		if err := srv.Shutdown(context.TODO()); err != nil {
			// TODO log
		}
	}
	if Downloader != nil {
		if err := Downloader.Close(); err != nil {
			// TODO log
		}
	}
}

func getAndPutServerConfig(startCfg *model.StartConfig) *model.ServerConfig {
	var serverConfig model.ServerConfig
	err := Downloader.GetConfig(&serverConfig)
	// first start
	if err != nil {
		if startCfg.Network == "tcp" {
			host, port, _ := net.SplitHostPort(startCfg.Address)
			serverConfig.Host = host
			serverConfig.Port, _ = strconv.Atoi(port)
		}
		serverConfig.Init()
		Downloader.PutConfig(serverConfig)
	}
	return &serverConfig
}
