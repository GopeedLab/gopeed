package download

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

type profileCaptureProvider struct{ options []enginewebview.OpenOptions }

func (*profileCaptureProvider) IsAvailable() bool { return true }
func (p *profileCaptureProvider) Open(opts enginewebview.OpenOptions) (enginewebview.Page, error) {
	p.options = append(p.options, opts)
	return fakeRuntimeWebViewPage{}, nil
}

func TestWebViewProfileBoundToExtension(t *testing.T) {
	root := t.TempDir()
	d := &Downloader{cfg: &DownloaderConfig{StorageDir: root, DownloaderStoreConfig: &base.DownloaderStoreConfig{}}, configLock: &sync.RWMutex{}}
	t.Cleanup(func() {
		if d.webviewProxy != nil {
			d.webviewProxy.Close()
		}
	})
	provider := &profileCaptureProvider{}
	for _, id := range []string{"extension-a", "extension-b", "extension-a", "../../escape"} {
		opener := &extensionWebViewOpener{downloader: d, provider: provider, identity: id}
		_, err := opener.Open(enginewebview.OpenOptions{ProfileID: "forged", DataPath: "/tmp/forged", ProxyURL: "http://wrong:80"})
		if err != nil {
			t.Fatal(err)
		}
	}
	a, b, reopen := provider.options[0], provider.options[1], provider.options[2]
	if a.ProfileID == b.ProfileID || a.DataPath == b.DataPath {
		t.Fatal("different extensions shared browser storage")
	}
	if a.ProfileID != reopen.ProfileID || a.DataPath != reopen.DataPath {
		t.Fatal("same extension lost profile identity")
	}
	for _, opts := range provider.options {
		if filepath.Dir(opts.DataPath) != filepath.Join(root, "webview") {
			t.Fatalf("profile escaped storage root: %s", opts.DataPath)
		}
		if opts.ProxyURL != d.webviewProxy.URL() && opts.ProxyURL != d.webviewProxy.SOCKSURL() {
			t.Fatal("host proxy was overridden")
		}
	}
}
