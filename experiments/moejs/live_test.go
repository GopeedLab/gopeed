package evaluation

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// Opt in: public endpoints are volatile and are not used in benchmark timings.
// Only URL resolution is attempted; blob registration and UA WebView stay mocked.
func TestLiveResolve(t *testing.T) {
	if os.Getenv("GOPEED_MOEJS_LIVE") != "1" {
		t.Skip("set GOPEED_MOEJS_LIVE=1 to contact public endpoints")
	}
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		var want any
		for _, engine := range []string{"goja", "moejs"} {
			t.Run(name+"/"+engine, func(t *testing.T) {
				r, e := newRuntime(engine, true)
				if e != nil {
					t.Fatal(e)
				}
				defer r.Close()
				p, e := os.ReadFile("replay.js")
				if e != nil {
					t.Fatal(e)
				}
				for _, s := range []string{fmt.Sprintf("var extensionName=%q;", name), string(p), "liveNetwork = true;", source(t, name)} {
					if _, e = r.Eval(s); e != nil {
						t.Fatal(e)
					}
				}
				started := time.Now()
				v, e := r.Eval("resolveOnce()")
				t.Logf("elapsed=%s", time.Since(started))
				if e != nil {
					t.Fatal(e)
				}
				if engine == "goja" {
					want = v
				} else if want != nil && v != want {
					t.Fatalf("live resolve outputs differ: %v / %v", want, v)
				}
				t.Log(v)
			})
		}
	}
}
