package rest

import (
	"compress/gzip"
	"encoding/json"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func Resolve(w http.ResponseWriter, r *http.Request) {
	var req base.Request
	if readJson(r, w, &req) {
		rr, err := Downloader.Resolve(&req)
		if err != nil {
			writeJson(w, model.NewErrorResult(err.Error()))
			return
		}
		writeJson(w, model.NewOkResult(rr))
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTask
	if readJson(r, w, &req) {
		var (
			taskId string
			err    error
		)
		if req.Rid != "" {
			taskId, err = Downloader.Create(req.Rid, req.Opts)
		} else if req.Req != nil {
			taskId, err = Downloader.DirectCreate(req.Req, req.Opts)
		} else {
			writeJson(w, model.NewErrorResult("param invalid: rid or req", model.CodeInvalidParam))
			return
		}
		if err != nil {
			writeJson(w, model.NewErrorResult(err.Error()))
			return
		}
		writeJson(w, model.NewOkResult(taskId))
	}
}

func PauseTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		writeJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	if err := Downloader.Pause(taskId); err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return
	}
	writeJson(w, model.NewOkResult(nil))
}

func ContinueTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		writeJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	if err := Downloader.Continue(taskId); err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return
	}
	writeJson(w, model.NewOkResult(nil))
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	force := r.FormValue("force")
	if taskId == "" {
		writeJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	if err := Downloader.Delete(taskId, force == "true"); err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return
	}
	writeJson(w, model.NewOkResult(nil))
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		writeJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	task := Downloader.GetTask(taskId)
	if task == nil {
		writeJson(w, model.NewErrorResult("task not found", model.CodeTaskNotFound))
		return
	}
	writeJson(w, model.NewOkResult(task))
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	status := r.FormValue("status")
	if status == "" {
		status = strings.Join([]string{
			string(base.DownloadStatusReady),
			string(base.DownloadStatusRunning),
			string(base.DownloadStatusPause),
			string(base.DownloadStatusError),
			string(base.DownloadStatusDone),
		}, ",")
	}
	statusArr := strings.Split(status, ",")
	tasks := Downloader.GetTasks()
	result := make([]*download.Task, 0)
	for _, task := range tasks {
		for _, s := range statusArr {
			if task.Status == base.Status(s) {
				result = append(result, task)
			}
		}
	}
	writeJson(w, model.NewOkResult(result))
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	writeJson(w, model.NewOkResult(getServerConfig()))
}

func PutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg download.DownloaderStoreConfig
	if readJson(r, w, &cfg) {
		if err := Downloader.PutConfig(&cfg); err != nil {
			writeJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	writeJson(w, model.NewOkResult(nil))
}

func DoProxy(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Target-Uri")
	if target == "" {
		writeJson(w, model.NewErrorResult("param invalid: X-Target-Uri", model.CodeInvalidParam))
		return
	}
	targetUrl, err := url.Parse(target)
	if err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return
	}
	r.RequestURI = ""
	r.URL = targetUrl
	r.Host = targetUrl.Host
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		defer reader.Close()
	default:
		reader = resp.Body
	}
	if _, err := io.Copy(w, reader); err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return
	}
}

func getServerConfig() *download.DownloaderStoreConfig {
	_, cfg, _ := Downloader.GetConfig()
	return cfg
}

func readJson(r *http.Request, w http.ResponseWriter, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJson(w, model.NewErrorResult(err.Error()))
		return false
	}
	return true
}

func writeJson(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}
