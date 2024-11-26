package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const instanceKey string = "apiInstance"

func ListenHttp(httpCfg *base.DownloaderHttpConfig, ai *Instance) error {
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
	r.Methods(http.MethodGet).Path("/api/v1/tasks/{id}/stats").HandlerFunc(GetTaskStats)
	r.Methods(http.MethodGet).Path("/api/v1/config").HandlerFunc(GetConfig)
	r.Methods(http.MethodPut).Path("/api/v1/config").HandlerFunc(PutConfig)
	r.Methods(http.MethodPost).Path("/api/v1/extensions").HandlerFunc(InstallExtension)
	r.Methods(http.MethodGet).Path("/api/v1/extensions").HandlerFunc(GetExtensions)
	r.Methods(http.MethodGet).Path("/api/v1/extensions/{identity}").HandlerFunc(GetExtension)
	r.Methods(http.MethodPut).Path("/api/v1/extensions/{identity}/settings").HandlerFunc(UpdateExtensionSettings)
	r.Methods(http.MethodPut).Path("/api/v1/extensions/{identity}/switch").HandlerFunc(SwitchExtension)
	r.Methods(http.MethodDelete).Path("/api/v1/extensions/{identity}").HandlerFunc(DeleteExtension)
	r.Methods(http.MethodGet).Path("/api/v1/extensions/{identity}/upgrade").HandlerFunc(UpgradeCheckExtension)
	r.Methods(http.MethodPost).Path("/api/v1/extensions/{identity}/upgrade").HandlerFunc(UpgradeExtension)
	r.Path("/api/v1/proxy").HandlerFunc(DoProxy)
	if ai.startCfg.WebEnable {
		r.PathPrefix("/fs/tasks").Handler(http.FileServer(&taskFileSystem{
			downloader: ai.downloader,
		}))
		r.PathPrefix("/fs/extensions").Handler(http.FileServer(&extensionFileSystem{
			downloader: ai.downloader,
		}))
		r.PathPrefix("/").Handler(http.FileServer(http.FS(ai.startCfg.WebFS)))
	}

	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), instanceKey, ai)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	if httpCfg.ApiToken != "" || (ai.startCfg.WebEnable && ai.startCfg.WebBasicAuth != nil) {
		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if httpCfg.ApiToken != "" && r.Header.Get("X-Api-Token") == httpCfg.ApiToken {
					h.ServeHTTP(w, r)
					return
				}
				if ai.startCfg.WebEnable && ai.startCfg.WebBasicAuth != nil {
					if r.Header.Get("Authorization") == ai.startCfg.WebBasicAuth.Authorization() {
						h.ServeHTTP(w, r)
						return
					}
					w.Header().Set("WWW-Authenticate", "Basic realm=\"gopeed web\"")
				}
				WriteStatusJson(w, http.StatusUnauthorized, model.NewErrorResult[any]("unauthorized", model.CodeUnauthorized))
			})
		})
	}

	// recover panic
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					err := errors.WithStack(fmt.Errorf("%v", v))
					ai.downloader.Logger.Error().Stack().Err(err).Msgf("http server panic: %s %s", r.Method, r.RequestURI)
					WriteJson(w, model.NewErrorResult[any](err.Error(), model.CodeError))
				}
			}()
			h.ServeHTTP(w, r)
		})
	})

	srv := &http.Server{Handler: handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type", "X-Api-Token", "X-Target-Uri"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
	)(r)}

	listener, err := net.Listen("tcp", net.JoinHostPort(httpCfg.Host, fmt.Sprintf("%d", httpCfg.Port)))
	if err != nil {
		return err
	}

	ai.srv = srv
	ai.listener = listener
	return nil
}

func getInstance(r *http.Request) *Instance {
	return r.Context().Value(instanceKey).(*Instance)
}

func Info(w http.ResponseWriter, r *http.Request) {
	WriteJson(w, getInstance(r).Info())
}

func Resolve(w http.ResponseWriter, r *http.Request) {
	var req base.Request
	if ReadJson(r, w, &req) {
		WriteJson(w, getInstance(r).Resolve(&req))
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTask
	if ReadJson(r, w, &req) {
		WriteJson(w, getInstance(r).CreateTask(&req))
	}
}

func CreateTaskBatch(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskBatch
	if ReadJson(r, w, &req) {
		WriteJson(w, getInstance(r).CreateTaskBatch(&req))
	}
}

func PauseTask(w http.ResponseWriter, r *http.Request) {
	var taskId string
	if !parseTaskId(r, w, &taskId) {
		return
	}
	WriteJson(w, getInstance(r).PauseTask(taskId))
}

func PauseTasks(w http.ResponseWriter, r *http.Request) {
	var filter download.TaskFilter
	if !parseFilter(r, w, &filter) {
		return
	}
	WriteJson(w, getInstance(r).PauseTasks(&filter))
}

func ContinueTask(w http.ResponseWriter, r *http.Request) {
	var taskId string
	if !parseTaskId(r, w, &taskId) {
		return
	}
	WriteJson(w, getInstance(r).ContinueTask(taskId))
}

func ContinueTasks(w http.ResponseWriter, r *http.Request) {
	var filter download.TaskFilter
	if !parseFilter(r, w, &filter) {
		return
	}
	WriteJson(w, getInstance(r).ContinueTasks(&filter))
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	var taskId string
	if !parseTaskId(r, w, &taskId) {
		return
	}
	force := r.FormValue("force")
	WriteJson(w, getInstance(r).DeleteTask(taskId, force == "true"))
}

func DeleteTasks(w http.ResponseWriter, r *http.Request) {
	var filter download.TaskFilter
	if !parseFilter(r, w, &filter) {
		return
	}
	force := r.FormValue("force")
	WriteJson(w, getInstance(r).DeleteTasks(&filter, force == "true"))
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	var taskId string
	if !parseTaskId(r, w, &taskId) {
		return
	}
	WriteJson(w, getInstance(r).GetTask(taskId))
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	var filter download.TaskFilter
	if !parseFilter(r, w, &filter) {
		return
	}
	WriteJson(w, getInstance(r).GetTasks(&filter))
}

func GetTaskStats(w http.ResponseWriter, r *http.Request) {
	var taskId string
	if !parseTaskId(r, w, &taskId) {
		return
	}
	WriteJson(w, getInstance(r).GetTaskStats(taskId))
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	WriteJson(w, getInstance(r).GetConfig())
}

func PutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg base.DownloaderStoreConfig
	if !ReadJson(r, w, &cfg) {
		return
	}
	WriteJson(w, getInstance(r).PutConfig(&cfg))
}

