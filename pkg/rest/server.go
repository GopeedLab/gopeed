package rest

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	srv         *http.Server
	runningPort int
	aesKey      []byte

	Downloader *download.Downloader
)

func Start(startCfg *model.StartConfig) (port int, err error) {
	// avoid repeat start
	if srv != nil {
		return runningPort, nil
	}

	var listener net.Listener
	srv, listener, err = BuildServer(startCfg)
	if err != nil {
		return
	}

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	if addr, ok := listener.Addr().(*net.TCPAddr); ok {
		port = addr.Port
		runningPort = port
	}
	return
}

func Stop() {
	defer func() {
		srv = nil
	}()

	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
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
		ProductionMode:    startCfg.ProductionMode,
		RefreshInterval:   startCfg.RefreshInterval,
		WhiteDownloadDirs: startCfg.WhiteDownloadDirs,
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

	if startCfg.WebEnable {
		aesKey = make([]byte, 32)
		if _, err := rand.Read(aesKey); err != nil {
			return nil, nil, errors.Wrap(err, "generate aes key failed")
		}
	}

	listener, err := net.Listen(startCfg.Network, startCfg.Address)
	if err != nil {
		return nil, nil, err
	}

	var r = mux.NewRouter()
	r.Methods(http.MethodGet).Path("/api/v1/info").HandlerFunc(Info)
	r.Methods(http.MethodPost).Path("/api/v1/resolve").HandlerFunc(Resolve)
	r.Methods(http.MethodPost).Path("/api/v1/tasks").HandlerFunc(CreateTask)
	r.Methods(http.MethodPost).Path("/api/v1/tasks/batch").HandlerFunc(CreateTaskBatch)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/{id}/pause").HandlerFunc(PauseTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/pause").HandlerFunc(PauseTasks)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/{id}/continue").HandlerFunc(ContinueTask)
	r.Methods(http.MethodPut).Path("/api/v1/tasks/continue").HandlerFunc(ContinueTasks)
	r.Methods(http.MethodDelete).Path("/api/v1/tasks/{id}").HandlerFunc(DeleteTask)
	r.Methods(http.MethodDelete).Path("/api/v1/tasks").HandlerFunc(DeleteTasks)
	r.Methods(http.MethodGet).Path("/api/v1/tasks/{id}").HandlerFunc(GetTask)
	r.Methods(http.MethodGet).Path("/api/v1/tasks").HandlerFunc(GetTasks)
	r.Methods(http.MethodGet).Path("/api/v1/tasks/{id}/stats").HandlerFunc(GetStats)
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
	r.Methods(http.MethodPost).Path("/api/v1/webhook/test").HandlerFunc(TestWebhook)
	r.Path("/api/v1/proxy").HandlerFunc(DoProxy)

	enableApiToken := startCfg.ApiToken != ""
	enableBasicAuth := startCfg.WebEnable && startCfg.WebAuth != nil
	if startCfg.WebEnable {
		if enableBasicAuth {
			r.Methods(http.MethodPost).Path("/api/web/login").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var loginReq model.WebAuth
				if ReadJson(r, w, &loginReq) {
					if loginReq.Username == startCfg.WebAuth.Username && loginReq.Password == startCfg.WebAuth.Password {
						// Generate a login token, Username:Password:Timestamp
						timestamp := time.Now().Unix()
						tokenData := fmt.Sprintf("%s:%s:%d", loginReq.Username, loginReq.Password, timestamp)
						token, err := aesEncrypt(aesKey, []byte(tokenData))
						if err != nil {
							WriteJson(w, model.NewErrorResult(err.Error()))
							return
						}

						WriteJson(w, model.NewOkResult(token))
						return
					}
				}
				WriteStatusJson(w, http.StatusUnauthorized, model.NewErrorResult("unauthorized", model.CodeUnauthorized))
			})
		}
		r.PathPrefix("/fs/tasks").Handler(http.FileServer(new(taskFileSystem)))
		r.PathPrefix("/fs/extensions").Handler(http.FileServer(new(extensionFileSystem)))
		r.PathPrefix("/").Handler(gzipMiddleware(http.FileServer(newEmbedCacheFileSystem(http.FS(startCfg.WebFS)))))
	}
	if enableApiToken || enableBasicAuth {
		writeUnauthorized := func(w http.ResponseWriter, r *http.Request) {
			WriteStatusJson(w, http.StatusUnauthorized, model.NewErrorResult("unauthorized", model.CodeUnauthorized))
		}

		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if enableApiToken {
					apiTokenHeader := r.Header["X-Api-Token"]
					// If api token header is set, only check api token ignore basic auth
					if len(apiTokenHeader) > 0 {
						if apiTokenHeader[0] == startCfg.ApiToken {
							h.ServeHTTP(w, r)
							return
						}

						writeUnauthorized(w, r)
						return
					}
				}

				if enableBasicAuth {
					if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/web/login" {
						h.ServeHTTP(w, r)
						return
					}

					token := r.Header.Get("Authorization")
					if token == "" {
						writeUnauthorized(w, r)
						return
					}

					token = strings.TrimPrefix(token, "Bearer ")
					tokenData, err := aesDecrypt(aesKey, token)
					if err != nil {
						writeUnauthorized(w, r)
						return
					}
					parts := strings.SplitN(string(tokenData), ":", 3)
					username := parts[0]
					password := parts[1]
					timestamp, _ := strconv.Atoi(parts[2])

					if username != startCfg.WebAuth.Username || password != startCfg.WebAuth.Password {
						writeUnauthorized(w, r)
						return
					}

					// Check if the token is expired (7 days)
					if time.Now().Unix()-int64(timestamp) > 7*24*3600 {
						writeUnauthorized(w, r)
						return
					}

					h.ServeHTTP(w, r)
					return
				}
				writeUnauthorized(w, r)
			})
		})
	}

	// recover panic
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					err := errors.WithStack(fmt.Errorf("%v", v))
					Downloader.Logger.Error().Stack().Err(err).Msgf("http server panic: %s %s", r.Method, r.RequestURI)
					WriteJson(w, model.NewErrorResult(err.Error(), model.CodeError))
				}
			}()
			h.ServeHTTP(w, r)
		})
	})

	srv = &http.Server{Handler: handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-Api-Token", "X-Target-Uri"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)(r)}
	return srv, listener, nil
}

