package download

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	btProtocol "github.com/GopeedLab/gopeed/pkg/protocol/bt"
	httpProtocol "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/dop251/goja"
)

func TestExtensionProtocolMethods(t *testing.T) {
	for _, extra := range []any{nil, map[string]any{"header": map[string]any{"Keep": "yes"}}, &httpProtocol.ReqExtra{Header: map[string]string{"Keep": "yes"}}} {
		vm := goja.New()
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		req := &base.Request{Extra: extra}
		task := &Task{Protocol: "http", Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}}
		if err := vm.Set("task", newOnStartExtensionTask(task)); err != nil {
			t.Fatal(err)
		}
		_, err := vm.RunString(`task.meta.req.setMethod('POST'); task.meta.req.setBody('payload'); task.meta.req.putHeader('Authorization','Bearer token'); task.meta.req.putHeader('Remove','remove'); task.meta.req.delHeader('REMOVE');`)
		if err != nil {
			t.Fatal(err)
		}
		if err := base.ParseReqExtra[httpProtocol.ReqExtra](req); err != nil {
			t.Fatal(err)
		}
		got := req.Extra.(*httpProtocol.ReqExtra)
		if got.Method != "POST" || got.Body != "payload" || got.Header["Authorization"] != "Bearer token" {
			t.Fatalf("mutation not applied: %#v", got)
		}
		if _, ok := got.Header["Remove"]; ok {
			t.Fatal("header deletion failed")
		}
		if extra != nil && got.Header["Keep"] != "yes" {
			t.Fatal("existing header lost")
		}
	}
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	req := &base.Request{Extra: &btProtocol.ReqExtra{Trackers: []string{"old"}}}
	task := &Task{Protocol: "bt", Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}}
	if err := vm.Set("task", newOnStartExtensionTask(task)); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunString(`task.meta.req.setTrackers(['new','another']);`); err != nil {
		t.Fatal(err)
	}
	got := req.Extra.(*btProtocol.ReqExtra).Trackers
	if len(got) != 2 || got[0] != "new" || got[1] != "another" {
		t.Fatalf("trackers = %#v", got)
	}
}

func TestExtensionProtocolMethodsRejectInvalidMutation(t *testing.T) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	original := &httpProtocol.ReqExtra{Method: "POST", Header: map[string]string{"authorization": "old", "Keep": "yes"}}
	req := &base.Request{Extra: original}
	task := newOnStartExtensionTask(&Task{Protocol: "http", Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}})
	if err := vm.Set("task", task); err != nil {
		t.Fatal(err)
	}
	for _, script := range []string{`task.meta.req.setHeaders({Keep:null})`, `task.meta.req.setHeaders({Keep:123})`, `task.meta.req.setHeaders({a:'1',A:'2'})`} {
		if _, err := vm.RunString(script); err == nil {
			t.Fatalf("accepted invalid mutation: %s", script)
		}
		if req.Extra != original || original.Header["Keep"] != "yes" {
			t.Fatal("invalid operation changed original extra")
		}
	}
	if _, err := vm.RunString(`task.meta.req.putHeader('Authorization','new');`); err != nil {
		t.Fatal(err)
	}
	got := req.Extra.(*httpProtocol.ReqExtra)
	if len(got.Header) != 2 || got.Header["Authorization"] != "new" || original.Header["authorization"] != "old" {
		t.Fatalf("header update = %#v", got.Header)
	}
	if _, err := vm.RunString(`task.meta.req.setHeaders({Only:'one'});`); err != nil {
		t.Fatal(err)
	}
	got = req.Extra.(*httpProtocol.ReqExtra)
	if got.Method != "POST" || len(got.Header) != 1 {
		t.Fatalf("setHeaders = %#v", got)
	}
	req.Extra = map[string]any{"header": 123}
	if _, err := vm.RunString(`task.meta.req.setBody('new')`); err == nil {
		t.Fatal("invalid protocol extra accepted")
	}
	if req.Extra.(map[string]any)["header"] != 123 {
		t.Fatal("conversion failure mutated extra")
	}
}

