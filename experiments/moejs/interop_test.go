package evaluation

import (
	"fmt"
	"testing"

	"github.com/MoeForge/moejs"
	"github.com/dop251/goja"
)

type request struct {
	URL string `json:"url"`
}
type context struct {
	Req *request       `json:"req"`
	Res map[string]any `json:"res"`
}

func (r *request) SetURL(s string) { r.URL = s }

// These are migration probes, asserting the observed difference, not claiming
// the existing Gopeed Go bridge has been ported.
func TestHostInterop(t *testing.T) {
	for _, name := range []string{"goja", "moejs"} {
		t.Run(name, func(t *testing.T) {
			c := &context{Req: &request{URL: "before"}}
			code := `ctx.req.url = "after"; ctx.res = {files:[{name:"video.mp4"}]}; typeof ctx.req.setURL`
			var v any
			var e error
			if name == "goja" {
				rt := goja.New()
				rt.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
				if e = rt.Set("ctx", c); e != nil {
					t.Fatal(e)
				}
				value, err := rt.RunString(code)
				e = err
				if e == nil {
					v = value.Export()
				}
				if c.Req.URL != "after" || c.Res == nil {
					t.Fatal("goja write through failed")
				}
			} else {
				rt := moejs.NewRuntime(moejs.Options{MutableIntrinsics: true})
				if e = rt.SetGlobal("ctx", c); e != nil {
					t.Fatal(e)
				}
				s, err := moejs.CompileScript("interop", code)
				if err != nil {
					t.Fatal(err)
				}
				value, err := rt.RunScript(s)
				e = err
				if e == nil {
					v, e = rt.ToGo(value)
				}
				if c.Req.URL != "before" || c.Res != nil {
					t.Fatal("expected snapshot semantics")
				}
			}
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("Go URL=%s, Go res=%v, JS method=%v", c.Req.URL, c.Res, v)
		})
	}
}
func TestFrozenCompatibility(t *testing.T) {
	r, e := New("moejs-frozen")
	if e != nil {
		t.Logf("shared frozen builtins reject host polyfill: %v", e)
		return
	}
	defer r.Close()
	b, e := r.Eval(`Object.defineProperty(Array.prototype,"__probe",{value:1}); true`)
	if e == nil {
		t.Fatalf("expected frozen builtins to reject mutation, got %v", b)
	}
	t.Log(e)
}
func BenchmarkBareRuntime(b *testing.B) {
	for _, name := range []string{"goja", "moejs-mutable", "moejs-shared"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if name == "goja" {
					r := goja.New()
					v, e := r.RunString("1+1")
					if e != nil || v.ToInteger() != 2 {
						b.Fatal(e)
					}
				} else {
					r := moejs.NewRuntime(moejs.Options{MutableIntrinsics: name == "moejs-mutable"})
					s, e := moejs.CompileScript("small", "1+1")
					if e != nil {
						b.Fatal(e)
					}
					v, e := r.RunScript(s)
					if e != nil || v.String() != "2" {
						b.Fatal(fmt.Sprint(v), e)
					}
				}
			}
		})
	}
}
