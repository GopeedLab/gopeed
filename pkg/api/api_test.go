package api

import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"github.com/GopeedLab/gopeed/pkg/download"
	"testing"
)

var instance, _ = Create(nil)

func TestInvoke(t *testing.T) {
	doTestInvoke(t, "Info", []any{}, model.CodeOk)
	doTestInvoke(t, "Info1", []any{}, model.CodeError)
	doTestInvoke(t, "CreateTask", []any{nil}, model.CodeError)
	doTestInvoke(t, "GetTasks", nil, model.CodeError)
	doTestInvoke(t, "GetTasks", []any{}, model.CodeError)
	doTestInvoke(t, "GetTasks", []any{"{abc:123"}, model.CodeError)
	doTestInvoke(t, "GetTasks", []any{&download.TaskFilter{}}, model.CodeOk)
	doTestInvoke(t, "SwitchExtension", []any{"test", &model.SwitchExtension{Status: true}}, model.CodeError)
}

func doTestInvoke(t *testing.T, method string, params []any, expectCode model.RespCode) {
	result := Invoke(instance, &Request{
		Method: method,
		Params: params,
	})
	buf, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var res model.Result[any]
	if err := json.Unmarshal(buf, &res); err != nil {
		t.Fatal(err)
	}

	if res.Code != expectCode {
		t.Fatalf("Invoke method [%s] failed, expect code [%d], got [%d], msg: %s", method, expectCode, res.Code, res.Msg)
	}
}
