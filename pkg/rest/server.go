package rest

import (
	"context"
	"encoding/json"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
			Downloader.Logger.Warn().Err(err).Msg("shutdown server failed")
		}
	}
	if Downloader != nil {
		if err := Downloader.Close(); err != nil {
			Downloader.Logger.Warn().Err(err).Msg("close downloader failed")
		}
	}
}

func BuildServer(startCfg *model.StartConfig) (*http.Server, net.Listener, error) {
	if startCfg == nil {
		startCfg = &model.StartConfig{}
	}
	startCfg.Init()

	downloadCfg := &download.DownloaderConfig{
		ProductionMode:  startCfg.ProductionMode,
		RefreshInterval: startCfg.RefreshInterval,
	}
	if startCfg.Storage == model.StorageBolt {
		downloadCfg.Storage = download.NewBoltStorage(startCfg.StorageDir)
	} else {
		downloadCfg.Storage = download.NewMemStorage()
	}
	downloadCfg.StorageDir = startCfg.StorageDir
	downloadCfg.Init()
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
	r.Methods(http.MethodPut).Path("/api/v1/tasks/pause").HandlerFunc(PauseAllTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/continue").HandlerFunc(ContinueAllTask)
	r.Methods(http.MethodDelete).Path("/api/v1/tasks/{id}").HandlerFunc(DeleteTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks/{id}").HandlerFunc(GetTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks").HandlerFunc(GetTasks)
	r.Methods(http.MethodGet).Path("/api/v1/config").HandlerFunc(GetConfig)
	r.Methods(http.MethodPut).Path("/api/v1/config").HandlerFunc(PutConfig)
	r.Methods(http.MethodPost).Path("/api/v1/extensions").HandlerFunc(InstallExtension)
	r.Methods(http.MethodGet).Path("/api/v1/extensions").HandlerFunc(GetExtensions)
	r.Methods(http.MethodGet).Path("/api/v1/extensions/{identity}").HandlerFunc(GetExtension)
	r.Methods(http.MethodPut).Path("/api/v1/extensions/{identity}/settings").HandlerFunc(UpdateExtensionSettings)
	r.Methods(http.MethodPut).Path("/api/v1/extensions/{identity}/switch").HandlerFunc(SwitchExtension)
	r.Methods(http.MethodDelete).Path("/api/v1/extensions/{identity}").HandlerFunc(DeleteExtension)
	r.Methods(http.MethodGet).Path("/api/v1/extensions/{identity}/update").HandlerFunc(UpdateCheckExtension)
	r.Methods(http.MethodPost).Path("/api/v1/extensions/{identity}/update").HandlerFunc(UpdateExtension)
	r.PathPrefix("/fs/extensions").Handler(http.FileServer(new(extensionFileSystem)))
	r.Path("/api/v1/proxy").HandlerFunc(DoProxy)
	if startCfg.WebEnable {
		r.PathPrefix("/").Handler(http.FileServer(http.FS(startCfg.WebFS)))
	}

	if startCfg.ApiToken != "" || (startCfg.WebEnable && startCfg.WebBasicAuth != nil) {
		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if startCfg.ApiToken != "" && r.Header.Get("X-Api-Token") == startCfg.ApiToken {
					h.ServeHTTP(w, r)
					return
				}
				if startCfg.WebEnable && startCfg.WebBasicAuth != nil {
					if r.Header.Get("Authorization") == startCfg.WebBasicAuth.Authorization() {
						h.ServeHTTP(w, r)
						return
					}
					w.Header().Set("WWW-Authenticate", "Basic realm=\"gopeed web\"")
				}
				WriteStatusJson(w, http.StatusUnauthorized, model.NewErrorResult("unauthorized", model.CodeUnauthorized))
			})
		})
	}

	srv = &http.Server{Handler: handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type", "X-Api-Token", "X-Target-Uri"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
	)(r)}
	return srv, listener, nil
}

// handle extension file resource
type extensionFileSystem struct {
}

func (e *extensionFileSystem) Open(name string) (http.File, error) {
	// remove prefix
	path := strings.TrimPrefix(name, "/fs/extensions")
	// match extension identity, eg: /fs/extensions/identity/xxx
	reg := regexp.MustCompile(`^/([^/]+)/(.*)$`)
	if !reg.MatchString(path) {
		return nil, os.ErrNotExist
	}
	matched := reg.FindStringSubmatch(path)
	if len(matched) != 3 {
		return nil, os.ErrNotExist
	}
	// get extension identity
	identity := matched[1]
	extension, err := Downloader.GetExtension(identity)
	if err != nil {
		return nil, os.ErrNotExist
	}
	extensionPath := Downloader.ExtensionPath(extension)
	return os.Open(filepath.Join(extensionPath, matched[2]))
}

func ReadJson(r *http.Request, w http.ResponseWriter, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return false
	}
	return true
}

func WriteJson(w http.ResponseWriter, v any) {
	WriteStatusJson(w, http.StatusOK, v)
}

func WriteStatusJson(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}
