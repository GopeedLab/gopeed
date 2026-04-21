package webview

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestRuntimeOpenAndExecuteFunctionSource(t *testing.T) {
	opener := &fakeOpener{
		page: &fakePage{executeValue: "ok"},
	}
	runtime := NewRuntime(opener, true)
	page, err := runtime.Open(map[string]any{
		"headless":  true,
		"debug":     true,
		"title":     "Gopeed",
		"width":     1280,
		"height":    720,
		"userAgent": "UA",
	})
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	fnValue, err := vm.RunString(`(a, b) => a + b`)
	if err != nil {
		t.Fatal(err)
	}
	value, err := page.Execute(fnValue, int64(1), int64(2))
	if err != nil {
		t.Fatal(err)
	}
	if value != "ok" {
		t.Fatalf("unexpected execute result: %#v", value)
	}
	if opener.opts.Title != "Gopeed" || opener.opts.UserAgent != "UA" {
		t.Fatalf("unexpected open options: %+v", opener.opts)
	}
	if opener.page.lastExecuteSource != "((a, b) => a + b)" {
		t.Fatalf("unexpected function source: %q", opener.page.lastExecuteSource)
	}
	if len(opener.page.lastExecuteArgs) != 2 || opener.page.lastExecuteArgs[0] != int64(1) || opener.page.lastExecuteArgs[1] != int64(2) {
		t.Fatalf("unexpected execute args: %#v", opener.page.lastExecuteArgs)
	}
}

func TestRuntimeExecuteRepairsObjectLiteralArrowFunction(t *testing.T) {
	opener := &fakeOpener{
		page: &fakePage{executeValue: map[string]any{"ok": true}},
	}
	runtime := NewRuntime(opener, true)
	page, err := runtime.Open()
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	fnValue, err := vm.RunString(`() => ({ ok: true })`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := page.Execute(fnValue); err != nil {
		t.Fatal(err)
	}
	if opener.page.lastExecuteSource != "(() => ({ ok: true }))" {
		t.Fatalf("expected repaired function source, got %q", opener.page.lastExecuteSource)
	}
}

func TestRuntimeExecuteRepairsMultilineObjectLiteralArrowFunction(t *testing.T) {
	opener := &fakeOpener{
		page: &fakePage{executeValue: map[string]any{"ok": true}},
	}
	runtime := NewRuntime(opener, true)
	page, err := runtime.Open()
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	fnValue, err := vm.RunString(`() => ({
		title: document.title || "",
		url: String(location.href || ""),
		userAgent: navigator.userAgent,
		readyState: document.readyState,
	})`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := page.Execute(fnValue); err != nil {
		t.Fatal(err)
	}
	expected := `(() => ({
		title: document.title || "",
		url: String(location.href || ""),
		userAgent: navigator.userAgent,
		readyState: document.readyState,
	}))`
	if opener.page.lastExecuteSource != expected {
		t.Fatalf("expected repaired multiline function source, got %q", opener.page.lastExecuteSource)
	}
}

func TestRuntimeHelpers(t *testing.T) {
	opener := &fakeOpener{
		page: &fakePage{
			executeQueue: []any{
				false,
				true,
				false,
				true,
				map[string]any{"matched": false, "value": nil},
				map[string]any{"matched": true, "value": "ready"},
				"https://example.com",
				"<html></html>",
			},
			cookies: []Cookie{
				{Name: "a", Value: "1", Domain: "example.com", Path: "/"},
				{Name: "b", Value: "two", Domain: "example.com", Path: "/"},
			},
		},
	}
	runtime := NewRuntime(opener, true)
	page, err := runtime.Open()
	if err != nil {
		t.Fatal(err)
	}
	if err := page.AddInitScript("window.__TEST__ = true"); err != nil {
		t.Fatal(err)
	}
	if opener.page.lastInitScript != "window.__TEST__ = true" {
		t.Fatalf("unexpected init script: %q", opener.page.lastInitScript)
	}
	if err := page.Goto("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if opener.page.lastGotoURL != "https://example.com" {
		t.Fatalf("unexpected goto url: %q", opener.page.lastGotoURL)
	}
	if err := page.WaitForLoad(map[string]any{"timeoutMs": 50, "pollIntervalMs": 1}); err != nil {
		t.Fatal(err)
	}
	found, err := page.WaitForSelector("#app", map[string]any{"timeoutMs": 50, "pollIntervalMs": 1})
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected selector to be found")
	}
	value, err := page.WaitForFunction(`() => window.__READY__`, map[string]any{"timeoutMs": 50, "pollIntervalMs": 1})
	if err != nil {
		t.Fatal(err)
	}
	if value != "ready" {
		t.Fatalf("unexpected waitForFunction value: %#v", value)
	}
	cookies, err := page.GetCookies()
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 2 || cookies[0].Name != "a" || cookies[0].Value != "1" || cookies[1].Name != "b" || cookies[1].Value != "two" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
	if err := page.SetCookie(map[string]any{
		"name":     "session",
		"value":    "abc",
		"domain":   "example.com",
		"path":     "/",
		"secure":   true,
		"httpOnly": true,
	}); err != nil {
		t.Fatal(err)
	}
	if opener.page.lastSetCookie.Name != "session" || !opener.page.lastSetCookie.Secure || !opener.page.lastSetCookie.HTTPOnly {
		t.Fatalf("unexpected set cookie: %#v", opener.page.lastSetCookie)
	}
	if err := page.DeleteCookie(map[string]any{
		"name":   "session",
		"domain": "example.com",
		"path":   "/",
	}); err != nil {
		t.Fatal(err)
	}
	if opener.page.lastDeleteCookie.Name != "session" || opener.page.lastDeleteCookie.Domain != "example.com" {
		t.Fatalf("unexpected delete cookie: %#v", opener.page.lastDeleteCookie)
	}
	if err := page.ClearCookies(); err != nil {
		t.Fatal(err)
	}
	if !opener.page.cookiesCleared {
		t.Fatal("expected cookies to be cleared")
	}
	url, err := page.URL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://example.com" {
		t.Fatalf("unexpected url: %q", url)
	}
	content, err := page.Content()
	if err != nil {
		t.Fatal(err)
	}
	if content != "<html></html>" {
		t.Fatalf("unexpected content: %q", content)
	}
	if err := page.Close(); err != nil {
		t.Fatal(err)
	}
	if !opener.page.closed {
		t.Fatal("expected page to be closed")
	}
}

