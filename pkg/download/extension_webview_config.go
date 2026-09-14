package download

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/GopeedLab/gopeed/pkg/base"
	"path/filepath"
	"runtime"

	webviewproxy "github.com/GopeedLab/gopeed/internal/webview/proxy"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

// extensionWebViewOpener binds browser state to the owning extension. Neither
// identity, filesystem paths nor proxy settings are accepted from JavaScript.
type extensionWebViewOpener struct {
	downloader *Downloader
	provider   enginewebview.Provider
	identity   string
	profile    *extensionWebViewProfile
}

func (o *extensionWebViewOpener) Open(opts enginewebview.OpenOptions) (enginewebview.Page, error) {
	if o.identity == "" {
		return nil, fmt.Errorf("webview extension identity is required")
	}
	profile := o.profile
	if profile == nil {
		profile = o.downloader.extensionWebViewProfile(o.identity)
	}
	profile.mu.Lock()
	defer profile.mu.Unlock()
	if profile.revoked {
		return nil, fmt.Errorf("extension WebView profile has been removed")
	}
	id, path, err := o.downloader.extensionWebViewLocation(o.identity)
	if err != nil {
		return nil, err
	}
	opts.ProfileID, opts.DataPath = id, path
	opts.ProxyURL, err = o.downloader.webviewProxyURL()
	if err != nil {
		return nil, err
	}
	page, err := o.provider.Open(opts)
	if err != nil {
		return nil, err
	}
	tracked := &extensionWebViewPage{Page: page, profile: profile}
	profile.pages[tracked] = struct{}{}
	return tracked, nil
}

func (d *Downloader) webviewProxyURL() (string, error) {
	d.webviewProxyLock.Lock()
	defer d.webviewProxyLock.Unlock()
	if d.closed.Load() {
		return "", fmt.Errorf("downloader is closed")
	}
	if d.webviewProxy == nil {
		server, err := webviewproxy.New(func(r *http.Request) (*url.URL, error) {
			if d.blob != nil && d.blob.IsURL(r.URL.String()) {
				return nil, nil
			}
			d.configLock.RLock()
			var cfg base.DownloaderProxyConfig
			if d.cfg.Proxy != nil {
				cfg = *d.cfg.Proxy
			}
			d.configLock.RUnlock()
			handler := cfg.ToHandler()
			if handler == nil {
				return nil, nil
			}
			return handler(r)
		})
		if err != nil {
			return "", err
		}
		d.webviewProxy = server
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return d.webviewProxy.SOCKSURL(), nil
	}
	return d.webviewProxy.URL(), nil
}

// A revoked profile is retained until a fresh installation. Existing runtimes
// retain this object, so reinstalling cannot let an old runtime recreate data.
type extensionWebViewProfile struct {
	mu       sync.Mutex
	deleteMu sync.Mutex
	revoked  bool
	pages    map[*extensionWebViewPage]struct{}
}

type extensionWebViewPage struct {
	enginewebview.Page
	profile *extensionWebViewProfile
	mu      sync.Mutex
	closed  bool
}

func (p *extensionWebViewPage) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	if err := p.Page.Close(); err != nil {
		return err
	}
	p.closed = true
	p.profile.mu.Lock()
	delete(p.profile.pages, p)
	p.profile.mu.Unlock()
	return nil
}

func (d *Downloader) extensionWebViewProfile(identity string) *extensionWebViewProfile {
	d.webviewProfilesLock.Lock()
	defer d.webviewProfilesLock.Unlock()
	if d.webviewProfiles == nil {
		d.webviewProfiles = make(map[string]*extensionWebViewProfile)
	}
	profile := d.webviewProfiles[identity]
	if profile == nil {
		profile = &extensionWebViewProfile{pages: make(map[*extensionWebViewPage]struct{})}
		d.webviewProfiles[identity] = profile
	}
	return profile
}

func (d *Downloader) extensionWebViewLocation(identity string) (string, string, error) {
	if identity == "" {
		return "", "", fmt.Errorf("webview extension identity is required")
	}
	sum := sha256.Sum256([]byte(identity))
	id := fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
	path, err := filepath.Abs(filepath.Join(d.cfg.StorageDir, "webview", fmt.Sprintf("%x", sum[:])))
	return id, path, err
}

func (d *Downloader) removeExtensionWebViewProfile(identity string) error {
	profile := d.extensionWebViewProfile(identity)
	profile.deleteMu.Lock()
	defer profile.deleteMu.Unlock()
	profile.mu.Lock()
	profile.revoked = true
	pages := make([]*extensionWebViewPage, 0, len(profile.pages))
	for page := range profile.pages {
		pages = append(pages, page)
	}
	profile.mu.Unlock()
	for _, page := range pages {
		if err := page.Close(); err != nil {
			return err
		}
	}
	id, path, err := d.extensionWebViewLocation(identity)
	if err != nil {
		return err
	}
	if provider := d.cfg.WebViewProvider; provider != nil {
		remover, ok := provider.(enginewebview.ProfileRemover)
		if !ok {
			// An unavailable backend cannot have created a profile this run.
			// Still fail if a persisted native profile directory exists.
			_, statErr := os.Stat(path)
			if provider.IsAvailable() || len(pages) != 0 || !os.IsNotExist(statErr) {
				return fmt.Errorf("WebView provider does not support profile removal")
			}
		} else if err := remover.RemoveProfile(id, path); err != nil {
			return err
		}
	}
	return os.RemoveAll(path)
}
