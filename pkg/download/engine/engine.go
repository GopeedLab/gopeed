package engine

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/formdata"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/xhr"
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

// RunString executes the script and returns the go type value
// if script result is promise, it will be resolved
func (e *Engine) RunString(script string) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	var result goja.Value
	e.loop.Run(func(runtime *goja.Runtime) {
		result, err = runtime.RunString(script)
	})
	if err != nil {
		return
	}
	return resolveResult(result)
}

// CallFunction calls the function and returns the go type value
// if function result is promise, it will be resolved
func (e *Engine) CallFunction(fn goja.Callable, args ...any) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	var result goja.Value
	e.loop.Run(func(runtime *goja.Runtime) {
		if args == nil {
			result, err = fn(nil)
		} else {
			var jsArgs []goja.Value
			for _, arg := range args {
				jsArgs = append(jsArgs, runtime.ToValue(arg))
			}
			result, err = fn(nil, jsArgs...)
		}
	})
	if err != nil {
		return nil, err
	}
	return resolveResult(result)
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
		runtime.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
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

func Run(script string) (value any, err error) {
	engine := NewEngine()
	return engine.RunString(script)
}

// if the value is Promise, it will be resolved and return the result.
func resolveResult(value goja.Value) (any, error) {
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
