package download

import (
	"crypto/sha256"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/base"
	"net/http"
	"net/url"
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
}

func (o *extensionWebViewOpener) Open(opts enginewebview.OpenOptions) (enginewebview.Page, error) {
	if o.identity == "" {
		return nil, fmt.Errorf("webview extension identity is required")
	}
	sum := sha256.Sum256([]byte(o.identity))
	opts.ProfileID = fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
	path, err := filepath.Abs(filepath.Join(o.downloader.cfg.StorageDir, "webview", fmt.Sprintf("%x", sum[:])))
	if err != nil {
		return nil, err
	}
	opts.DataPath = path
	opts.ProxyURL, err = o.downloader.webviewProxyURL()
	if err != nil {
		return nil, err
	}
	return o.provider.Open(opts)
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