func TestExtensionProtocolMethodsOtherProtocolsAreNoOp(t *testing.T) {
	for _, protocol := range []string{"http", "bt", "hls"} {
		for _, onError := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/onError=%v", protocol, onError), func(t *testing.T) {
				vm := goja.New()
				vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
				// An invalid extra proves that unrelated methods skip conversion entirely.
				original := map[string]any{"unchanged": true, "header": 123}
				req := &base.Request{Extra: original}
				task := &Task{Protocol: protocol, Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}}
				var wrapper any = newOnStartExtensionTask(task)
				if onError {
					wrapper = newOnErrorExtensionTask(nil, task)
				}
				if err := vm.Set("task", wrapper); err != nil {
					t.Fatal(err)
				}
				if protocol != "http" {
					if _, err := vm.RunString(`task.meta.req.setMethod('POST');task.meta.req.setBody('body');task.meta.req.setHeaders({a:null});task.meta.req.putHeader('A','value');task.meta.req.delHeader('A');`); err != nil {
						t.Fatal(err)
					}
				}
				if protocol != "bt" {
					if _, err := vm.RunString(`task.meta.req.setTrackers(['udp://example.com']);`); err != nil {
						t.Fatal(err)
					}
				}
				if !reflect.DeepEqual(req.Extra, original) {
					t.Fatalf("extra changed: %#v", req.Extra)
				}
				original["same-object"] = true
				if req.Extra.(map[string]any)["same-object"] != true {
					t.Fatal("no-op replaced extra")
				}
			})
		}
	}
}

func TestExtensionOnErrorRequestWritesToOriginalTask(t *testing.T) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	req := &base.Request{URL: "https://example.com", Extra: &httpProtocol.ReqExtra{Header: map[string]string{"Keep": "yes"}}}
	task := &Task{Protocol: "http", Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}}
	if err := vm.Set("task", newOnErrorExtensionTask(nil, task)); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunString(`
   if (typeof task.continue !== 'function') throw new Error('missing continue');
   if (typeof task.putHeader !== 'undefined') throw new Error('task still exposes protocol methods');
   if (typeof task.req !== 'undefined') throw new Error('legacy task.req');
   const request = task.meta.req;
   if (request.url !== 'https://example.com' || request.extra.header.Keep !== 'yes') throw new Error('request data missing');
   if (!task.meta.opts) throw new Error('metadata fields missing');
   request.setHeaders({Keep:'new'});
   request.putHeader('Authorization','Bearer new');
   request.setMethod('POST');
   if (typeof task.setUrl !== 'undefined' || typeof task.extra !== 'undefined') throw new Error('legacy task API');
   request.setUrl('https://example.com/new');
   request.setLabels({keep:'yes',remove:'yes'});
   request.putLabel('new','value');
   request.delLabel('remove');
   if (request.url !== 'https://example.com/new' || request.labels.new !== 'value' || request.extra.method !== 'POST' || request.extra.header.Authorization !== 'Bearer new') throw new Error('request data did not refresh');
 `); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(req.Labels, map[string]string{"keep": "yes", "new": "value"}) {
		t.Fatalf("labels = %#v", req.Labels)
	}
	got := req.Extra.(*httpProtocol.ReqExtra)
	if got.Method != "POST" || got.Header["Keep"] != "new" || got.Header["Authorization"] != "Bearer new" || req.URL != "https://example.com/new" {
		t.Fatalf("original request not updated: %#v, %#v", req, got)
	}
}

func TestExtensionBTRequestInitializesNilExtra(t *testing.T) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	req := &base.Request{}
	task := &Task{Protocol: "bt", Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}}
	if err := vm.Set("task", newOnStartExtensionTask(task)); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunString(`task.meta.req.setTrackers(['udp://example.com']);`); err != nil {
		t.Fatal(err)
	}
	got, ok := req.Extra.(*btProtocol.ReqExtra)
	if !ok || !reflect.DeepEqual(got.Trackers, []string{"udp://example.com"}) {
		t.Fatalf("extra = %#v", req.Extra)
	}
}
