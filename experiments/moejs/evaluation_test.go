package evaluation

import (
	"fmt"
	"os"
	"testing"
)

func source(t testing.TB, name string) string {
	t.Helper()
	b, e := os.ReadFile("testdata/extensions/" + name + ".js")
	if e != nil {
		t.Fatal("run python3 fetch_sources.py first: ", e)
	}
	return string(b)
}
func setup(t testing.TB, engine, name string) Runtime {
	t.Helper()
	r, e := New(engine)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(r.Close)
	b, e := os.ReadFile("replay.js")
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{fmt.Sprintf("var extensionName = %q;", name), string(b), source(t, name)} {
		if _, e = r.Eval(s); e != nil {
			t.Fatal(e)
		}
	}
	return r
}
func TestExtensionReplay(t *testing.T) {
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		t.Run(name, func(t *testing.T) {
			var want any
			for _, engine := range []string{"goja", "moejs"} {
				t.Run(engine, func(t *testing.T) {
					r := setup(t, engine, name)
					v, e := r.Eval("resolveOnce()")
					if e != nil {
						t.Fatal(e)
					}
					t.Log(v)
					if engine == "goja" {
						want = v
					} else if want != nil && v != want {
						t.Fatalf("output differs\ngoja: %s\nmoejs: %s", want, v)
					}
				})
			}
		})
	}
}
func BenchmarkExtension(b *testing.B) {
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		for _, engine := range []string{"goja", "moejs"} {
			b.Run(name+"/"+engine+"/warm", func(b *testing.B) {
				r := setup(b, engine, name)
				if _, e := r.Eval("resolveOnce()"); e != nil {
					b.Fatal(e)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, e := r.Eval("resolveOnce()"); e != nil {
						b.Fatal(e)
					}
				}
			})
			b.Run(name+"/"+engine+"/cold", func(b *testing.B) {
				src := source(b, name)
				p, e := os.ReadFile("replay.js")
				if e != nil {
					b.Fatal(e)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r, e := New(engine)
					if e != nil {
						b.Fatal(e)
					}
					for _, s := range []string{fmt.Sprintf("var extensionName=%q;", name), string(p), src, "resolveOnce()"} {
						if _, e = r.Eval(s); e != nil {
							r.Close()
							b.Fatal(e)
						}
					}
					r.Close()
				}
			})
		}
	}
}

func TestExtensionLifecycle(t *testing.T) {
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		t.Run(name, func(t *testing.T) {
			var want any
			for _, engine := range []string{"goja", "moejs"} {
				t.Run(engine, func(t *testing.T) {
					r := setup(t, engine, name)
					v, e := r.Eval("lifecycleOnce()")
					if e != nil {
						t.Fatal(e)
					}
					t.Log(v)
					if engine == "goja" {
						want = v
					} else if want != nil && v != want {
						t.Fatalf("lifecycle differs\ngoja: %s\nmoejs: %s", want, v)
					}
				})
			}
		})
	}
}
func BenchmarkBufferedTransport(b *testing.B) {
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		for _, engine := range []string{"goja", "moejs"} {
			b.Run(name+"/"+engine, func(b *testing.B) {
				r := setup(b, engine, name)
				if _, e := r.Eval("Response = BufferedResponse; resolveOnce()"); e != nil {
					b.Fatal(e)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, e := r.Eval("resolveOnce()"); e != nil {
						b.Fatal(e)
					}
				}
			})
		}
	}
}
