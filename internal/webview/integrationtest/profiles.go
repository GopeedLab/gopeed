package integrationtest

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	webviewproxy "github.com/GopeedLab/gopeed/internal/webview/proxy"
	webview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

// RunProfileContract exercises the same host API through both native and RPC
// providers. It never bypasses the host bridge with direct Flutter plugin calls.
func RunProfileContract(t *testing.T, provider webview.Provider) {
	t.Helper()
	waitForProviderAvailable(t, provider, defaultAvailabilityTimeout)
	remover, ok := provider.(webview.ProfileRemover)
	if !ok {
		t.Fatal("provider does not implement host profile removal")
	}
	proxyURL, secureHits := profileProxy(t)
	a := newContractProfile(t, provider, remover, proxyURL)
	b := newContractProfile(t, provider, remover, proxyURL)
	first := a.open(t)
	assertProfileValue(t, first, `async () => { localStorage.setItem('owner','a'); return 'a'; }`, "a")
	assertProfileValue(t, first, profileDatabaseScript(true), "saved")
	if err := first.SetCookie(webview.Cookie{Name: "session", Value: "a", Domain: "profiles.invalid", Path: "/", HTTPOnly: true, Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	second := b.open(t)
	assertProfileValue(t, second, `() => localStorage.getItem('owner')`, nil)
	assertProfileValue(t, second, profileDatabaseScript(false), nil)
	assertProfileCookies(t, second, "", "new profile B")
	assertProfileValue(t, second, `() => { localStorage.setItem('owner','b'); return 'b'; }`, "b")
	if err := second.SetCookie(webview.Cookie{Name: "session", Value: "b", Domain: "profiles.invalid", Path: "/", HTTPOnly: true}); err != nil {
		t.Fatal(err)
	}
	shared := a.open(t)
	assertProfileValue(t, shared, `() => localStorage.getItem('owner')`, "a")
	assertProfileValue(t, shared, profileDatabaseScript(false), "saved")
	assertProfileCookies(t, shared, "a", "shared profile A")
	storedCookies, err := shared.GetCookies()
	if err != nil || len(storedCookies) != 1 || storedCookies[0].Expires.Before(time.Now()) {
		t.Fatalf("persistent cookie lost its expiration: %#v, error %v", storedCookies, err)
	}
	assertProfileValue(t, shared, `() => document.cookie`, "") // HttpOnly stays hidden.
	if err := second.ClearCookies(); err != nil {
		t.Fatal(err)
	}
	assertProfileCookies(t, first, "a", "clearing B leaves A intact")
	if err := second.SetCookie(webview.Cookie{Name: "session", Value: "b", Domain: "profiles.invalid", Path: "/", HTTPOnly: true}); err != nil {
		t.Fatal(err)
	}
	a.close(t)
	reopened := a.open(t)
	assertProfileValue(t, reopened, `() => localStorage.getItem('owner')`, "a")
	assertProfileValue(t, reopened, profileDatabaseScript(false), "saved")
	assertProfileCookies(t, reopened, "a", "reopened profile A")
	// Include a session cookie immediately before deletion to catch stale
	// in-process CookieStore caches when the same profile is recreated.
	if err := reopened.SetCookie(webview.Cookie{Name: "session", Value: "remove-me", Domain: "profiles.invalid", Path: "/", HTTPOnly: true}); err != nil {
		t.Fatal(err)
	}
	a.close(t)
	a.remove(t)
	a.remove(t) // Retrying a successful uninstall is harmless.
	reinstalled := a.open(t)
	assertProfileValue(t, reinstalled, `() => localStorage.getItem('owner')`, nil)
	assertProfileValue(t, reinstalled, profileDatabaseScript(false), nil)
	assertProfileCookies(t, reinstalled, "", "reinstalled profile A")
	assertProfileValue(t, second, `() => localStorage.getItem('owner')`, "b")
	assertProfileCookies(t, second, "b", "B after removing A")
	// Refuse the HTTPS tunnel deliberately. Seeing it at our proxy proves HTTPS
	// routing without adding a certificate-validation bypass to the test host.
	// WebView2 may commit its built-in error page and report navigation as
	// complete. The invariant here is that the HTTPS tunnel reaches the proxy.
	_ = reinstalled.Goto("https://profiles.invalid/", webview.GotoOptions{TimeoutMS: 5000})
	deadline := time.Now().Add(5 * time.Second)
	for secureHits.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if secureHits.Load() == 0 {
		t.Fatal("HTTPS bypassed the host proxy")
	}
}

type contractProfile struct {
	provider webview.Provider
	remover  webview.ProfileRemover
	options  webview.OpenOptions
	pages    []webview.Page
}

func newContractProfile(t *testing.T, provider webview.Provider, remover webview.ProfileRemover, proxyURL string) *contractProfile {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile")
	hash := sha256.Sum256([]byte(path))
	id := fmt.Sprintf("%x-%x-%x-%x-%x", hash[:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
	p := &contractProfile{provider: provider, remover: remover, options: webview.OpenOptions{ProfileID: id, DataPath: path, ProxyURL: proxyURL, Headless: true, Title: "Gopeed profile contract"}}
	t.Cleanup(func() { p.close(t); p.remove(t) })
	return p
}
func (p *contractProfile) open(t *testing.T) webview.Page {
	t.Helper()
	page, err := p.provider.Open(p.options)
	if err != nil {
		t.Fatal(err)
	}
	p.pages = append(p.pages, page)
	if err := page.Goto("http://profiles.invalid/", webview.GotoOptions{TimeoutMS: 10000}); err != nil {
		t.Fatal(err)
	}
	return page
}
func (p *contractProfile) close(t *testing.T) {
	t.Helper()
	for _, page := range p.pages {
		if err := page.Close(); err != nil {
			t.Errorf("close profile page: %v", err)
		}
	}
	p.pages = nil
}
func (p *contractProfile) remove(t *testing.T) {
	t.Helper()
	if err := p.remover.RemoveProfile(p.options.ProfileID, p.options.DataPath); err != nil {
		t.Errorf("remove profile: %v", err)
	}
}
func assertProfileValue(t *testing.T, page webview.Page, script string, want any) {
	t.Helper()
	got, err := page.Execute(script)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("script %s: got %#v, want %#v", script, got, want)
	}
}
func assertProfileCookies(t *testing.T, page webview.Page, want, stage string) {
	t.Helper()
	// Native cookie managers propagate updates across browser processes
	// asynchronously. Wait for the expected state rather than assuming the
	// first IPC read already includes the preceding write.
	deadline := time.Now().Add(5 * time.Second)
	for {
		cookies, err := page.GetCookies()
		if err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		matches := len(cookies) == 0 && want == "" || len(cookies) == 1 && want != "" && cookies[0].Name == "session" && cookies[0].Value == want && cookies[0].HTTPOnly
		if matches {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: cookies %#v, want %q", stage, cookies, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func profileDatabaseScript(write bool) string {
	operation := "store.get('owner')"
	if write {
		operation = "store.put('saved','owner')"
	}
	return fmt.Sprintf(`async () => await new Promise((resolve,reject)=>{
 const request=indexedDB.open('profile-contract',1);
 request.onupgradeneeded=()=>request.result.createObjectStore('state');
 request.onerror=()=>reject(String(request.error));
 request.onsuccess=()=>{
  const db=request.result, tx=db.transaction('state','readwrite'), store=tx.objectStore('state');
  const op=%s; let value;
  op.onsuccess=()=>value=%s;
  tx.oncomplete=()=>{db.close();resolve(value===undefined?null:value)};
  tx.onerror=()=>{db.close();reject(String(tx.error))};
 };
})`, operation, map[bool]string{true: "'saved'", false: "op.result"}[write])
}

func profileProxy(t *testing.T) (string, *atomic.Int32) {
	t.Helper()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprint(w, "<!doctype html><title>Profile contract</title>")
	}))
	t.Cleanup(origin.Close)
	secureHits := &atomic.Int32{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			origin.Config.Handler.ServeHTTP(w, r)
			return
		}
		if r.Host == "profiles.invalid:443" {
			secureHits.Add(1)
			http.Error(w, "HTTPS routing fixture rejects tunnels", http.StatusBadGateway)
			return
		}
		if r.Host != "profiles.invalid:80" {
			http.Error(w, "unexpected target", http.StatusBadGateway)
			return
		}
		remote, err := net.DialTimeout("tcp", origin.Listener.Addr().String(), 5*time.Second)
		if err != nil {
			http.Error(w, "origin unavailable", http.StatusBadGateway)
			return
		}
		defer remote.Close()
		client, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer client.Close()
		if _, err := buffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			return
		}
		if err := buffer.Flush(); err != nil {
			return
		}
		go func() { io.Copy(remote, buffer); remote.Close() }()
		io.Copy(client, remote)
	}))
	t.Cleanup(upstream.Close)
	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := webviewproxy.New(func(*http.Request) (*url.URL, error) { return upstreamURL, nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proxy.Close() })
	if runtime.GOOS == "darwin" {
		return proxy.SOCKSURL(), secureHits
	}
	return proxy.URL(), secureHits
}
