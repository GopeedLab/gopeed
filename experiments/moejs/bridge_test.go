package evaluation

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/MoeForge/moejs"
)

// Demonstrate an explicit copy-in/copy-out resolve bridge, since FromGo does
// not provide goja's live Go-struct wrappers. Existing task wrappers still need
// native methods and lifecycle ownership before this can ship.
func TestMoeResolveWriteback(t *testing.T) {
	r := setup(t, "moejs", "bilibili").(*Moe)
	req := &base.Request{URL: "https://www.bilibili.com/video/BV1BJ4m1e7g8?p=2"}
	done := make(chan error, 1)
	if !r.loop.RunOnLoop(func(rt *moejs.Runtime) { done <- rt.SetGlobal("hostContext", map[string]any{"req": req}) }) {
		t.Fatal("loop stopped")
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	v, e := r.Eval(`(async()=>{await handlers.onResolve(hostContext);
 // Goja coerces numeric JS label values into map[string]string. JSON does not.
 for (const f of hostContext.res.files) {
  if (f.req.labels) f.req.labels = Object.fromEntries(Object.entries(f.req.labels).map(([k,v])=>[k,String(v)]));
 }
 return JSON.stringify(hostContext.res)})()`)
	if e != nil {
		t.Fatal(e)
	}
	var res base.Resource
	if e = json.Unmarshal([]byte(fmt.Sprint(v)), &res); e != nil {
		t.Fatal(e)
	}
	if e = res.Validate(); e != nil {
		t.Fatal(e)
	}
	if len(res.Files) != 1 || res.Files[0].Name != "BV1BJ4m1e7g8.真实扩展 Replay 测试(P2).1080p.mp4" {
		t.Fatalf("unexpected resource: %+v", res)
	}
	t.Logf("Go Resource.Validate passed, name=%s", res.Files[0].Name)
}

func TestBufferedReplay(t *testing.T) {
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		t.Run(name, func(t *testing.T) {
			var want any
			for _, engine := range []string{"goja", "moejs"} {
				t.Run(engine, func(t *testing.T) {
					r := setup(t, engine, name)
					v, e := r.Eval("Response = BufferedResponse; resolveOnce()")
					if e != nil {
						t.Fatal(e)
					}
					if engine == "goja" {
						want = v
					} else if v != want {
						t.Fatalf("output differs: %v / %v", want, v)
					}
				})
			}
		})
	}
}
