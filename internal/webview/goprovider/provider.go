//go:build cgo

package goprovider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	runtimepkg "runtime"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	webview "github.com/GopeedLab/webview_go"
)

type Provider struct{}

func New() enginewebview.Provider {
	return &Provider{}
}

func (p *Provider) IsAvailable() bool {
	return webview.IsAvailable()
}

func (p *Provider) Open(opts enginewebview.OpenOptions) (enginewebview.Page, error) {
	pw := newPageWrapper(opts)
	if err := pw.start(); err != nil {
		return nil, err
	}
	return pw, nil
}

type pageWrapper struct {
	opts             enginewebview.OpenOptions
	events           enginewebview.EventHub
	navigationErrors chan error

	callbackName string
	readyID      string

	view webview.WebView

	mu      sync.Mutex
	pending map[string]chan evalResult
	url     string
	closed  bool

	ready        chan error
	done         chan struct{}
	loads        chan string
	domReady     chan struct{}
	sameDocument chan struct{}
}

type evalMessage struct {
	Event        string `json:"event,omitempty"`
	SameDocument bool   `json:"sameDocument,omitempty"`
	ID           string `json:"id"`
	Ready        bool   `json:"ready,omitempty"`
	Load         bool   `json:"load,omitempty"`
	URL          string `json:"url,omitempty"`
	OK           bool   `json:"ok,omitempty"`
	Value        any    `json:"value,omitempty"`
	Error        string `json:"error,omitempty"`
}

type evalResult struct {
	value any
	err   error
}

type nativeCookieJSON struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  string `json:"expires"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"httpOnly"`
}

var providerSeq atomic.Uint64

// Keep native page creation aligned with the provider availability budget used
// by the integration contract. Cold WebView2 startup can exceed ten seconds.
const startupTimeout = time.Minute

func newPageWrapper(opts enginewebview.OpenOptions) *pageWrapper {
	seq := providerSeq.Add(1)
	return &pageWrapper{
		opts:             opts,
		navigationErrors: make(chan error, 8),
		callbackName:     fmt.Sprintf("__gopeedWebViewCallback_%d", seq),
		readyID:          fmt.Sprintf("__ready__%d", seq),
		pending:          make(map[string]chan evalResult),
		ready:            make(chan error, 1),
		done:             make(chan struct{}),
		loads:            make(chan string, 8),
		domReady:         make(chan struct{}, 8),
		sameDocument:     make(chan struct{}, 8),
	}
}

