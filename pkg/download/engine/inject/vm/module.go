package vm

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type Vm struct {
	loop *eventloop.EventLoop
}

func (vm *Vm) Set(name string, value any) {
	defer func() {
		if r := recover(); r != nil {
			// silently ignore panic in Set
		}
	}()
	vm.loop.Run(func(runtime *goja.Runtime) {
		runtime.Set(name, value)
	})
}

func (vm *Vm) Get(name string) (value any) {
	defer func() {
		if r := recover(); r != nil {
			// silently ignore panic in Get
		}
	}()
	vm.loop.Run(func(runtime *goja.Runtime) {
		value = runtime.Get(name)
	})
	return
}

func (vm *Vm) RunString(script string) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()

	vm.loop.Run(func(runtime *goja.Runtime) {
		value, err = runtime.RunString(script)
	})
	return
}

func (vm *Vm) Close() {
	if vm.loop != nil {
		vm.loop.Stop()
	}
}

func Enable(runtime *goja.Runtime) error {
	return runtime.Set("__gopeed_create_vm", func(call goja.FunctionCall) goja.Value {
		loop := eventloop.NewEventLoop()
		loop.Start()
		v := &Vm{
			loop: loop,
		}
		return runtime.ToValue(v)
	})
}
