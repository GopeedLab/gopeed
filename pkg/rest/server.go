package rest

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	goapi "github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

var (
	runtimeMu sync.Mutex

	Downloader *download.Downloader
	APIService *goapi.Service
)

const (
	webAuthCookieName = "gopeed-session"
	webAuthSessionTTL = 7 * 24 * time.Hour
)

type webSessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
	now      func() time.Time
}

func newWebSessionStore() *webSessionStore {
	return &webSessionStore{
		sessions: make(map[string]time.Time),
		now:      time.Now,
	}
}

func (s *webSessionStore) create() (string, time.Time, error) {
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", time.Time{}, errors.Wrap(err, "generate web session failed")
	}

	sessionID := base64.RawURLEncoding.EncodeToString(rawToken)
	expiresAt := s.now().Add(webAuthSessionTTL)

	s.mu.Lock()
	defer s.mu.Unlock()
	for existingToken, expiry := range s.sessions {
		if !expiry.After(s.now()) {
			delete(s.sessions, existingToken)
		}
	}
	s.sessions[sessionID] = expiresAt
	return sessionID, expiresAt, nil
}

func (s *webSessionStore) valid(sessionID string) bool {
	if sessionID == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.sessions[sessionID]
	if !ok {
		return false
	}
	if !expiresAt.After(s.now()) {
		delete(s.sessions, sessionID)
		return false
	}
	return true
}

func Start(startCfg *model.StartConfig) (port int, err error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	// avoid repeat start
	if APIService != nil {
		return apiServer.runningPort, nil
	}
	if startCfg == nil {
		startCfg = &model.StartConfig{}
	}
	startCfg.Init()
	if err = startCfg.Validate(); err != nil {
		return 0, err
	}
	if err = initializeCore(startCfg); err != nil {
		return 0, err
	}
	if startCfg.NativeMode {
		storedConfig, configErr := Downloader.GetConfig()
		if configErr != nil {
			return 0, configErr
		}
		apiConfig := storedConfig.API
		if apiConfig == nil {
			apiConfig = (&base.APIServerConfig{}).Init()
		} else {
			apiConfig.Init()
		}
		startCfg.ApiEnable = &apiConfig.Enable
		startCfg.Network = apiConfig.Network
		startCfg.Address = apiConfig.Address
		startCfg.ApiToken = apiConfig.Token
	}
	if err := Downloader.ContinueOnStartup(); err != nil {
		Downloader.Logger.Warn().Err(err).Msg("auto-start tasks failed")
	}
	if !startCfg.APIEnabled() {
		return 0, nil
	}
	config := &base.APIServerConfig{
		Enable:  true,
		Network: startCfg.Network,
		Address: startCfg.Address,
		Token:   startCfg.ApiToken,
	}
	port, startErr := apiServer.startWithConfigLocked(startCfg, config)
	if startErr != nil && startCfg.NativeMode {
		Downloader.Logger.Error().Err(startErr).Msg("optional API server failed to start")
		return 0, nil
	}
	return port, startErr
}

func Stop() {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if err := apiServer.stopLocked(); err != nil && Downloader != nil {
		Downloader.Logger.Warn().Err(err).Msg("shutdown server failed")
	}
	if APIService != nil {
		APIService.SubscribeTaskEvents(0, nil)
	}
	if Downloader != nil {
		if err := Downloader.Close(); err != nil {
			Downloader.Logger.Warn().Err(err).Msg("close downloader failed")
		}
	}
	APIService = nil
	apiServer = &apiServerManager{}
}

func BuildServer(startCfg *model.StartConfig) (*http.Server, net.Listener, error) {
	return buildServer(startCfg, true)
}