func resolvePath(urlPath string, prefix string) (identity string, path string, err error) {
	// remove prefix
	clearPath := strings.TrimPrefix(urlPath, prefix)
	// match extension identity, eg: /fs/extensions/identity/xxx
	reg := regexp.MustCompile(`^/([^/]+)/(.*)$`)
	if !reg.MatchString(clearPath) {
		err = os.ErrNotExist
		return
	}
	matched := reg.FindStringSubmatch(clearPath)
	if len(matched) != 3 {
		err = os.ErrNotExist
		return
	}
	return matched[1], matched[2], nil
}

// handle task file resource
type taskFileSystem struct {
}

func (e *taskFileSystem) Open(name string) (http.File, error) {
	// get extension identity
	identity, path, err := resolvePath(name, "/fs/tasks")
	if err != nil {
		return nil, err
	}
	task := Downloader.GetTask(identity)
	if task == nil {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(task.Meta.RootDirPath(), path))
}

// handle extension file resource
type extensionFileSystem struct {
}

func (e *extensionFileSystem) Open(name string) (http.File, error) {
	// get extension identity
	identity, path, err := resolvePath(name, "/fs/extensions")
	if err != nil {
		return nil, err
	}
	extension, err := Downloader.GetExtension(identity)
	if err != nil {
		return nil, os.ErrNotExist
	}
	extensionPath := Downloader.ExtensionPath(extension)
	return os.Open(filepath.Join(extensionPath, path))
}

type embedCacheFileSystem struct {
	fs          http.FileSystem
	lastModTime time.Time
}

func newEmbedCacheFileSystem(fs http.FileSystem) *embedCacheFileSystem {
	efs := &embedCacheFileSystem{
		fs:          fs,
		lastModTime: time.Now(),
	}

	exe, err := os.Executable()
	if err != nil {
		return efs
	}

	fi, err := os.Stat(exe)
	if err != nil {
		return efs
	}

	efs.lastModTime = fi.ModTime()
	return efs
}

func (e *embedCacheFileSystem) Open(name string) (http.File, error) {
	file, err := e.fs.Open(name)
	if err != nil {
		return nil, err
	}

	return &embedFile{
		File:        file,
		lastModTime: e.lastModTime,
	}, nil
}

type embedFile struct {
	http.File
	lastModTime time.Time
}

type embedFileInfo struct {
	fs.FileInfo
	lastModTime time.Time
}

func (e *embedFileInfo) ModTime() time.Time {
	return e.lastModTime
}

func (e *embedFile) Stat() (fs.FileInfo, error) {
	fi, err := e.File.Stat()
	if err != nil {
		return nil, err
	}
	return &embedFileInfo{
		FileInfo:    fi,
		lastModTime: e.lastModTime,
	}, nil
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

func aesEncrypt(key, data []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func aesDecrypt(key []byte, encryptedData string) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(cipherText) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, cipherText := cipherText[:gcm.NonceSize()], cipherText[gcm.NonceSize():]
	return gcm.Open(nil, nonce, cipherText, nil)
}