func (p *pageWrapper) start() error {
	startedAt := time.Now()
	run := func() {
		runtimepkg.LockOSThread()
		defer runtimepkg.UnlockOSThread()

		var w webview.WebView
		if p.opts.DataPath != "" {
			var err error
			w, err = webview.NewWithOptions(webview.Options{Debug: p.opts.Debug, Headless: p.opts.Headless, DataPath: p.opts.DataPath, ProxyURL: p.opts.ProxyURL})
			if err != nil {
				p.ready <- err
				close(p.done)
				return
			}
		} else if p.opts.Headless {
			w = webview.NewHeadless(p.opts.Debug)
		} else {
			w = webview.New(p.opts.Debug)
		}
		p.view = w
		w.SetEventHandler(p.handleNativeEvent)
		w.SetTitle(firstNonEmpty(p.opts.Title, "Gopeed WebView"))
		w.SetSize(defaultWindowDimension(p.opts.Width, 1280), defaultWindowDimension(p.opts.Height, 800), webview.HintNone)

		if p.opts.UserAgent != "" {
			type userAgentSetter interface {
				SetUserAgent(userAgent string)
			}
			if setter, ok := w.(userAgentSetter); ok {
				setter.SetUserAgent(p.opts.UserAgent)
			}
		}

		if err := w.Bind(p.callbackName, func(payload string) error {
			return p.handleCallback(payload)
		}); err != nil {
			w.Destroy()
			p.ready <- err
			close(p.done)
			return
		}

		w.Init(buildLoadNotifyScript(p.callbackName))
		applyWindowOptions(w, p.opts)
		w.SetHtml(buildBootstrapHTML(p.callbackName, p.readyID))
		if runtimepkg.GOOS == "linux" {
			// All GTK pages share the process-wide event loop below.
			return
		}
		w.Run()
		p.handleNativeEvent(webview.Event{Name: "closed"})
		w.Destroy()
		close(p.done)
	}

	if !postMainThreadTask(run) {
		go run()
	}

	select {
	case err := <-p.ready:
		p.tracef("open %dms", time.Since(startedAt).Milliseconds())
		return err
	case <-time.After(startupTimeout):
		p.tracef("open timeout %dms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("webview startup timeout")
	}
}

func (p *pageWrapper) AddInitScript(script string) error {
	return p.dispatch(func(w webview.WebView) error {
		w.Init(script)
		return nil
	})
}

func (p *pageWrapper) Goto(url string, opts enginewebview.GotoOptions) error {
	startedAt := time.Now()
	p.drainLoads()
	for {
		select {
		case <-p.navigationErrors:
			continue
		default:
		}
		break
	}
	if err := p.dispatch(func(w webview.WebView) error {
		w.Navigate(url)
		return nil
	}); err != nil {
		return err
	}
	timeout := 30 * time.Second
	if opts.TimeoutMS > 0 {
		timeout = time.Duration(opts.TimeoutMS) * time.Millisecond
	}
	waitUntil := opts.WaitUntil
	if waitUntil == "" {
		waitUntil = "load"
	}
	err := p.waitForNavigation(timeout, waitUntil)
	if err != nil {
		p.tracef("goto failed %dms url=%s err=%v", time.Since(startedAt).Milliseconds(), url, err)
		return err
	}
	p.tracef("goto %dms url=%s", time.Since(startedAt).Milliseconds(), url)
	return nil
}

func (p *pageWrapper) Execute(expression string, args ...any) (any, error) {
	startedAt := time.Now()
	if args == nil {
		args = []any{}
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	reqID := fmt.Sprintf("req-%d", providerSeq.Add(1))
	ch := make(chan evalResult, 1)

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("webview page is closed")
	}
	p.pending[reqID] = ch
	p.mu.Unlock()

	script := buildExecuteScript(p.callbackName, reqID, expression, string(argsJSON))
	if err := p.dispatch(func(w webview.WebView) error {
		w.Eval(script)
		return nil
	}); err != nil {
		p.mu.Lock()
		delete(p.pending, reqID)
		p.mu.Unlock()
		return nil, err
	}

	var result evalResult
	select {
	case result = <-ch:
	case <-p.done:
		return nil, fmt.Errorf("webview page is closed")
	}
	if result.err != nil {
		p.tracef("execute failed %dms err=%v", time.Since(startedAt).Milliseconds(), result.err)
		return result.value, result.err
	}
	p.tracef("execute %dms", time.Since(startedAt).Milliseconds())
	return result.value, nil
}

func (p *pageWrapper) GetCookies() ([]enginewebview.Cookie, error) {
	cookies, err := p.nativeCookies()
	if err != nil {
		return nil, err
	}
	result := make([]enginewebview.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		item := enginewebview.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
		}
		if cookie.Expires != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, cookie.Expires); err == nil {
				item.Expires = parsed
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (p *pageWrapper) SetCookie(cookie enginewebview.Cookie) error {
	type cookieSetter interface {
		SetCookie(cookie webview.Cookie) error
	}
	return p.dispatch(func(w webview.WebView) error {
		setter, ok := w.(cookieSetter)
		if !ok {
			return fmt.Errorf("native cookie access is unavailable")
		}
		return setter.SetCookie(webview.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  cookie.Expires,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
		})
	})
}

func (p *pageWrapper) DeleteCookie(cookie enginewebview.Cookie) error {
	type cookieDeleter interface {
		DeleteCookie(name string, domain string, path string) error
	}
	return p.dispatch(func(w webview.WebView) error {
		deleter, ok := w.(cookieDeleter)
		if !ok {
			return fmt.Errorf("native cookie access is unavailable")
		}
		return deleter.DeleteCookie(cookie.Name, cookie.Domain, cookie.Path)
	})
}

func (p *pageWrapper) ClearCookies() error {
	type cookieClearer interface {
		ClearCookies() error
	}
	return p.dispatch(func(w webview.WebView) error {
		clearer, ok := w.(cookieClearer)
		if !ok {
			return fmt.Errorf("native cookie access is unavailable")
		}
		return clearer.ClearCookies()
	})
}

func (p *pageWrapper) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.failPendingLocked()
	w := p.view
	wait := p.done
	p.mu.Unlock()

	if w != nil {
		// PostQuitMessage (used by the Windows WebView2 terminate_impl)
		// only takes effect on the thread that owns the message loop.
		// Dispatch the termination to the webview's UI thread so that
		// the quit message lands on the correct queue.
		terminateDone := make(chan struct{})
		w.Dispatch(func() {
			if runtimepkg.GOOS == "linux" {
				w.Destroy()
				close(p.done)
			} else {
				w.Terminate()
			}
			close(terminateDone)
		})
		<-terminateDone
	}
	<-wait
	p.events.Emit(enginewebview.Event{Name: "closed", Data: map[string]any{"reason": "api"}})
	return nil
}

