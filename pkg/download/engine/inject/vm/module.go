package vm

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type Vm struct {
	loop *eventloop.EventLoop
}

func (vm *Vm) Set(name string, value any) {
	vm.loop.Run(func(runtime *goja.Runtime) {
		runtime.Set(name, value)
	})
}

func (vm *Vm) Get(name string) (value any) {
	vm.loop.Run(func(runtime *goja.Runtime) {
		value = runtime.Get(name)
	})
	return
}

func (vm *Vm) RunString(script string) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	vm.loop.Run(func(runtime *goja.Runtime) {
		value, err = runtime.RunString(script)
	})
	return
}

func Enable(runtime *goja.Runtime) error {
	return runtime.Set("__gopeed_create_vm", func(call goja.FunctionCall) goja.Value {
		return runtime.ToValue(&Vm{
			loop: eventloop.NewEventLoop(),
		})
	})
}
