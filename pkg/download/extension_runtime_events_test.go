package download

import (
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download/engine"
	wv "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"runtime"
	"testing"
	"time"
)

type eventTestPage struct {
	fakeRuntimeWebViewPage
	wv.EventHub
}

func (p *eventTestPage) Close() error {
	p.Emit(wv.Event{Name: "closed", Data: map[string]any{"reason": "api"}})
	return nil
}

type eventTestOpener struct{ page wv.Page }

func (o eventTestOpener) Open(wv.OpenOptions) (wv.Page, error) { return o.page, nil }

func TestExtensionInfoHostMetadata(t *testing.T) {
	e := engine.NewEngine(nil)
	defer e.Close()
	info := NewExtensionInfo(&Extension{Name: "sample", Author: "test", Version: "1.2.3"})
	if err := injectGopeed(e.Runtime, &Instance{Info: info}, e.Post); err != nil {
		t.Fatal(err)
	}
	result, err := e.RunString(`gopeed.info.hostVersion + "/" + gopeed.info.version + "/" + gopeed.info.os + "/" + gopeed.info.arch`)
	if err != nil || result != base.Version+"/1.2.3/"+runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("%v %v", result, err)
	}
}

func TestExtensionWebViewEvents(t *testing.T) {
	e := engine.NewEngine(nil)
	defer e.Close()
	page := &eventTestPage{}
	runtime := wv.NewRuntime(eventTestOpener{page}, true)
	defer runtime.Close()
	if err := injectGopeed(e.Runtime, &Instance{Runtime: &InstanceRuntime{WebView: runtime}}, e.Post); err != nil {
		t.Fatal(err)
	}
	_, err := e.RunString(`var page = gopeed.runtime.webview.open(); var seen = []; var off;
 (async () => {
  const registered = page.on("load", () => { seen.push("load"); return new Promise(() => {}); });
  if (!(registered instanceof Promise)) throw new Error("on must return a Promise");
  off = await registered;
  await page.on("closed", data => seen.push(data.reason));
  for (const args of [["invalid", () => {}], ["load", 1]]) {
   let rejected = false;
   try { await page.on(...args); } catch (_) { rejected = true; }
   if (!rejected) throw new Error("invalid listener accepted");
  }
 })()`)
	if err != nil {
		t.Fatal(err)
	}
	page.Emit(wv.Event{Name: "load", Data: map[string]any{"url": "https://example.test"}})
	result, err := e.RunString(`off(); off(); seen.join(",")`)
	if err != nil || result != "load" {
		t.Fatalf("%v %v", result, err)
	}
	page.Emit(wv.Event{Name: "load"})
	if _, err = e.RunString(`page.close()`); err != nil {
		t.Fatal(err)
	}
	result, err = e.RunString(`seen.join(",")`)
	if err != nil || result != "load,api" {
		t.Fatalf("%v %v", result, err)
	}
}

// Navigation must not occupy the Goja loop: a notification can execute page
// operations even while the original goto promise is still pending.
type navigationEventTestPage struct {
	eventTestPage
	release chan struct{}
}

func (p *navigationEventTestPage) Goto(url string, _ wv.GotoOptions) error {
	p.Emit(wv.Event{Name: "load", Data: map[string]any{"url": url}})
	select {
	case <-p.release:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("event callback blocked by navigation")
	}
}
func (p *navigationEventTestPage) Execute(string, ...any) (any, error) {
	close(p.release)
	return nil, nil
}
func TestExtensionWebViewEventsDuringNavigation(t *testing.T) {
	e := engine.NewEngine(nil)
	defer e.Close()
	page := &navigationEventTestPage{release: make(chan struct{})}
	runtime := wv.NewRuntime(eventTestOpener{page}, true)
	defer runtime.Close()
	if err := injectGopeed(e.Runtime, &Instance{Runtime: &InstanceRuntime{WebView: runtime}}, e.Post); err != nil {
		t.Fatal(err)
	}
	result, err := e.RunString(`(async () => {
  const page = await gopeed.runtime.webview.open();
  await page.on("load", () => page.execute("release"));
  if (await page.addInitScript("globalThis.ready = true") !== undefined) throw new Error("init result must be void");
  if (await page.goto("https://example.test") !== undefined) throw new Error("goto result must be void");
  return true;
 })()`)
	if err != nil || result != true {
		t.Fatalf("%v %v", result, err)
	}
}
