package download

import (
	"fmt"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"github.com/dop251/goja"
)

func injectGopeed(vm *goja.Runtime, gopeed *Instance, post func(func(*goja.Runtime)) bool) error {
	gopeedObject := vm.NewObject()
	if gopeed == nil {
		return vm.Set("gopeed", gopeedObject)
	}
	if err := gopeedObject.Set("events", newJSEventsRuntime(vm, gopeed.Events)); err != nil {
		return err
	}
	if err := gopeedObject.Set("host", newInstanceHost()); err != nil {
		return err
	}
	if err := gopeedObject.Set("info", gopeed.Info); err != nil {
		return err
	}
	if err := gopeedObject.Set("logger", gopeed.Logger); err != nil {
		return err
	}
	if err := gopeedObject.Set("settings", gopeed.Settings); err != nil {
		return err
	}
	if err := gopeedObject.Set("storage", gopeed.Storage); err != nil {
		return err
	}
	runtimeObject := vm.NewObject()
	if gopeed.Runtime != nil {
		if err := runtimeObject.Set("ffmpeg", vm.Get("__gopeed_ffmpeg")); err != nil {
			return err
		}
		if err := runtimeObject.Set("blob", newJSBlobRuntime(vm)); err != nil {
			return err
		}
		if gopeed.Runtime.WebView != nil {
			if err := runtimeObject.Set("webview", newJSWebViewRuntime(vm, gopeed.Runtime.WebView, post)); err != nil {
				return err
			}
		}
	}
	if err := gopeedObject.Set("runtime", runtimeObject); err != nil {
		return err
	}
	return vm.Set("gopeed", gopeedObject)
}

func newJSBlobRuntime(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("createObjectURL", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(vm.Get("__gopeed_blob_create_object_url"))
		if !ok {
			panic(vm.ToValue(fmt.Errorf("blob runtime is not available")))
		}
		value, err := fn(goja.Undefined(), call.Argument(0), call.Argument(1))
		if err != nil {
			panic(vm.ToValue(err))
		}
		return value
	})
	_ = obj.Set("revokeObjectURL", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(vm.Get("__gopeed_blob_revoke_object_url"))
		if !ok {
			panic(vm.ToValue(fmt.Errorf("blob runtime is not available")))
		}
		if _, err := fn(goja.Undefined(), call.Argument(0)); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	return obj
}

func newJSEventsRuntime(vm *goja.Runtime, events InstanceEvents) *goja.Object {
	obj := vm.NewObject()
	register := func(event ActivationEvent, call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.ToValue(fmt.Errorf("missing handler")))
		}
		fnValue := call.Argument(0)
		exported, ok := fnValue.Export().(func(goja.FunctionCall) goja.Value)
		if !ok {
			panic(vm.ToValue(fmt.Errorf("handler must be a function")))
		}
		events.register(event, engine.JSFunction(exported))
		return goja.Undefined()
	}
	_ = obj.Set("onResolve", func(call goja.FunctionCall) goja.Value {
		return register(EventOnResolve, call)
	})
	_ = obj.Set("onStart", func(call goja.FunctionCall) goja.Value {
		return register(EventOnStart, call)
	})
	_ = obj.Set("onError", func(call goja.FunctionCall) goja.Value {
		return register(EventOnError, call)
	})
	_ = obj.Set("onDone", func(call goja.FunctionCall) goja.Value {
		return register(EventOnDone, call)
	})
	return obj
}

func newJSWebViewRuntime(vm *goja.Runtime, runtime *enginewebview.Runtime, post func(func(*goja.Runtime)) bool) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("isAvailable", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(runtime.IsAvailable())
	})
	_ = obj.Set("open", func(call goja.FunctionCall) goja.Value {
		page, err := runtime.Open(optionalMap(call.Argument(0)))
		if err != nil {
			panic(vm.ToValue(err))
		}
		return newJSWebViewPage(vm, page, post)
	})
	return obj
}