func (p *pageWrapper) tracef(format string, args ...any) {
	if !strings.Contains(p.opts.Title, "youtube-sabr") {
		return
	}
	fmt.Printf("goprovider "+format+"\n", args...)
}

func (p *pageWrapper) waitForNavigation(timeout time.Duration, waitUntil string) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-p.done:
			return fmt.Errorf("webview page is closed")
		case err := <-p.navigationErrors:
			return err
		case <-timer.C:
			return fmt.Errorf("webview navigation timeout")
		case <-p.sameDocument:
			return nil
		case <-p.domReady:
			if waitUntil == "domcontentloaded" {
				return nil
			}
		case <-p.loads:
			return nil
		}
	}
}

func (p *pageWrapper) dispatch(fn func(w webview.WebView) error) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("webview page is closed")
	}
	w := p.view
	p.mu.Unlock()
	if w == nil {
		return fmt.Errorf("webview is not initialized")
	}

	done := make(chan error, 1)
	w.Dispatch(func() {
		done <- fn(w)
	})
	select {
	case err := <-done:
		return err
	case <-p.done:
		return fmt.Errorf("webview page is closed")
	}
}

func (p *pageWrapper) nativeCookies() ([]nativeCookieJSON, error) {
	type cookieGetter interface {
		GetCookies(url string) ([]webview.Cookie, error)
	}

	var cookies []webview.Cookie
	err := p.dispatch(func(w webview.WebView) error {
		getter, ok := w.(cookieGetter)
		if !ok {
			return fmt.Errorf("native cookie access is unavailable")
		}
		p.mu.Lock()
		url := p.url
		p.mu.Unlock()
		if url == "" {
			url = "about:blank"
		}
		var err error
		cookies, err = getter.GetCookies(url)
		return err
	})
	if err != nil {
		return nil, err
	}

	result := make([]nativeCookieJSON, 0, len(cookies))
	for _, cookie := range cookies {
		item := nativeCookieJSON{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
		}
		if !cookie.Expires.IsZero() {
			item.Expires = cookie.Expires.UTC().Format(time.RFC3339Nano)
		}
		result = append(result, item)
	}
	return result, nil
}

func (p *pageWrapper) handleCallback(payload string) error {
	var msg evalMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return err
	}
	if msg.Ready && msg.ID == p.readyID {
		select {
		case p.ready <- nil:
		default:
		}
		return nil
	}
	if msg.Event == "domcontentloaded" {
		select {
		case p.domReady <- struct{}{}:
		default:
		}
		return nil
	}
	if msg.Event == "url-changed" {
		if msg.SameDocument {
			select {
			case p.sameDocument <- struct{}{}:
			default:
			}
		}
		p.mu.Lock()
		changed := p.url != msg.URL
		p.url = msg.URL
		p.mu.Unlock()
		if changed {
			p.events.Emit(enginewebview.Event{Name: "url-changed", Data: map[string]any{"url": msg.URL, "sameDocument": msg.SameDocument}})
		}
		return nil
	}
	if msg.Load {
		p.events.Emit(enginewebview.Event{Name: "load", Data: map[string]any{"url": msg.URL}})
		select {
		case p.loads <- msg.URL:
		default:
		}
		return nil
	}

	p.mu.Lock()
	ch := p.pending[msg.ID]
	delete(p.pending, msg.ID)
	p.mu.Unlock()
	if ch == nil {
		return nil
	}

	if !msg.OK {
		ch <- evalResult{err: errors.New(strings.TrimSpace(msg.Error))}
		return nil
	}
	ch <- evalResult{value: msg.Value}
	return nil
}

func (p *pageWrapper) drainLoads() {
	for {
		select {
		case <-p.domReady:
			continue
		default:
		}
		break
	}
	for {
		select {
		case <-p.sameDocument:
			continue
		default:
		}
		break
	}
	for {
		select {
		case <-p.loads:
		default:
			return
		}
	}
}

