package engine

import (
	_ "embed"
	"errors"
	"fmt"
	"sync"

	"github.com/GopeedLab/gopeed/pkg/base"
	gojaerror "github.com/GopeedLab/gopeed/pkg/download/engine/inject/error"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/formdata"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/stream"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/vm"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/xhr"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	gojaurl "github.com/dop251/goja_nodejs/url"
)

//go:embed polyfill/out/index.js
var polyfillScript string

type Engine struct {
	loop *eventloop.EventLoop

	Runtime *goja.Runtime
}

type JSFunction func(goja.FunctionCall) goja.Value

// RunString executes the script and returns the go type value
// if script result is promise, it will be resolved
func (e *Engine) RunString(script string) (value any, err error) {
	return e.runOnLoop(func(runtime *goja.Runtime) (goja.Value, error) {
		return runtime.RunString(script)
	})
}

// CallFunction calls the function and returns the go type value
// if function result is promise, it will be resolved
func (e *Engine) CallFunction(fn any, args ...any) (value any, err error) {
	return e.runOnLoop(func(runtime *goja.Runtime) (goja.Value, error) {
		var jsArgs []goja.Value
		for _, arg := range args {
			jsArgs = append(jsArgs, runtime.ToValue(arg))
		}
		switch f := fn.(type) {
		case goja.Callable:
			if args == nil {
				return f(nil)
			}
			return f(nil, jsArgs...)
		case JSFunction:
			return callExportedFunction(func(call goja.FunctionCall) goja.Value {
				return f(call)
			}, jsArgs...)
		default:
			return nil, fmt.Errorf("unsupported function type: %T", fn)
		}
	})
}

func (e *Engine) runOnLoop(fn func(runtime *goja.Runtime) (goja.Value, error)) (any, error) {
	type result struct {
		value any
		err   error
	}
	ch := make(chan result, 1)
	ok := e.loop.RunOnLoop(func(runtime *goja.Runtime) {
		var once sync.Once
		sendResult := func(res result) {
			once.Do(func() {
				ch <- res
			})
		}
		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case error:
					sendResult(result{err: v})
				case goja.Value:
					sendResult(result{err: exportJSError(v)})
				default:
					sendResult(result{err: fmt.Errorf("panic: %v", r)})
				}
			}
		}()
		value, err := fn(runtime)
		if err != nil {
			sendResult(result{err: err})
			return
		}
		if p, ok := value.Export().(*goja.Promise); ok {
			switch p.State() {
			case goja.PromiseStateFulfilled:
				sendResult(result{value: exportJSValue(p.Result())})
				return
			case goja.PromiseStateRejected:
				sendResult(result{err: exportJSError(p.Result())})
				return
			}
			promiseObj := value.ToObject(runtime)
			thenVal := promiseObj.Get("then")
			thenFn, ok := goja.AssertFunction(thenVal)
			if !ok {
				sendResult(result{err: errors.New("promise.then is not callable")})
				return
			}
			onFulfilled := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
				sendResult(result{value: exportJSValue(call.Argument(0))})
				return goja.Undefined()
			})
			onRejected := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
				sendResult(result{err: exportJSError(call.Argument(0))})
				return goja.Undefined()
			})
			if _, err := thenFn(promiseObj, onFulfilled, onRejected); err != nil {
				sendResult(result{err: err})
			}
			return
		}
		sendResult(result{value: exportJSValue(value)})
	})
	if !ok {
		return nil, errors.New("engine loop terminated")
	}
	res := <-ch
	if res.err != nil {
		return nil, res.err
	}
	return res.value, nil
}

func (e *Engine) Close() {
	e.loop.Terminate()
}

type Config struct {
	ProxyConfig  *base.DownloaderProxyConfig
	StreamConfig *stream.Config
}

func NewEngine(cfg *Config) *Engine {
	if cfg == nil {
		cfg = &Config{}
	}
	loop := eventloop.NewEventLoop()
	engine := &Engine{
		loop: loop,
	}
	loop.Start()
	done := make(chan struct{})
	loop.RunOnLoop(func(runtime *goja.Runtime) {
		defer close(done)
		engine.Runtime = runtime
		runtime.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		vm.Enable(runtime)
		gojaurl.Enable(runtime)
		if err := gojaerror.Enable(runtime); err != nil {
			return
		}
		if err := file.Enable(runtime); err != nil {
			return
		}
		if err := formdata.Enable(runtime); err != nil {
			return
		}
		if err := xhr.Enable(runtime, cfg.ProxyConfig.ToHandler()); err != nil {
			return
		}
		if _, err := runtime.RunString(polyfillScript); err != nil {
			return
		}
		// polyfill global
		if err := runtime.Set("global", runtime.GlobalObject()); err != nil {
			return
		}
		// polyfill window
		if err := runtime.Set("window", runtime.GlobalObject()); err != nil {
			return
		}
		// polyfill window.location
		if _, err := runtime.RunString("global.location = new URL('http://localhost');"); err != nil {
			return
		}
		if err := stream.Enable(runtime, loop, cfg.StreamConfig); err != nil {
			return
		}
		return
	})
	<-done
	return engine
}

func Run(script string) (value any, err error) {
	engine := NewEngine(nil)
	return engine.RunString(script)
}

func exportJSValue(value goja.Value) any {
	if value == nil {
		return nil
	}
	return value.Export()
}

func exportJSError(value goja.Value) error {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return errors.New("promise rejected")
	}
	if err, ok := value.Export().(error); ok {
		return err
	}
	stack := value.String()
	if ro, ok := value.(*goja.Object); ok {
		stackVal := ro.Get("stack")
		if stackVal != nil && stackVal.String() != "" {
			stack = stackVal.String()
		}
	}
	return errors.New(stack)
}

func callExportedFunction(fn func(goja.FunctionCall) goja.Value, args ...goja.Value) (ret goja.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case goja.Value:
				err = exportJSError(v)
			default:
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()
	return fn(goja.FunctionCall{
		This:      nil,
		Arguments: args,
	}), nil
}
