package rest

import (
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"github.com/monkeyWie/gopeed-core/pkg/rest/util"
	"net/http"
)

func Resolve(w http.ResponseWriter, r *http.Request) {
	var req base.Request
	if util.ReadJson(w, r, &req) {
		resource, err := Downloader.Resolve(&req)
		if err != nil {
			util.WriteJson(w, model.NewError(500, err.Error()))
			return
		}
		util.WriteJson(w, resource)
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskReq
	if util.ReadJson(w, r, &req) {
		taskId, err := Downloader.Create(req.Resource, req.Options)
		if err != nil {
			util.WriteJson(w, model.NewError(500, err.Error()))
			return
		}
		util.WriteJson(w, taskId)
	}
}
