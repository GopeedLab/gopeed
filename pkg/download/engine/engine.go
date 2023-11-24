package engine

import (
	_ "embed"
	"errors"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/formdata"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/vm"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/xhr"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/process"
	"github.com/dop251/goja_nodejs/url"
	"sync"
	"time"
)

//go:embed polyfill/out/index.js
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
			err = r.(error)
		}
	}()

	var result goja.Value
	result, err = e.runAndDone(func(runtime *goja.Runtime) (goja.Value, error) {
		return runtime.RunString(script)
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
			err = r.(error)
		}
	}()

	var result goja.Value
	result, err = e.runAndDone(func(runtime *goja.Runtime) (goja.Value, error) {
		if args == nil {
			return fn(nil)
		} else {
			var jsArgs []goja.Value
			for _, arg := range args {
				jsArgs = append(jsArgs, runtime.ToValue(arg))
			}
			return fn(nil, jsArgs...)
		}
	})
	if err != nil {
		return nil, err
	}
	return resolveResult(result)
}

// loop.Run will hang if the script result has a non-stop code, such as setInterval.
// Therefore, a trick must be used to run the script and wait for it to complete
func (e *Engine) runAndDone(fn func(runtime *goja.Runtime) (goja.Value, error)) (result goja.Value, err error) {
	var finished sync.WaitGroup
	finished.Add(1)
	go func() {
		e.loop.Run(func(runtime *goja.Runtime) {
			result, err = fn(runtime)
			finished.Done()
		})
	}()
	finished.Wait()

	if err != nil {
		return
	}

	// if result is promise, wait for it to be resolved
	if p, ok := result.Export().(*goja.Promise); ok {
		waitCh := make(chan struct{})

		// check promise state every 100 milliseconds, until it is resolved
		go func() {
			defer close(waitCh)
			for {
				if p.State() == goja.PromiseStatePending {
					time.Sleep(time.Millisecond * 100)
					continue
				}
				break
			}
		}()

		// if the promise is not resolved within 120 seconds, it will be forced to resolve
		select {
		case <-waitCh:
		case <-time.After(time.Second * 120):
			err = errors.New("promise timeout")
		}
	}

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
		runtime.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		vm.Enable(runtime)
		url.Enable(runtime)
		process.Enable(runtime)
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
		// polyfill global
		if err := runtime.Set("global", runtime.GlobalObject()); err != nil {
			return
		}
		// polyfill window.location
		if _, err := runtime.RunString("global.location = new URL('http://localhost');"); err != nil {
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