func newJSWebViewPage(vm *goja.Runtime, page *enginewebview.PageHandle, post func(func(*goja.Runtime)) bool) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("on", func(call goja.FunctionCall) goja.Value {
		promise, resolve, reject := vm.NewPromise()
		handler, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			_ = reject(vm.NewTypeError("handler must be a function"))
			return vm.ToValue(promise)
		}
		unsubscribe, err := page.On(call.Argument(0).String(), func(event enginewebview.Event) {
			post(func(vm *goja.Runtime) {
				// A callback is a notification: never await its returned promise.
				_, _ = handler(goja.Undefined(), vm.ToValue(event.Data))
			})
		})
		if err != nil {
			_ = reject(vm.NewGoError(err))
		} else {
			_ = resolve(func(goja.FunctionCall) goja.Value { unsubscribe(); return goja.Undefined() })
		}
		return vm.ToValue(promise)
	})

	_ = obj.Set("addInitScript", func(call goja.FunctionCall) goja.Value {
		script, err := requireStringArg(call, 0, "script")
		if err != nil {
			panic(vm.ToValue(err))
		}
		return webviewPromise(vm, post, func() (any, error) { return nil, page.AddInitScript(script) })
	})
	_ = obj.Set("goto", func(call goja.FunctionCall) goja.Value {
		url, err := requireStringArg(call, 0, "url")
		if err != nil {
			panic(vm.ToValue(err))
		}
		opts := optionalMap(call.Argument(1))
		return webviewPromise(vm, post, func() (any, error) { return nil, page.Goto(url, opts) })
	})
	_ = obj.Set("execute", func(call goja.FunctionCall) goja.Value {
		expression, err := enginewebview.NormalizeExecutableValue(call.Argument(0))
		if err != nil {
			panic(vm.ToValue(err))
		}
		result, err := page.Execute(expression, exportArgs(call.Arguments[1:])...)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(result)
	})
	_ = obj.Set("focus", func(call goja.FunctionCall) goja.Value {
		selector, err := requireStringArg(call, 0, "selector")
		if err != nil {
			panic(vm.ToValue(err))
		}
		if err := page.Focus(selector); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("click", func(call goja.FunctionCall) goja.Value {
		selector, err := requireStringArg(call, 0, "selector")
		if err != nil {
			panic(vm.ToValue(err))
		}
		if err := page.Click(selector, optionalMap(call.Argument(1))); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("type", func(call goja.FunctionCall) goja.Value {
		selector, err := requireStringArg(call, 0, "selector")
		if err != nil {
			panic(vm.ToValue(err))
		}
		text, err := requireStringArg(call, 1, "text")
		if err != nil {
			panic(vm.ToValue(err))
		}
		if err := page.Type(selector, text, optionalMap(call.Argument(2))); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("waitForSelector", func(call goja.FunctionCall) goja.Value {
		selector, err := requireStringArg(call, 0, "selector")
		if err != nil {
			panic(vm.ToValue(err))
		}
		result, err := page.WaitForSelector(
			selector,
			optionalMap(call.Argument(1)),
		)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(result)
	})
	_ = obj.Set("waitForFunction", func(call goja.FunctionCall) goja.Value {
		expression, err := enginewebview.NormalizeExecutableValue(call.Argument(0))
		if err != nil {
			panic(vm.ToValue(err))
		}
		args := exportArgs(call.Arguments[1:])
		result, err := page.WaitForFunction(expression, args...)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(result)
	})
	_ = obj.Set("getCookies", func(goja.FunctionCall) goja.Value {
		result, err := page.GetCookies()
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(result)
	})
	_ = obj.Set("setCookie", func(call goja.FunctionCall) goja.Value {
		if err := page.SetCookie(optionalMap(call.Argument(0))); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("deleteCookie", func(call goja.FunctionCall) goja.Value {
		if err := page.DeleteCookie(optionalMap(call.Argument(0))); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("clearCookies", func(goja.FunctionCall) goja.Value {
		if err := page.ClearCookies(); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("url", func(goja.FunctionCall) goja.Value {
		result, err := page.URL()
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(result)
	})
	_ = obj.Set("content", func(goja.FunctionCall) goja.Value {
		result, err := page.Content()
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(result)
	})
	_ = obj.Set("close", func(goja.FunctionCall) goja.Value {
		return webviewPromise(vm, post, func() (any, error) { return nil, page.Close() })
	})
	return obj
}

func optionalMap(value goja.Value) map[string]any {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	if exported, ok := value.Export().(map[string]any); ok {
		return exported
	}
	return nil
}

func exportArgs(values []goja.Value) []any {
	if len(values) == 0 {
		return nil
	}
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value.Export())
	}
	return args
}

func requireStringArg(call goja.FunctionCall, index int, name string) (string, error) {
	value := call.Argument(index)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "", fmt.Errorf(`missing or invalid "%s"`, name)
	}
	return value.String(), nil
}

// Browser operations run outside Goja, allowing notifications to be delivered
// while navigation is pending. Only the event loop touches JavaScript values.
func webviewPromise(vm *goja.Runtime, post func(func(*goja.Runtime)) bool, work func() (any, error)) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	go func() {
		value, err := work()
		post(func(vm *goja.Runtime) {
			if err != nil {
				_ = reject(vm.NewGoError(err))
			} else if value == nil {
				_ = resolve(goja.Undefined())
			} else {
				_ = resolve(value)
			}
		})
	}()
	return vm.ToValue(promise)
}
