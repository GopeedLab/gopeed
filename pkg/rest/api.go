package rest

import (
	"compress/gzip"
	"github.com/gorilla/mux"
	"github.com/monkeyWie/gopeed/pkg/base"
	"github.com/monkeyWie/gopeed/pkg/download"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
	"github.com/monkeyWie/gopeed/pkg/rest/util"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func Resolve(w http.ResponseWriter, r *http.Request) {
	var req base.Request
	if util.ReadJson(w, r, &req) {
		resource, err := Downloader.Resolve(&req)
		if err != nil {
			util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
			return
		}
		util.WriteJsonOk(w, model.NewResultWithData(resource))
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTask
	if util.ReadJson(w, r, &req) {
		taskId, err := Downloader.Create(req.Res, req.Opts)
		if err != nil {
			util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
			return
		}
		util.WriteJsonOk(w, model.NewResultWithData(taskId))
	}
}

func PauseTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if err := Downloader.Pause(taskId); err != nil {
		util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
		return
	}
	util.WriteJsonOk(w, nil)
}

func ContinueTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if err := Downloader.Continue(taskId); err != nil {
		util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
		return
	}
	util.WriteJsonOk(w, nil)
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	force := r.FormValue("force")
	if err := Downloader.Delete(taskId, force == "true"); err != nil {
		util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
		return
	}
	util.WriteJsonOk(w, nil)
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	task := Downloader.GetTask(taskId)
	if task == nil {
		util.WriteJson(w, http.StatusNotFound, model.NewResultWithMsg("task not found"))
		return
	}
	util.WriteJsonOk(w, model.NewResultWithData(task))
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	status := r.FormValue("status")
	if status == "" {
		util.WriteJson(w, http.StatusBadRequest, model.NewResultWithMsg("param is required: status"))
		return
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
	util.WriteJsonOk(w, model.NewResultWithData(result))
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	util.WriteJsonOk(w, model.NewResultWithData(getServerConfig()))
}

func PutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg download.DownloaderStoreConfig
	if util.ReadJson(w, r, &cfg) {
		if err := Downloader.PutConfig(&cfg); err != nil {
			util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
			return
		}
	}
	util.WriteJsonOk(w, nil)
}

func DoProxy(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Target-Uri")
	if target == "" {
		util.WriteJson(w, http.StatusBadRequest, model.NewResultWithMsg("header is required: X-Target-Uri"))
		return
	}
	targetUrl, err := url.Parse(target)
	if err != nil {
		util.WriteJson(w, http.StatusBadRequest, model.NewResultWithMsg(err.Error()))
		return
	}
	r.RequestURI = ""
	r.URL = targetUrl
	r.Host = targetUrl.Host
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		util.WriteJson(w, http.StatusBadRequest, model.NewResultWithMsg(err.Error()))
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
		util.WriteJson(w, http.StatusBadRequest, model.NewResultWithMsg(err.Error()))
		return
	}
}

func DoCommand(w http.ResponseWriter, r *http.Request) {
	var action model.Command
	if util.ReadJson(w, r, &action) {
		ret, err := Downloader.Handle(action.Protocol, action.Action, action.Params)
		if err != nil {
			util.WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
			return
		}
		util.WriteJsonOk(w, model.NewResultWithData(ret))
		return
	}
	util.WriteJsonOk(w, nil)
}

func getServerConfig() *download.DownloaderStoreConfig {
	_, cfg, _ := Downloader.GetConfig()
	return cfg
}
