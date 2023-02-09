package extension

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/download/extension/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/extension/inject/formdata"
	"github.com/GopeedLab/gopeed/pkg/download/extension/inject/xhr"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/url"
)

//go:embed polyfill/dist/index.js
var polyfillScript string

type Engine struct {
	loop *eventloop.EventLoop

	Runtime *goja.Runtime
}

func (e *Engine) RunString(script string) (value goja.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	e.loop.Run(func(runtime *goja.Runtime) {
		value, err = runtime.RunString(script)
	})
	return
}

func (e *Engine) RunNative(cb goja.Callable, args ...goja.Value) (value goja.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	e.loop.Run(func(runtime *goja.Runtime) {
		if args == nil {
			value, err = cb(nil)
		} else {
			value, err = cb(nil, args...)
		}
	})
	return
}

func (e *Engine) Close() {
	e.loop.Stop()
}

func NewEngine() *Engine {
	loop := eventloop.NewEventLoop()
	engine := &Engine{
		loop: loop,
	}
	loop.Run(func(runtime *goja.Runtime) {
		engine.Runtime = runtime
		url.Enable(runtime)
		if err := file.Enable(runtime); err != nil {
			return
		}
		if err := formdata.Enable(runtime); err != nil {
			return
		}
		if err := xhr.Enable(runtime); err != nil {
			return
		}
		if _, err := runtime.RunString(polyfillScript); err != nil {
			return
		}
		return
	})
	return engine
}

func Run(script string) (value goja.Value, err error) {
	engine := NewEngine()
	return engine.RunString(script)
}

// ResolveResult if the value is Promise, it will be resolved and return the result.
func ResolveResult(value goja.Value) (any, error) {
	export := value.Export()
	switch export.(type) {
	case *goja.Promise:
		p := export.(*goja.Promise)
		switch p.State() {
		case goja.PromiseStatePending:
			return nil, nil
		case goja.PromiseStateFulfilled:
			return p.Result().Export(), nil
		case goja.PromiseStateRejected:
			return nil, errors.New(p.Result().String())
		}
	}
	return export, nil
}
