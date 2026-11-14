package evaluation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAsyncHTTP(t *testing.T) {
	cancelled := make(chan struct{}, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wait" {
			<-r.Context().Done()
			cancelled <- struct{}{}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"value":42}`)
	}))
	defer server.Close()
	for _, engine := range []string{"goja", "moejs"} {
		t.Run(engine, func(t *testing.T) {
			r, e := newRuntime(engine, true)
			if e != nil {
				t.Fatal(e)
			}
			defer r.Close()
			v, e := r.Eval(fmt.Sprintf(`(async()=>{const r=await fetch(%q);return (await r.json()).value})()`, server.URL))
			if e != nil || fmt.Sprint(v) != "42" {
				t.Fatalf("fetch value=%v, error=%v", v, e)
			}
			v, e = r.Eval(fmt.Sprintf(`(async()=>{const c=new AbortController();const p=fetch(%q,{signal:c.signal});setTimeout(()=>c.abort(new Error('evaluation cancel')),10);try{await p;return false}catch(e){return e.name+': '+e.message}})()`, server.URL+"/wait"))
			want := "evaluation cancel"
			// Existing Gopeed fetch catch path constructs DOMException without a
			// definition in this standalone host. Record this baseline divergence;
			// independently require actual cancellation of the server request below.
			if engine == "goja" {
				want = "ReferenceError: DOMException is not defined"
			}
			if e != nil || !strings.Contains(fmt.Sprint(v), want) {
				t.Fatalf("abort value=%v, error=%v", v, e)
			}
			t.Logf("abort rejection: %v", v)
			select {
			case <-cancelled:
			case <-time.After(2 * time.Second):
				t.Fatal("request context was not cancelled")
			}
			probe, e := r.Eval(`(()=>{try{const c=new AbortController();c.abort();return c.signal.reason?.name}catch(e){return e.name+': '+e.message}})()`)
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("default abort probe: %v", probe)
		})
	}
}
