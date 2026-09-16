package download

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	webview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

type removalProvider struct {
	pages   map[string][]*removalPage
	removed []string
	failure error
	entered chan struct{}
	resume  chan struct{}
}
type removalPage struct {
	fakeRuntimeWebViewPage
	closed bool
}

func (p *removalPage) Close() error          { p.closed = true; return nil }
func (p *removalProvider) IsAvailable() bool { return true }
func (p *removalProvider) Open(opts webview.OpenOptions) (webview.Page, error) {
	if p.entered != nil {
		close(p.entered)
		<-p.resume
	}
	if p.pages == nil {
		p.pages = make(map[string][]*removalPage)
	}
	page := &removalPage{}
	p.pages[opts.ProfileID] = append(p.pages[opts.ProfileID], page)
	return page, nil
}
func (p *removalProvider) RemoveProfile(id, path string) error {
	for _, page := range p.pages[id] {
		if !page.closed {
			return errors.New("profile removed before closing pages")
		}
	}
	if p.failure != nil {
		return p.failure
	}
	p.removed = append(p.removed, id)
	return nil
}
func removalDownloader(t *testing.T, provider *removalProvider) *Downloader {
	t.Helper()
	storage := NewMemStorage()
	if err := storage.Setup([]string{bucketExtension, bucketExtensionStorage}); err != nil {
		t.Fatal(err)
	}
	d := &Downloader{cfg: &DownloaderConfig{StorageDir: t.TempDir(), DownloaderStoreConfig: &base.DownloaderStoreConfig{}, WebViewProvider: provider}, storage: storage, configLock: &sync.RWMutex{}}
	t.Cleanup(func() {
		if d.webviewProxy != nil {
			d.webviewProxy.Close()
		}
	})
	return d
}
func profileSentinel(t *testing.T, d *Downloader, id string) string {
	t.Helper()
	_, path, err := d.extensionWebViewLocation(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(path, "cookies.sqlite")
	if err := os.WriteFile(sentinel, []byte("private browser state"), 0600); err != nil {
		t.Fatal(err)
	}
	return sentinel
}
func TestDeleteExtensionRemovesWebViewData(t *testing.T) {
	provider := &removalProvider{}
	d := removalDownloader(t, provider)
	ext, err := d.InstallExtensionByFolder("testdata/extensions/basic", true)
	if err != nil {
		t.Fatal(err)
	}
	identity := ext.Identity
	opener := &extensionWebViewOpener{downloader: d, provider: provider, identity: identity, profile: d.extensionWebViewProfile(identity)}
	for i := 0; i < 2; i++ {
		if _, err := opener.Open(webview.OpenOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	sentinel := profileSentinel(t, d, identity)
	other := profileSentinel(t, d, "other-extension")
	// Updating the installation keeps both its profile object and stored data.
	if _, err := d.InstallExtensionByFolder("testdata/extensions/basic", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("upgrade removed browser data", err)
	}
	if err := d.DeleteExtension(identity); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("browser data retained: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal("other extension data was removed", err)
	}
	if _, err := os.Stat(ext.DevPath); err != nil {
		t.Fatal("dev source was removed", err)
	}
	if len(provider.removed) != 1 {
		t.Fatal("native profile was not removed")
	}
	if _, err := opener.Open(webview.OpenOptions{}); err == nil {
		t.Fatal("old runtime recreated removed profile")
	}
	if _, err := d.InstallExtensionByFolder("testdata/extensions/basic", true); err != nil {
		t.Fatal(err)
	}
	fresh := &extensionWebViewOpener{downloader: d, provider: provider, identity: identity, profile: d.extensionWebViewProfile(identity)}
	if _, err := fresh.Open(webview.OpenOptions{}); err != nil {
		t.Fatal("reinstall could not open profile", err)
	}
	if _, err := opener.Open(webview.OpenOptions{}); err == nil {
		t.Fatal("reinstall reactivated old runtime")
	}
}
func TestDeleteExtensionProfileFailureIsRetryable(t *testing.T) {
	provider := &removalProvider{failure: errors.New("native store busy")}
	d := removalDownloader(t, provider)
	ext, err := d.InstallExtensionByFolder("testdata/extensions/basic", true)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := profileSentinel(t, d, ext.Identity)
	if err := d.DeleteExtension(ext.Identity); err == nil {
		t.Fatal("cleanup failure was swallowed")
	}
	if _, err := d.GetExtension(ext.Identity); err != nil {
		t.Fatal("extension removed before cleanup completed")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("profile path removed before native cleanup completed")
	}
	provider.failure = nil
	if err := d.DeleteExtension(ext.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := d.GetExtension(ext.Identity); !errors.Is(err, ErrExtensionNotFound) {
		t.Fatal("extension was not uninstalled")
	}
}
func TestRemoveProfileWaitsForOpeningPage(t *testing.T) {
	provider := &removalProvider{entered: make(chan struct{}), resume: make(chan struct{})}
	d := removalDownloader(t, provider)
	opener := &extensionWebViewOpener{downloader: d, provider: provider, identity: "race", profile: d.extensionWebViewProfile("race")}
	opened := make(chan error, 1)
	go func() { _, err := opener.Open(webview.OpenOptions{}); opened <- err }()
	<-provider.entered
	removed := make(chan error, 1)
	go func() { removed <- d.removeExtensionWebViewProfile("race") }()
	close(provider.resume)
	if err := <-opened; err != nil {
		t.Fatal(err)
	}
	if err := <-removed; err != nil {
		t.Fatal(err)
	}
	if len(provider.removed) != 1 {
		t.Fatal("profile not removed")
	}
}

func TestUninstallWithoutAvailableWebView(t *testing.T) {
	d := removalDownloader(t, &removalProvider{})
	d.cfg.WebViewProvider = webview.NewUnavailableProvider()
	ext, err := d.InstallExtensionByFolder("testdata/extensions/basic", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.DeleteExtension(ext.Identity); err != nil {
		t.Fatal(err)
	}
}