func InstallExtension(w http.ResponseWriter, r *http.Request) {
	var req model.InstallExtension
	if !ReadJson(r, w, &req) {
		return
	}
	WriteJson(w, getInstance(r).InstallExtension(&req))
}

func GetExtensions(w http.ResponseWriter, r *http.Request) {
	WriteJson(w, getInstance(r).GetExtensions())
}

func GetExtension(w http.ResponseWriter, r *http.Request) {
	var identity string
	if !parseIdentity(r, w, &identity) {
		return
	}
	WriteJson(w, getInstance(r).GetExtension(identity))
}

func UpdateExtensionSettings(w http.ResponseWriter, r *http.Request) {
	var identity string
	if !parseIdentity(r, w, &identity) {
		return
	}
	var req model.UpdateExtensionSettings
	if !ReadJson(r, w, &req) {
		return
	}
	WriteJson(w, getInstance(r).UpdateExtensionSettings(identity, &req))
}

func SwitchExtension(w http.ResponseWriter, r *http.Request) {
	var identity string
	if !parseIdentity(r, w, &identity) {
		return
	}
	var switchExtension model.SwitchExtension
	if !ReadJson(r, w, &switchExtension) {
		return
	}
	WriteJson(w, getInstance(r).SwitchExtension(identity, &switchExtension))
}

func DeleteExtension(w http.ResponseWriter, r *http.Request) {
	var identity string
	if !parseIdentity(r, w, &identity) {
		return
	}
	WriteJson(w, getInstance(r).DeleteExtension(identity))
}

func UpgradeCheckExtension(w http.ResponseWriter, r *http.Request) {
	var identity string
	if !parseIdentity(r, w, &identity) {
		return
	}
	WriteJson(w, getInstance(r).UpgradeCheckExtension(identity))
}

func UpgradeExtension(w http.ResponseWriter, r *http.Request) {
	var identity string
	if !parseIdentity(r, w, &identity) {
		return
	}
	WriteJson(w, getInstance(r).UpgradeExtension(identity))
}

func DoProxy(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Target-Uri")
	if target == "" {
		writeError(w, "param invalid: X-Target-Uri")
		return
	}
	targetUrl, err := url.Parse(target)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	r.RequestURI = ""
	r.URL = targetUrl
	r.Host = targetUrl.Host
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	w.Write(buf)
}

func parsePathParam(r *http.Request, w http.ResponseWriter, name string, value *string) bool {
	vars := mux.Vars(r)
	*value = vars[name]
	if *value == "" {
		WriteJson(w, model.NewErrorResult[any]("param invalid: id", model.CodeInvalidParam))
		return false
	}
	return true
}

func parseTaskId(r *http.Request, w http.ResponseWriter, taskId *string) bool {
	return parsePathParam(r, w, "id", taskId)
}

func parseIdentity(r *http.Request, w http.ResponseWriter, identity *string) bool {
	return parsePathParam(r, w, "identity", identity)
}

func parseFilter(r *http.Request, w http.ResponseWriter, filter *download.TaskFilter) bool {
	if err := r.ParseForm(); err != nil {
		WriteJson(w, model.NewErrorResult[any](err.Error()))
		return false
	}

	filter.IDs = r.Form["id"]
	filter.Statuses = convertStatues(r.Form["status"])
	filter.NotStatuses = convertStatues(r.Form["notStatus"])
	return true
}

func convertStatues(statues []string) []base.Status {
	result := make([]base.Status, 0)
	for _, status := range statues {
		result = append(result, base.Status(status))
	}
	return result
}

func writeError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(msg))
}

func ReadJson(r *http.Request, w http.ResponseWriter, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteJson(w, model.NewErrorResult[any](err.Error()))
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
	downloader *download.Downloader
}

func (e *taskFileSystem) Open(name string) (http.File, error) {
	// get extension identity
	identity, path, err := resolvePath(name, "/fs/tasks")
	if err != nil {
		return nil, err
	}
	task := e.downloader.GetTask(identity)
	if task == nil {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(task.Meta.RootDirPath(), path))
}

// handle extension file resource
type extensionFileSystem struct {
	downloader *download.Downloader
}

func (e *extensionFileSystem) Open(name string) (http.File, error) {
	// get extension identity
	identity, path, err := resolvePath(name, "/fs/extensions")
	if err != nil {
		return nil, err
	}
	extension, err := e.downloader.GetExtension(identity)
	if err != nil {
		return nil, os.ErrNotExist
	}
	extensionPath := e.downloader.ExtensionPath(extension)
	return os.Open(filepath.Join(extensionPath, path))
}