func TestRuntimeInteractionHelpers(t *testing.T) {
	opener := &fakeOpener{
		page: &fakePage{executeValue: true},
	}
	runtime := NewRuntime(opener, true)
	page, err := runtime.Open()
	if err != nil {
		t.Fatal(err)
	}

	if err := page.Focus("#search"); err != nil {
		t.Fatal(err)
	}
	if len(opener.page.lastExecuteArgs) != 1 || opener.page.lastExecuteArgs[0] != "#search" {
		t.Fatalf("unexpected focus args: %#v", opener.page.lastExecuteArgs)
	}
	if !strings.Contains(opener.page.lastExecuteSource, "element.focus()") {
		t.Fatalf("unexpected focus source: %q", opener.page.lastExecuteSource)
	}

	if err := page.Click("#submit", map[string]any{"delay": 12}); err != nil {
		t.Fatal(err)
	}
	if len(opener.page.lastExecuteArgs) != 2 || opener.page.lastExecuteArgs[0] != "#submit" || opener.page.lastExecuteArgs[1] != int64(12) {
		t.Fatalf("unexpected click args: %#v", opener.page.lastExecuteArgs)
	}
	if !strings.Contains(opener.page.lastExecuteSource, "performClick") {
		t.Fatalf("unexpected click source: %q", opener.page.lastExecuteSource)
	}

	if err := page.Type("#search", "Gopeed", map[string]any{"delay": 5}); err != nil {
		t.Fatal(err)
	}
	if len(opener.page.lastExecuteArgs) != 3 || opener.page.lastExecuteArgs[0] != "#search" || opener.page.lastExecuteArgs[1] != "Gopeed" || opener.page.lastExecuteArgs[2] != int64(5) {
		t.Fatalf("unexpected type args: %#v", opener.page.lastExecuteArgs)
	}
	if !strings.Contains(opener.page.lastExecuteSource, "element is not typable") {
		t.Fatalf("unexpected type source: %q", opener.page.lastExecuteSource)
	}
}

type fakeOpener struct {
	opts OpenOptions
	page *fakePage
}

func (o *fakeOpener) Open(opts OpenOptions) (Page, error) {
	o.opts = opts
	return o.page, nil
}

type fakePage struct {
	lastInitScript    string
	lastGotoURL       string
	lastGotoOpts      GotoOptions
	lastExecuteSource string
	lastExecuteArgs   []any
	executeValue      any
	executeQueue      []any
	cookies           []Cookie
	lastSetCookie     Cookie
	lastDeleteCookie  Cookie
	cookiesCleared    bool
	closed            bool
}

func (p *fakePage) AddInitScript(script string) error {
	p.lastInitScript = script
	return nil
}

func (p *fakePage) Goto(url string, opts GotoOptions) error {
	p.lastGotoURL = url
	p.lastGotoOpts = opts
	return nil
}

func (p *fakePage) Execute(expression string, args ...any) (any, error) {
	p.lastExecuteSource = expression
	p.lastExecuteArgs = args
	if len(p.executeQueue) > 0 {
		value := p.executeQueue[0]
		p.executeQueue = p.executeQueue[1:]
		return value, nil
	}
	return p.executeValue, nil
}

func (p *fakePage) GetCookies() ([]Cookie, error) {
	return append([]Cookie(nil), p.cookies...), nil
}

func (p *fakePage) SetCookie(cookie Cookie) error {
	p.lastSetCookie = cookie
	return nil
}

func (p *fakePage) DeleteCookie(cookie Cookie) error {
	p.lastDeleteCookie = cookie
	return nil
}

func (p *fakePage) ClearCookies() error {
	p.cookiesCleared = true
	return nil
}

func (p *fakePage) Close() error {
	p.closed = true
	return nil
}
