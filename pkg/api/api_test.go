package api

import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"github.com/GopeedLab/gopeed/pkg/download"
	"testing"
)

var instance, _ = Create(nil)

func TestInvoke(t *testing.T) {
	doTestInvoke(t, "Info", []any{})
	doTestInvoke(t, "GetTasks", nil)
	doTestInvoke(t, "GetTasks", []any{})
	doTestInvoke(t, "GetTasks", []any{nil})
	doTestInvoke(t, "GetTasks", []any{&download.TaskFilter{}})
}

func doTestInvoke(t *testing.T, method string, params []any) *model.Result[any] {
	reqParams := make([]string, len(params))
	for i, p := range params {
		bytes, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		reqParams[i] = string(bytes)
	}

	result := Invoke(instance, &Request{
		Method: method,
		Params: reqParams,
	})
	buf, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var res model.Result[any]
	if err := json.Unmarshal(buf, &res); err != nil {
		t.Fatal(err)
	}
	return &res
}
