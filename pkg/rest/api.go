package rest

import (
	"github.com/gorilla/mux"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/download"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"github.com/monkeyWie/gopeed-core/pkg/rest/util"
	"net/http"
	"strings"
)

func Resolve(w http.ResponseWriter, r *http.Request) {
	var req base.Request
	if util.ReadJson(w, r, &req) {
		resource, err := Downloader.Resolve(&req)
		if err != nil {
			util.WriteJson(w, 500, model.NewResultWithMsg(err.Error()))
			return
		}
		util.WriteJsonOk(w, model.NewResultWithData(resource))
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskReq
	if util.ReadJson(w, r, &req) {
		taskId, err := Downloader.Create(req.Resource, req.Options)
		if err != nil {
			util.WriteJson(w, 500, model.NewResultWithMsg(err.Error()))
			return
		}
		util.WriteJsonOk(w, model.NewResultWithData(taskId))
	}
}

func PauseTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	Downloader.Pause(taskId)
	util.WriteJsonOk(w, nil)
}

func ContinueTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	Downloader.Continue(taskId)
	util.WriteJsonOk(w, nil)
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	task := Downloader.GetTask(taskId)
	if task == nil {
		util.WriteJson(w, 404, model.NewResultWithMsg("task not found"))
		return
	}
	util.WriteJsonOk(w, model.NewResultWithData(task))
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	status := r.FormValue("status")
	if status == "" {
		util.WriteJson(w, 400, model.NewResultWithMsg("param is required: status"))
		return
	}
	statusArr := strings.Split(status, ",")
	tasks := Downloader.GetTasks()
	var result []*download.Task
	for _, task := range tasks {
		for _, s := range statusArr {
			if task.Status == base.Status(s) {
				result = append(result, task)
			}
		}
	}
	util.WriteJsonOk(w, model.NewResultWithData(tasks))
}
