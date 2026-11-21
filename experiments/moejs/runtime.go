// Experimental adapter only; this is not a production engine switch.
package evaluation

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	production "github.com/GopeedLab/gopeed/pkg/download/engine"
	"github.com/MoeForge/moejs"
	"github.com/MoeForge/moejs/eventloop"
	"github.com/MoeForge/moejs/webapi"
)

type Runtime interface {
	Eval(string) (any, error)
	Close()
}

type Goja struct{ *production.Engine }

func (g *Goja) Eval(s string) (any, error) { return g.RunString(s) }

type Moe struct {
	rt   *moejs.Runtime
	loop *eventloop.Loop
	api  *webapi.API
}

func readHostScript(path string) (string, error) {
	b, err := os.ReadFile(filepath.Join("..", "..", "pkg", "download", "engine", path))
	return string(b), err
}

func New(name string) (Runtime, error) { return newRuntime(name, false) }

func newRuntime(name string, network bool) (Runtime, error) {
	if name == "goja" {
		return &Goja{production.NewEngine(nil)}, nil
	}
	if name != "moejs" && name != "moejs-frozen" {
		return nil, fmt.Errorf("unknown engine %s", name)
	}
	// Preserve goja's mutable-builtin contract; shared builtins are a separate probe.
	rt := moejs.NewRuntime(moejs.Options{MutableIntrinsics: name != "moejs-frozen", MaxDynamicSource: -1})
	loop, err := eventloop.New(rt, eventloop.Options{})
	if err != nil {
		return nil, err
	}
	opts := webapi.Options{}
	if network {
		opts.Client = &http.Client{Timeout: 20 * time.Second}
	}
	api, err := webapi.Install(loop, opts)
	if err != nil {
		loop.Terminate()
		return nil, err
	}
	m := &Moe{rt, loop, api}
	if err = loop.Start(); err != nil {
		m.Close()
		return nil, err
	}
	scripts := []string{"globalThis.global = globalThis; globalThis.window = globalThis; globalThis.location = new URL('http://localhost');"}
	for _, path := range []string{"polyfill/out/index.js", "inject/xhr/xhr.js"} {
		s, e := readHostScript(path)
		if e != nil {
			m.Close()
			return nil, e
		}
		scripts = append(scripts, s)
	}
	for _, s := range scripts {
		if _, e := m.Eval(s); e != nil {
			m.Close()
			return nil, e
		}
	}
	return m, nil
}

func (m *Moe) Close() { m.api.Close(); m.loop.Terminate() }
func (m *Moe) Eval(source string) (any, error) {
	// Compile each evaluation, matching production Engine.RunString.
	script, err := moejs.CompileScript("evaluation.js", source)
	if err != nil {
		return nil, err
	}
	type result struct {
		value any
		err   error
	}
	ch := make(chan result, 1)
	ok := m.loop.RunOnLoop(func(rt *moejs.Runtime) {
		v, e := rt.RunScript(script)
		if e != nil {
			ch <- result{err: e}
			return
		}
		if _, _, promise := moejs.PromiseResult(v); promise {
			then, e := rt.Get(v, "then")
			if e != nil {
				ch <- result{err: e}
				return
			}
			fulfilled := rt.Function("fulfilled", 1, func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
				out, e := rt.ToGo(moejs.Arg(args, 0))
				ch <- result{out, e}
				return moejs.Undefined(), nil
			})
			rejected := rt.Function("rejected", 1, func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
				v := moejs.Arg(args, 0)
				stack, e := rt.Get(v, "stack")
				message := v.String()
				if e == nil && !stack.IsUndefined() {
					message = stack.String()
				}
				ch <- result{err: fmt.Errorf("%s", message)}
				return moejs.Undefined(), nil
			})
			_, e = rt.CallFunction(then, v, fulfilled, rejected)
			if e != nil {
				ch <- result{err: e}
			}
			return
		}
		out, e := rt.ToGo(v)
		ch <- result{out, e}
	})
	if !ok {
		return nil, fmt.Errorf("loop terminated")
	}
	select {
	case r := <-ch:
		return r.value, r.err
	case <-time.After(30 * time.Second):
		m.rt.Interrupt("evaluation timeout")
		return nil, fmt.Errorf("evaluation timed out")
	}
}
