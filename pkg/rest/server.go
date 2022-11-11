package rest

import (
	"context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/monkeyWie/gopeed/pkg/download"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
	"github.com/monkeyWie/gopeed/pkg/util"
	"net"
	"net/http"
)

var (
	srv *http.Server

	Downloader *download.Downloader
)

func Start(startCfg *model.StartConfig) (int, error) {
	srv, listener, err := BuildServer(startCfg)
	if err != nil {
		return 0, err
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			// TODO log
			panic(err)
		}
	}()

	port := 0
	if addr, ok := listener.Addr().(*net.TCPAddr); ok {
		port = addr.Port
	}
	return port, nil
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

func BuildServer(startCfg *model.StartConfig) (*http.Server, net.Listener, error) {
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
	downloadCfg.Init()
	downloadCfg.RefreshInterval = startCfg.RefreshInterval
	Downloader = download.NewDownloader(downloadCfg)
	if err := Downloader.Setup(); err != nil {
		return nil, nil, err
	}

	if startCfg.Network == "unix" {
		util.SafeRemove(startCfg.Address)
	}

	listener, err := net.Listen(startCfg.Network, startCfg.Address)
	if err != nil {
		return nil, nil, err
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
	r.Methods(http.MethodPost).Path("/api/v1/action").HandlerFunc(DoAction)
	if startCfg.WebEnable {
		r.PathPrefix("/").Handler(http.FileServer(http.FS(startCfg.WebFS)))
	}

	srv = &http.Server{Handler: handlers.CORS(
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
	)(r)}
	return srv, listener, nil
}
