//go:build cgo

package goprovider

import (
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	webview "github.com/GopeedLab/webview_go"
	"github.com/dop251/goja"
	"testing"
	"time"
)

func TestInternalErrorDocumentDoesNotReportPageEvents(t *testing.T) {
	vm := goja.New()
	if _, err := vm.RunString(`
  var window = globalThis;
  window.top = window;
  var location = {protocol: "chrome-error:", href: "chrome-error://chromewebdata/"};
  var notifications = [];
  var listeners = [];
  function notify(payload) { notifications.push(payload); }
  window.addEventListener = (name) => listeners.push(name);
 `); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunString(buildLoadNotifyScript("notify")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"notifications", "listeners"} {
		value, err := vm.RunString(name + ".length")
		if err != nil || value.ToInteger() != 0 {
			t.Fatalf("internal error page registered %s: %v (%v)", name, value, err)
		}
	}
	// The provider must wait for the native failure, preserving the failed URL.
	page := newPageWrapper(enginewebview.OpenOptions{})
	got := make(chan string, 1)
	_, err := page.On("load-error", func(e enginewebview.Event) { got <- e.Data["url"].(string) })
	if err != nil {
		t.Fatal(err)
	}
	page.handleNativeEvent(webview.Event{Name: "load-error", URL: "https://example.test/failure", Message: "TLS handshake failed"})
	if err := page.waitForNavigation(time.Second, "load"); err == nil {
		t.Fatal("native failure completed navigation successfully")
	}
	if url := <-got; url != "https://example.test/failure" {
		t.Fatal(url)
	}
}
