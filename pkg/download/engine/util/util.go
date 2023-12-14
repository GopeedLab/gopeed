package util

import (
	"github.com/dop251/goja"
)

func ThrowTypeError(vm *goja.Runtime, msg string) {
	panic(vm.NewTypeError(msg))
}

func AssertError[T error](err error) (t T, r bool) {
	if err == nil {
		return
	}
	if e, ok := err.(T); ok {
		return e, true
	}
	if e, ok := err.(*goja.Exception); ok {
		if ee, okk := e.Value().Export().(T); okk {
			return ee, true
		}
	}
	return
}
