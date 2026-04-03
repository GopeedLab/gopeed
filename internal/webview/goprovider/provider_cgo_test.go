//go:build cgo

package goprovider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func openTestPage(t *testing.T, opts ...enginewebview.OpenOptions) (enginewebview.Page, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Test Page</title></head><body><h1 id="heading">Hello Webview</h1></body></html>`)
	}))
	t.Cleanup(func() { server.Close() })

	var opt enginewebview.OpenOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Width == 0 {
		opt.Width = 640
	}
	if opt.Height == 0 {
		opt.Height = 480
	}

	provider := New()
	page, err := provider.Open(opt)
	if err != nil {
		t.Fatalf("failed to open webview: %v", err)
	}
	t.Cleanup(func() { page.Close() })

	return page, server
}

func TestWebviewNavigateAndGetTitle(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.Navigate(server.URL, enginewebview.NavigateOptions{TimeoutMS: 10000}); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	title, err := page.Execute(`() => document.title`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if title != "Test Page" {
		t.Fatalf("expected title 'Test Page', got %v", title)
	}
}

func TestWebviewExecuteWithArgs(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.Navigate(server.URL, enginewebview.NavigateOptions{TimeoutMS: 10000}); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	result, err := page.Execute(`(selector) => document.querySelector(selector)?.textContent`, "#heading")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result != "Hello Webview" {
		t.Fatalf("expected 'Hello Webview', got %v", result)
	}
}

func TestWebviewExecuteAsyncFunction(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.Navigate(server.URL, enginewebview.NavigateOptions{TimeoutMS: 10000}); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	result, err := page.Execute(`async () => {
		await new Promise(r => setTimeout(r, 50));
		return document.title;
	}`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result != "Test Page" {
		t.Fatalf("expected 'Test Page', got %v", result)
	}
}

func TestWebviewExecuteWithError(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.Navigate(server.URL, enginewebview.NavigateOptions{TimeoutMS: 10000}); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	_, err := page.Execute(`() => { throw new Error("intentional"); }`)
	if err == nil {
		t.Fatal("expected error from JS throw")
	}
}

func TestWebviewAddInitScript(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.AddInitScript(`window.__initValue = 42;`); err != nil {
		t.Fatalf("addInitScript failed: %v", err)
	}

	if err := page.Navigate(server.URL, enginewebview.NavigateOptions{TimeoutMS: 10000}); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	value, err := page.Execute(`() => window.__initValue`)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if value != float64(42) {
		t.Fatalf("expected 42, got %v", value)
	}
}