func buildServer(startCfg *model.StartConfig, continueOnStartup bool) (*http.Server, net.Listener, error) {
	if startCfg == nil {
		startCfg = &model.StartConfig{}
	}
	startCfg.Init()
	if err := startCfg.Validate(); err != nil {
		return nil, nil, err
	}

	if err := initializeCore(startCfg); err != nil {
		return nil, nil, err
	}

	if startCfg.Network == "unix" {
		util.SafeRemove(startCfg.Address)
	}

	listener, err := net.Listen(startCfg.Network, startCfg.Address)
	if err != nil {
		return nil, nil, err
	}
	if continueOnStartup {
		if err := Downloader.ContinueOnStartup(); err != nil {
			Downloader.Logger.Warn().Err(err).Msg("auto-start tasks failed")
		}
	}

	var r = mux.NewRouter()
	for _, route := range APIService.RouteSpecs() {
		route := route
		r.Methods(route.Method).Path(route.Pattern).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				WriteJson(w, model.NewErrorResult(err.Error()))
				return
			}
			response := route.Handler(&goapi.Context{
				Method:     r.Method,
				Path:       r.URL.Path,
				Query:      r.URL.Query(),
				Body:       body,
				Headers:    r.Header,
				PathParams: mux.Vars(r),
			})
			WriteStatusJson(w, response.StatusCode, response.Body)
		})
	}
	r.Path("/api/web/proxy").HandlerFunc(DoProxy)

	enableApiToken := startCfg.ApiToken != ""
	enableWebAuth := startCfg.WebEnable && startCfg.WebAuth != nil
	var webSessions *webSessionStore
	if enableWebAuth {
		webSessions = newWebSessionStore()
	}
	if startCfg.WebEnable {
		if enableWebAuth {
			r.Methods(http.MethodPost).Path("/api/web/login").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var loginReq model.WebAuth
				if ReadJson(r, w, &loginReq) {
					if loginReq.Username == startCfg.WebAuth.Username && loginReq.Password == startCfg.WebAuth.Password {
						sessionID, expiresAt, err := webSessions.create()
						if err != nil {
							WriteJson(w, model.NewErrorResult(err.Error()))
							return
						}
						http.SetCookie(w, &http.Cookie{
							Name:     webAuthCookieName,
							Value:    sessionID,
							Path:     "/",
							MaxAge:   int(webAuthSessionTTL.Seconds()),
							Expires:  expiresAt,
							HttpOnly: true,
							Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
							SameSite: http.SameSiteLaxMode,
						})

						WriteJson(w, model.NewNilResult())
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
	if enableApiToken || enableWebAuth {
		writeUnauthorized := func(w http.ResponseWriter, r *http.Request) {
			WriteStatusJson(w, http.StatusUnauthorized, model.NewErrorResult("unauthorized", model.CodeUnauthorized))
		}

		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				protectedPath := strings.HasPrefix(r.URL.Path, "/api/") ||
					r.URL.Path == "/fs/tasks" || strings.HasPrefix(r.URL.Path, "/fs/tasks/") ||
					r.URL.Path == "/fs/extensions" || strings.HasPrefix(r.URL.Path, "/fs/extensions/")
				if r.URL.Path == "/api/web/login" || !protectedPath {
					h.ServeHTTP(w, r)
					return
				}

				if enableApiToken {
					apiTokenHeader := r.Header["X-Api-Token"]
					// If an API token header is set, validate it before other authentication methods.
					if len(apiTokenHeader) > 0 {
						if apiTokenHeader[0] == startCfg.ApiToken {
							h.ServeHTTP(w, r)
							return
						}

						writeUnauthorized(w, r)
						return
					}
				}

				if enableWebAuth {
					cookie, err := r.Cookie(webAuthCookieName)
					if err != nil {
						writeUnauthorized(w, r)
						return
					}
					if !webSessions.valid(cookie.Value) {
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

	server := &http.Server{Handler: handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-Api-Token", "X-Target-Uri"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)(r)}
	return server, listener, nil
}

func startConfigForAPIServer(config *base.APIServerConfig) *model.StartConfig {
	enabled := config.Enable
	return &model.StartConfig{
		Network:        config.Network,
		Address:        config.Address,
		ApiEnable:      &enabled,
		ApiToken:       config.Token,
		NativeMode:     true,
		Storage:        model.StorageMem,
		WebEnable:      false,
		ProductionMode: true,
	}
}

func cloneAPIServerConfig(config *base.APIServerConfig) *base.APIServerConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	return &cloned
}

func sameAPIServerConfig(left, right *base.APIServerConfig) bool {
	return left != nil && right != nil &&
		left.Enable == right.Enable && left.Network == right.Network &&
		left.Address == right.Address && left.Token == right.Token
}

func initializeCore(startCfg *model.StartConfig) error {
	if APIService != nil {
		return nil
	}
	downloadCfg := &download.DownloaderConfig{
		ProductionMode:    startCfg.ProductionMode,
		RefreshInterval:   startCfg.RefreshInterval,
		WhiteDownloadDirs: startCfg.WhiteDownloadDirs,
		WebViewProvider:   startCfg.WebViewProvider,
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
		return err
	}
	service, err := goapi.NewService(Downloader)
	if err != nil {
		_ = Downloader.Close()
		Downloader = nil
		return err
	}
	APIService = service
	return nil
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
