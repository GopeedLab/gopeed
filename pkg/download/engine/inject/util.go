package inject

import (
	"github.com/dop251/goja"
)

func ThrowTypeError(vm *goja.Runtime, msg string) {
	panic(vm.NewTypeError(msg))
}
