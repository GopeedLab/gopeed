//go:build cgo && webview && windows

package goprovider

import (
	"runtime"
	"testing"

	webview "github.com/GopeedLab/webview_go"
)

// warmUpWebView absorbs WebView2's potentially slow first initialization on
// cold Windows runners before exercising the provider's startup contract.
func warmUpWebView(t *testing.T) {
	t.Helper()
	if !webview.IsAvailable() {
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w := webview.NewHeadless(false)
	defer w.Destroy()
	if err := w.Bind("__gopeedWebViewWarmup", func() {
		w.Terminate()
	}); err != nil {
		t.Fatalf("failed to bind webview warm-up callback: %v", err)
	}
	w.Navigate(`data:text/html,<script>globalThis.__gopeedWebViewWarmup()</script>`)
	w.Run()
}