func buildBootstrapHTML(callbackName string, readyID string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
<head><meta charset="utf-8"></head>
<body>
<script>
window.addEventListener("load", () => {
  const callback = globalThis[%q];
  if (typeof callback === "function") {
    callback(JSON.stringify({ id: %q, ready: true }));
  }
}, { once: true });
</script>
</body>
</html>`, callbackName, readyID)
}

func buildExecuteScript(callbackName string, reqID string, expression string, argsJSON string) string {
	callbackJSON, _ := json.Marshal(callbackName)
	reqJSON, _ := json.Marshal(reqID)
	exprJSON, _ := json.Marshal(expression)
	argsLiteral := argsJSON
	if argsLiteral == "" || argsLiteral == "null" {
		argsLiteral = "[]"
	}
	return fmt.Sprintf(`(() => {
  const __cb = globalThis[%s];
  const __id = %s;
  const __expression = %s;
  const __args = %s;
  Promise.resolve()
    .then(async () => {
      const __target = (0, eval)(__expression);
      const __value = await (typeof __target === "function" ? __target(...__args) : __target);
      return { id: __id, ok: true, value: __value ?? null };
    })
    .catch((error) => ({
      id: __id,
      ok: false,
      error: [
        error && error.name ? error.name : "Error",
        error && error.message ? error.message : String(error),
        error && error.stack ? error.stack : "",
      ].filter(Boolean).join(": ").replace(": @", "\n@"),
    }))
    .then((message) => __cb(JSON.stringify(message)));
})();`, string(callbackJSON), string(reqJSON), string(exprJSON), argsLiteral)
}

func buildLoadNotifyScript(callbackName string) string {
	callbackJSON, _ := json.Marshal(callbackName)
	return fmt.Sprintf(`(() => {
  if (window !== window.top) return;
  const send = (message) => {
    const cb = globalThis[%s];
    if (typeof cb === "function") cb(JSON.stringify(message));
  };
  let lastURL = String(location.href);
  const changed = () => {
    const url = String(location.href);
    if (url === lastURL) return;
    lastURL = url;
    send({event: "url-changed", url, sameDocument: true});
  };
  send({event: "url-changed", url: lastURL, sameDocument: false});
  for (const name of ["pushState", "replaceState"]) {
    const original = history[name];
    history[name] = function(...args) {
      const result = Reflect.apply(original, this, args);
      changed();
      return result;
    };
  }
  window.addEventListener("hashchange", changed);
  window.addEventListener("popstate", changed);
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) send({event: "url-changed", url: String(location.href), sameDocument: false});
    else changed();
  });
  window.addEventListener("DOMContentLoaded", () => send({event: "domcontentloaded"}), {once:true});
  window.addEventListener("load", () => send({load: true, url: String(location.href)}), {once: true});
})();`, string(callbackJSON))
}

func (p *pageWrapper) On(name string, handler func(enginewebview.Event)) (func(), error) {
	return p.events.On(name, handler)
}

func (p *pageWrapper) failPendingLocked() {
	for id, ch := range p.pending {
		ch <- evalResult{err: fmt.Errorf("webview page is closed")}
		delete(p.pending, id)
	}
}

func (p *pageWrapper) handleNativeEvent(event webview.Event) {
	switch event.Name {
	case "load-error":
		p.events.Emit(enginewebview.Event{Name: "load-error", Data: map[string]any{"url": event.URL, "message": event.Message}})
		select {
		case p.navigationErrors <- errors.New(event.Message):
		default:
		}
	case "closed":
		p.mu.Lock()
		api := p.closed
		p.closed = true
		p.failPendingLocked()
		p.mu.Unlock()
		reason := "user"
		if api {
			reason = "api"
		}
		p.events.Emit(enginewebview.Event{Name: "closed", Data: map[string]any{"reason": reason}})
		if runtimepkg.GOOS == "linux" && !api {
			p.view.Dispatch(func() { p.view.Destroy(); close(p.done) })
		}
	}
}

func defaultWindowDimension(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (p *Provider) RemoveProfile(profileID, dataPath string) error {
	done := make(chan error, 1)
	run := func() { done <- webview.RemoveProfile(dataPath) }
	if !postMainThreadTask(run) {
		run()
	}
	return <-done
}
