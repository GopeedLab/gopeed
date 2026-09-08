package rest

import (
	"encoding/json"
	"net/url"

	goapi "github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

const (
	TaskEventDone  = uint64(goapi.TaskEventDone)
	TaskEventError = uint64(goapi.TaskEventError)
)

// Dispatch executes one in-process API request. Native scheduling and callback
// behavior belong to bind/native; this function only bridges into the shared
// API Service without opening an HTTP listener.
func Dispatch(method, path, rawQuery, body string) string {
	runtimeMu.RLock()
	service := APIService
	runtimeMu.RUnlock()
	if service == nil {
		return marshalInvokeResult(model.NewErrorResult("service not started"))
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return marshalInvokeResult(model.NewErrorResult(err.Error()))
	}
	_, data := service.JSON(method, path, query, []byte(body), nil)
	return string(data)
}

func SubscribeTaskEvents(mask uint64, listener func(payload string)) {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	if APIService == nil {
		return
	}
	APIService.SubscribeTaskEvents(goapi.TaskEventMask(mask), listener)
}

func marshalInvokeResult(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `{"code":1000,"msg":"marshal response failed"}`
	}
	return string(data)
}
