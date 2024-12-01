package bind

import (
	"encoding/json"
	"errors"
	"github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"sync"
)

type instanceArray struct {
	mu    sync.RWMutex
	array []*api.Instance
}

// append appends an instance to the array.
// Returns the index of the instance in the array.
func (ia *instanceArray) append(instance *api.Instance) int {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	ia.array = append(ia.array, instance)
	return len(ia.array) - 1
}

func (ia *instanceArray) get(index int) (*api.Instance, bool) {
	ia.mu.RLock()
	defer ia.mu.RUnlock()
	if index < 0 || index >= len(ia.array) {
		return nil, false
	}
	return ia.array[index], true
}

var instances = &instanceArray{
	mu:    sync.RWMutex{},
	array: make([]*api.Instance, 0),
}

func Create(config *model.StartConfig) string {
	config.ProductionMode = true
	instance, err := api.Create(config)
	if err != nil {
		return BuildResult(err)
	}
	return BuildResult(instances.append(instance))
}

func Invoke(index int, request *api.Request) string {
	instance, ok := instances.get(index)
	if !ok {
		return BuildResult(errors.New("instance not found"))
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
