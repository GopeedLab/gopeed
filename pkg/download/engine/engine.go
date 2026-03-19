package engine

import (
	_ "embed"
	"errors"

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
	return e.runOnLoop(func(runtime *goja.Runtime) (goja.Value, error) {
		return runtime.RunString(script)
	})
}

// CallFunction calls the function and returns the go type value
// if function result is promise, it will be resolved
func (e *Engine) CallFunction(fn goja.Callable, args ...any) (value any, err error) {
	return e.runOnLoop(func(runtime *goja.Runtime) (goja.Value, error) {
		if args == nil {
			return fn(nil)
		}
		var jsArgs []goja.Value
		for _, arg := range args {
			jsArgs = append(jsArgs, runtime.ToValue(arg))
		}
		return fn(nil, jsArgs...)
	})
}

func (e *Engine) runOnLoop(fn func(runtime *goja.Runtime) (goja.Value, error)) (any, error) {
	type result struct {
		value goja.Value
		err   error
	}
	ch := make(chan result, 1)
	ok := e.loop.RunOnLoop(func(runtime *goja.Runtime) {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					ch <- result{err: err}
					return
				}
				ch <- result{err: errors.New("panic")}
			}
		}()
		value, err := fn(runtime)
		ch <- result{value: value, err: err}
	})
	if !ok {
		return nil, errors.New("engine loop terminated")
	}
	res := <-ch
	if res.err != nil {
		return nil, res.err
	}
	return resolveResult(res.value)
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
		if err := stream.Enable(runtime, cfg.StreamConfig); err != nil {
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

// if the value is Promise, it will be resolved and return the result.
func resolveResult(value goja.Value) (any, error) {
	export := value.Export()
	switch export.(type) {
	case *goja.Promise:
		p := export.(*goja.Promise)
		for p.State() == goja.PromiseStatePending {
			time.Sleep(time.Millisecond * 10)
		}
		switch p.State() {
		case goja.PromiseStatePending:
			return nil, nil
		case goja.PromiseStateFulfilled:
			return p.Result().Export(), nil
		case goja.PromiseStateRejected:
			if err, ok := p.Result().Export().(error); ok {
				return nil, err
			} else {
				stack := p.Result().String()
				result := p.Result()
				if ro, ok := result.(*goja.Object); ok {
					stackVal := ro.Get("stack")
					if stackVal != nil && stackVal.String() != "" {
						stack = stackVal.String()
					}
				}
				return nil, errors.New(stack)
			}
		}
	}
	return export, nil
}
