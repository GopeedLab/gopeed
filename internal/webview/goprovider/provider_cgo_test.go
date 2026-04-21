//go:build cgo && webview

package goprovider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func openTestPage(t *testing.T, opts ...enginewebview.OpenOptions) (*enginewebview.PageHandle, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/next":
			fmt.Fprint(w, `<!DOCTYPE html>
<html>
  <head><title>Next Page</title></head>
  <body>
    <h1 id="next-page">Navigation complete</h1>
  </body>
</html>`)
		default:
			fmt.Fprint(w, `<!DOCTYPE html>
<html>
  <head><title>Test Page</title></head>
  <body>
    <input id="name" type="text" />
    <div id="name-value"></div>
    <button id="clicker" type="button">Click me</button>
    <div id="click-count">0</div>
    <button id="nav-btn" type="button">Navigate</button>
    <script>
      window.__inputEvents = [];
      window.__readyValue = null;
      const nameInput = document.getElementById('name');
      const nameValue = document.getElementById('name-value');
      nameInput.addEventListener('input', () => {
        nameValue.textContent = nameInput.value;
        window.__inputEvents.push(nameInput.value);
      });
      document.getElementById('clicker').addEventListener('click', () => {
        const counter = document.getElementById('click-count');
        counter.textContent = String(Number(counter.textContent || '0') + 1);
      });
      document.getElementById('nav-btn').addEventListener('click', () => {
        setTimeout(() => {
          window.location.href = '/next';
        }, 50);
      });
      setTimeout(() => {
        const marker = document.createElement('div');
        marker.id = 'delayed';
        marker.textContent = 'Loaded later';
        document.body.appendChild(marker);
      }, 150);
      setTimeout(() => {
        window.__readyValue = { status: 'ready', source: 'timer' };
      }, 200);
    </script>
  </body>
</html>`)
		}
	}))
	t.Cleanup(server.Close)

	openOpts := enginewebview.OpenOptions{
		Width:  640,
		Height: 480,
		Title:  "Gopeed WebView Test",
	}
	if len(opts) > 0 {
		openOpts = opts[0]
	}
	if openOpts.Width == 0 {
		openOpts.Width = 640
	}
	if openOpts.Height == 0 {
		openOpts.Height = 480
	}

	runtime := enginewebview.NewRuntime(New(), true)
	page, err := runtime.Open(map[string]any{
		"headless": openOpts.Headless,
		"debug":    openOpts.Debug,
		"title":    openOpts.Title,
		"width":    openOpts.Width,
		"height":   openOpts.Height,
	})
	if err != nil {
		t.Fatalf("failed to open webview: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Close()
	})
	return page, server
}

func TestWebviewPageHandleInteractionHelpers(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.Goto(server.URL, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatalf("goto failed: %v", err)
	}

	if err := page.Focus("#name"); err != nil {
		t.Fatalf("focus failed: %v", err)
	}
	activeElement, err := page.Execute(`() => document.activeElement && document.activeElement.id`)
	if err != nil {
		t.Fatalf("activeElement execute failed: %v", err)
	}
	if activeElement != "name" {
		t.Fatalf("expected active element to be #name, got %v", activeElement)
	}

	if err := page.Type("#name", "Gopeed", map[string]any{"delay": 5}); err != nil {
		t.Fatalf("type failed: %v", err)
	}
	typeState, err := page.Execute(`() => ({
		value: document.querySelector('#name').value,
		mirror: document.querySelector('#name-value').textContent,
		inputCount: window.__inputEvents.length
	})`)
	if err != nil {
		t.Fatalf("type state execute failed: %v", err)
	}
	stateMap, ok := typeState.(map[string]any)
	if !ok {
		t.Fatalf("unexpected type state payload: %T", typeState)
	}
	if stateMap["value"] != "Gopeed" || stateMap["mirror"] != "Gopeed" {
		t.Fatalf("unexpected type state: %#v", stateMap)
	}
	if stateMap["inputCount"] != float64(len("Gopeed")) {
		t.Fatalf("unexpected input event count: %#v", stateMap["inputCount"])
	}

	if err := page.Click("#clicker"); err != nil {
		t.Fatalf("click failed: %v", err)
	}
	clickCount, err := page.Execute(`() => document.querySelector('#click-count').textContent`)
	if err != nil {
		t.Fatalf("click count execute failed: %v", err)
	}
	if clickCount != "1" {
		t.Fatalf("expected click count to be 1, got %v", clickCount)
	}

	found, err := page.WaitForSelector("#delayed", map[string]any{
		"timeoutMs":      5000,
		"pollIntervalMs": 25,
		"visible":        true,
	})
	if err != nil {
		t.Fatalf("waitForSelector failed: %v", err)
	}
	if !found {
		t.Fatal("expected delayed selector to appear")
	}

	value, err := page.WaitForFunction(`() => window.__readyValue`, map[string]any{
		"timeoutMs":      5000,
		"pollIntervalMs": 25,
	})
	if err != nil {
		t.Fatalf("waitForFunction failed: %v", err)
	}
	readyValue, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected waitForFunction value: %T", value)
	}
	if readyValue["status"] != "ready" {
		t.Fatalf("unexpected waitForFunction payload: %#v", readyValue)
	}
}

func TestWebviewWaitForSelectorTimeout(t *testing.T) {
	page, server := openTestPage(t)

	if err := page.Goto(server.URL, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatalf("goto failed: %v", err)
	}
	found, err := page.WaitForSelector("#missing", map[string]any{
		"timeoutMs":      200,
		"pollIntervalMs": 25,
	})
	if err != nil {
		t.Fatalf("waitForSelector failed: %v", err)
	}
	if found {
		t.Fatal("expected missing selector wait to time out")
	}
}
