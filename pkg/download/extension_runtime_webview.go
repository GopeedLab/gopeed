package download

import (
	"fmt"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"github.com/dop251/goja"
)

func injectGopeed(vm *goja.Runtime, gopeed *Instance) error {
	gopeedObject := vm.NewObject()
	if gopeed == nil {
		return vm.Set("gopeed", gopeedObject)
	}
	if err := gopeedObject.Set("events", newJSEventsRuntime(vm, gopeed.Events)); err != nil {
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
	if gopeed.Runtime != nil && gopeed.Runtime.WebView != nil {
		if err := runtimeObject.Set("webview", newJSWebViewRuntime(vm, gopeed.Runtime.WebView)); err != nil {
			return err
		}
	}
	if err := gopeedObject.Set("runtime", runtimeObject); err != nil {
		return err
	}
	return vm.Set("gopeed", gopeedObject)
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

func newJSWebViewRuntime(vm *goja.Runtime, runtime *enginewebview.Runtime) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("isAvailable", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(runtime.IsAvailable())
	})
	_ = obj.Set("open", func(call goja.FunctionCall) goja.Value {
		page, err := runtime.Open(optionalMap(call.Argument(0)))
		if err != nil {
			panic(vm.ToValue(err))
		}
		return newJSWebViewPage(vm, page)
	})
	return obj
}

func newJSWebViewPage(vm *goja.Runtime, page *enginewebview.PageHandle) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("addInitScript", func(call goja.FunctionCall) goja.Value {
		script, err := requireStringArg(call, 0, "script")
		if err != nil {
			panic(vm.ToValue(err))
		}
		if err := page.AddInitScript(script); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("goto", func(call goja.FunctionCall) goja.Value {
		url, err := requireStringArg(call, 0, "url")
		if err != nil {
			panic(vm.ToValue(err))
		}
		if err := page.Goto(
			url,
			optionalMap(call.Argument(1)),
		); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
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
		if err := page.Close(); err != nil {
			panic(vm.ToValue(err))
		}
		return goja.Undefined()
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
