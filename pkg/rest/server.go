package rest

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/monkeyWie/gopeed-core/internal/protocol/bt"
	fhttp "github.com/monkeyWie/gopeed-core/internal/protocol/http"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"net"
	"net/http"
)

var (
	srv *http.Server

	Downloader = download.NewDownloader(new(fhttp.FetcherBuilder), new(bt.FetcherBuilder))
)

func Start(addr string, port int) (int, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return 0, err
	}

	var r = mux.NewRouter()
	r.Methods(http.MethodPost).Path("/api/v1/resolve").HandlerFunc(Resolve)
	r.Methods(http.MethodPost).Path("/api/v1/tasks").HandlerFunc(CreateTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/{id}/pause").HandlerFunc(PauseTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/{id}/continue").HandlerFunc(ContinueTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks/{id}").HandlerFunc(GetTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks").HandlerFunc(GetTasks)

	srv = &http.Server{Handler: r}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			// TODO log
			panic(err)
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func Stop() error {
	if srv != nil {
		return srv.Shutdown(context.TODO())
	}
	return nil
}
