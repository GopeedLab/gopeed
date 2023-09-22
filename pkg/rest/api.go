package rest

import (
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
	if ReadJson(r, w, &req) {
		rr, err := Downloader.Resolve(&req)
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(rr))
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTask
	if ReadJson(r, w, &req) {
		var (
			taskId string
			err    error
		)
		if req.Rid != "" {
			taskId, err = Downloader.Create(req.Rid, req.Opts)
		} else if req.Req != nil {
			taskId, err = Downloader.CreateDirect(req.Req, req.Opts)
		} else {
			WriteJson(w, model.NewErrorResult("param invalid: rid or req", model.CodeInvalidParam))
			return
		}
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(taskId))
	}
}

func CreateTaskBatch(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskBatch
	if ReadJson(r, w, &req) {
		if len(req.Reqs) == 0 {
			WriteJson(w, model.NewErrorResult("param invalid: reqs", model.CodeInvalidParam))
			return
		}
		taskIds, err := Downloader.CreateDirectBatch(req.Reqs, req.Opts)
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(taskIds))
	}
}

func PauseTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		WriteJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	if err := Downloader.Pause(taskId); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func ContinueTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		WriteJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	if err := Downloader.Continue(taskId); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func PauseAllTask(w http.ResponseWriter, r *http.Request) {
	if err := Downloader.PauseAll(); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func ContinueAllTask(w http.ResponseWriter, r *http.Request) {
	if err := Downloader.ContinueAll(); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	force := r.FormValue("force")
	if taskId == "" {
		WriteJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	if err := Downloader.Delete(taskId, force == "true"); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		WriteJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	task := Downloader.GetTask(taskId)
	if task == nil {
		WriteJson(w, model.NewErrorResult("task not found", model.CodeTaskNotFound))
		return
	}
	WriteJson(w, model.NewOkResult(task))
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	status := r.FormValue("status")
	tasks := Downloader.GetTasks()
	if status == "" {
		WriteJson(w, model.NewOkResult(tasks))
		return
	}
	statusArr := strings.Split(status, ",")
	result := make([]*download.Task, 0)
	for _, task := range tasks {
		for _, s := range statusArr {
			if task.Status == base.Status(s) {
				result = append(result, task)
			}
		}
	}
	WriteJson(w, model.NewOkResult(result))
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	WriteJson(w, model.NewOkResult(getServerConfig()))
}

func PutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg download.DownloaderStoreConfig
	if ReadJson(r, w, &cfg) {
		if err := Downloader.PutConfig(&cfg); err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	WriteJson(w, model.NewNilResult())
}

func InstallExtension(w http.ResponseWriter, r *http.Request) {
	var req model.InstallExtension
	if ReadJson(r, w, &req) {
		installedExt, err := Downloader.InstallExtensionByGit(req.URL)
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(installedExt.Identity))
	}
}

func GetExtensions(w http.ResponseWriter, r *http.Request) {
	list := Downloader.GetExtensions()
	WriteJson(w, model.NewOkResult(list))
}

func GetExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	ext, err := Downloader.GetExtension(identity)
	if err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewOkResult(ext))
}

func UpdateExtensionSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	var req model.UpdateExtensionSettings
	if ReadJson(r, w, &req) {
		if err := Downloader.UpdateExtensionSettings(identity, req.Settings); err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	WriteJson(w, model.NewNilResult())
}

func SwitchExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	var switchExtension model.SwitchExtension
	if ReadJson(r, w, &switchExtension) {
		if err := Downloader.SwitchExtension(identity, switchExtension.Status); err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	WriteJson(w, model.NewNilResult())
}

func DeleteExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	if err := Downloader.DeleteExtension(identity); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func UpdateCheckExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	newVersion, err := Downloader.UpgradeCheckExtension(identity)
	if err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewOkResult(&model.UpdateCheckExtensionResp{
		NewVersion: newVersion,
	}))
}

func UpgradeExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	if err := Downloader.UpgradeExtension(identity); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
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

func writeError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(msg))
}

func getServerConfig() *download.DownloaderStoreConfig {
	cfg, _ := Downloader.GetConfig()
	return cfg
}
