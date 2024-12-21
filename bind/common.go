package bind

import (
	"encoding/json"
	"errors"
	"github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/api/model"
)

// Global singleton instance
var instance *api.Instance

func Init(config *model.StartConfig) (err error) {
	if instance != nil {
		return nil
	}

	config.ProductionMode = true
	instance, err = api.Create(config)
	return
}

func Invoke(request *api.Request) string {
	if instance == nil {
		return BuildResult(errors.New("instance not initialized"))
	}
	return jsonEncode(api.Invoke(instance, request))
}

func BuildResult(data any) string {
	if err, ok := data.(error); ok {
		buf, _ := json.Marshal(model.NewErrorResult[any](err.Error()))
		return string(buf)
	}
	return jsonEncode(model.NewOkResult(data))
}

func jsonEncode(data any) string {
	buf, _ := json.Marshal(data)
	return string(buf)
}
