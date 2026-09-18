package integrationtest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	wv "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

// RunEventContract exercises the same public semantics on native and RPC hosts.
func RunEventContract(t *testing.T, provider wv.Provider) {
	t.Helper()
	page, _, url := openTestPage(t, provider, wv.OpenOptions{Headless: true, Title: "WebView event contract"})
	events := make(chan wv.Event, 100)
	for _, name := range []string{"url-changed", "load", "load-error", "closed"} {
		if _, err := page.On(name, func(e wv.Event) { events <- e }); err != nil {
			t.Fatal(err)
		}
	}
	next := func(name string) wv.Event {
		t.Helper()
		select {
		case event := <-events:
			if event.Name != name {
				t.Fatalf("got event %s (%v), want %s", event.Name, event.Data, name)
			}
			return event
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for %s", name)
		}
		return wv.Event{}
	}
	quiet := func() {
		t.Helper()
		select {
		case e := <-events:
			t.Fatalf("unexpected event: %+v", e)
		case <-time.After(150 * time.Millisecond):
		}
	}
	quiet()
	if err := page.AddInitScript(`globalThis.__initBeforePage = document.readyState`); err != nil {
		t.Fatal(err)
	}
	if err := page.Goto(url, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatal(err)
	}
	e := next("url-changed")
	if strings.TrimRight(fmt.Sprint(e.Data["url"]), "/") != strings.TrimRight(url, "/") || e.Data["sameDocument"] != false {
		t.Fatal(e)
	}
	next("load")
	if result, err := page.Execute(`() => globalThis.__initBeforePage`); err != nil || result != "loading" {
		t.Fatalf("init script: %v %v", result, err)
	}
	if err := page.AddInitScript(`globalThis.__onlyNext = true`); err != nil {
		t.Fatal(err)
	}
	if result, err := page.Execute(`() => typeof globalThis.__onlyNext`); err != nil || result != "undefined" {
		t.Fatalf("init ran in current page: %v %v", result, err)
	}
	for _, expression := range []string{
		`() => { history.pushState({}, '', '#push'); }`,
		`() => { history.replaceState({}, '', '#replace'); }`,
		`() => { location.hash = 'hash'; }`,
	} {
		if _, err := page.Execute(expression); err != nil {
			t.Fatal(err)
		}
		e := next("url-changed")
		if e.Data["sameDocument"] != true {
			t.Fatal(e)
		}
		quiet()
	}
	if _, err := page.Execute(`() => { location.reload(); }`); err != nil {
		t.Fatal(err)
	}
	next("load") // Same address on refresh is not a URL change.
	if result, err := page.Execute(`() => globalThis.__onlyNext`); err != nil || result != true {
		t.Fatalf("next init: %v %v", result, err)
	}
	if _, err := page.Execute(`() => { const f=document.createElement('iframe'); f.src='/next'; document.body.append(f); }`); err != nil {
		t.Fatal(err)
	}
	quiet()
	// The fixture serves plain HTTP. Attempting TLS against it produces a
	// real navigation failure through both HTTP CONNECT and SOCKS proxies.
	// An unreachable HTTP origin would instead become a valid 502 document
	// when the host's HTTP proxy handles the connection error.
	failedURL := strings.Replace(url, "http://", "https://", 1) + "/failure"
	if err := page.Goto(failedURL, map[string]any{"timeoutMs": 3000}); err == nil {
		t.Fatal("failed navigation succeeded")
	}
	e = next("load-error")
	if e.Data["message"] == "" || !strings.Contains(fmt.Sprint(e.Data["url"]), "/failure") {
		t.Fatal(e)
	}
	quiet() // A handled failure must not load a browser-generated fallback page.
	if err := page.Close(); err != nil {
		t.Fatal(err)
	}
	e = next("closed")
	if e.Data["reason"] != "api" {
		t.Fatal(e)
	}
	quiet()
}
