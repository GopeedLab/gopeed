package integrationtest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

const TestUserAgent = "GopeedWebViewIntegration/1.0"

type ContractOptions struct {
	AvailabilityTimeout time.Duration
	CookieDomainMode    CookieDomainMode
	CookieTestURL       string
}

type CookieDomainMode int

const (
	CookieDomainModeRequired CookieDomainMode = iota
	CookieDomainModeOmit
)

func RunProviderContract(t *testing.T, provider enginewebview.Provider, opts ContractOptions) {
	t.Helper()

	timeout := opts.AvailabilityTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	waitForProviderAvailable(t, provider, timeout)

	page, server, testURL := openTestPage(t, provider, enginewebview.OpenOptions{
		Headless:  true,
		Title:     "Gopeed WebView Integration",
		Width:     640,
		Height:    480,
		UserAgent: TestUserAgent,
	})

	if err := page.Goto(testURL, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatalf("goto failed: %v", err)
	}

	ua, err := page.Execute(`() => navigator.userAgent`)
	if err != nil {
		t.Fatalf("navigator.userAgent execute failed: %v", err)
	}
	if ua != TestUserAgent {
		t.Fatalf("unexpected user agent: got %v want %q", ua, TestUserAgent)
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

	found, err = page.WaitForSelector("#missing", map[string]any{
		"timeoutMs":      200,
		"pollIntervalMs": 25,
	})
	if err != nil {
		t.Fatalf("waitForSelector timeout case failed: %v", err)
	}
	if found {
		t.Fatal("expected missing selector wait to time out")
	}

	cookieTestURL := opts.CookieTestURL
	if cookieTestURL == "" {
		cookieTestURL = testURL
	}
	runCookieLifecycleAssertions(t, page, server, cookieTestURL, opts.CookieDomainMode)
}

func waitForProviderAvailable(t *testing.T, provider enginewebview.Provider, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if provider != nil && provider.IsAvailable() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("provider did not become available within %s", timeout)
}

func openTestPage(t *testing.T, provider enginewebview.Provider, opts enginewebview.OpenOptions) (*enginewebview.PageHandle, *httptest.Server, string) {
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

	runtime := enginewebview.NewRuntime(provider, true)
	page, err := runtime.Open(map[string]any{
		"headless":  opts.Headless,
		"debug":     opts.Debug,
		"title":     opts.Title,
		"width":     opts.Width,
		"height":    opts.Height,
		"userAgent": opts.UserAgent,
	})
	if err != nil {
		t.Fatalf("failed to open webview: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.Close()
	})

	return page, server, server.URL
}

func runCookieLifecycleAssertions(t *testing.T, page *enginewebview.PageHandle, _ *httptest.Server, testURL string, domainMode CookieDomainMode) {
	t.Helper()

	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("parse cookie test url failed: %v", err)
	}
	cookieDomain := parsedURL.Hostname()

	if err := page.ClearCookies(); err != nil {
		t.Fatalf("clearCookies before cookie assertions failed: %v", err)
	}

	visibleCookie := map[string]any{
		"name":  "visible",
		"value": "1",
		"path":  "/",
	}
	secretCookie := map[string]any{
		"name":     "secret",
		"value":    "2",
		"path":     "/",
		"httpOnly": true,
		"expires":  time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	}
	deleteCookie := map[string]any{
		"name": "visible",
		"path": "/",
	}
	if domainMode == CookieDomainModeRequired {
		visibleCookie["domain"] = cookieDomain
		secretCookie["domain"] = cookieDomain
		deleteCookie["domain"] = cookieDomain
	}

	if err := page.SetCookie(visibleCookie); err != nil {
		t.Fatalf("set visible cookie failed: %v", err)
	}
	if err := page.SetCookie(secretCookie); err != nil {
		t.Fatalf("set secret cookie failed: %v", err)
	}

	if err := page.Goto(testURL, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatalf("goto after setCookie failed: %v", err)
	}

	documentCookie, err := page.Execute(`() => document.cookie`)
	if err != nil {
		t.Fatalf("document.cookie execute failed: %v", err)
	}
	documentCookieText, ok := documentCookie.(string)
	if !ok {
		t.Fatalf("unexpected document.cookie payload: %T", documentCookie)
	}
	if !strings.Contains(documentCookieText, "visible=1") {
		t.Fatalf("expected visible cookie in document.cookie, got %q", documentCookieText)
	}
	if strings.Contains(documentCookieText, "secret=2") {
		t.Fatalf("expected httpOnly cookie to be hidden from document.cookie, got %q", documentCookieText)
	}

	cookies, err := page.GetCookies()
	if err != nil {
		t.Fatalf("getCookies failed: %v", err)
	}
	if len(cookies) < 2 {
		t.Fatalf("expected at least 2 cookies, got %#v", cookies)
	}
	if !hasCookie(cookies, "visible", "1") {
		t.Fatalf("expected visible cookie in native cookie store, got %#v", cookies)
	}
	if !hasCookie(cookies, "secret", "2") {
		t.Fatalf("expected secret cookie in native cookie store, got %#v", cookies)
	}

	if err := page.DeleteCookie(deleteCookie); err != nil {
		t.Fatalf("deleteCookie failed: %v", err)
	}
	if err := page.Goto(testURL, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatalf("goto after deleteCookie failed: %v", err)
	}

	documentCookie, err = page.Execute(`() => document.cookie`)
	if err != nil {
		t.Fatalf("document.cookie execute after delete failed: %v", err)
	}
	documentCookieText, ok = documentCookie.(string)
	if !ok {
		t.Fatalf("unexpected document.cookie payload after delete: %T", documentCookie)
	}
	if strings.Contains(documentCookieText, "visible=1") {
		t.Fatalf("expected visible cookie to be deleted, got %q", documentCookieText)
	}
	if strings.Contains(documentCookieText, "secret=2") {
		t.Fatalf("expected httpOnly cookie to remain hidden after delete, got %q", documentCookieText)
	}

	cookies, err = page.GetCookies()
	if err != nil {
		t.Fatalf("getCookies after delete failed: %v", err)
	}
	if hasCookie(cookies, "visible", "1") {
		t.Fatalf("expected visible cookie to be removed from native store, got %#v", cookies)
	}
	if !hasCookie(cookies, "secret", "2") {
		t.Fatalf("expected secret cookie to remain in native store after delete, got %#v", cookies)
	}

	if err := page.ClearCookies(); err != nil {
		t.Fatalf("clearCookies failed: %v", err)
	}
	if err := page.Goto(testURL, map[string]any{"timeoutMs": 10000}); err != nil {
		t.Fatalf("goto after clearCookies failed: %v", err)
	}
	documentCookie, err = page.Execute(`() => document.cookie`)
	if err != nil {
		t.Fatalf("document.cookie execute after clear failed: %v", err)
	}
	documentCookieText, ok = documentCookie.(string)
	if !ok {
		t.Fatalf("unexpected document.cookie payload after clear: %T", documentCookie)
	}
	if documentCookieText != "" {
		t.Fatalf("expected document.cookie to be empty after clear, got %q", documentCookieText)
	}
	cookies, err = page.GetCookies()
	if err != nil {
		t.Fatalf("getCookies after clear failed: %v", err)
	}
	if len(cookies) != 0 {
		t.Fatalf("expected cookie store to be empty after clear, got %#v", cookies)
	}
}

func hasCookie(cookies []enginewebview.Cookie, name string, value string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value == value {
			return true
		}
	}
	return false
}
