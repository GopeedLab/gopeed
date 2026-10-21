package download

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// Exercise real Git clone/install/check/update without depending on GitHub or
// public mirror availability. The proxy deliberately rejects the official URL.
// github.com/gopeed-test/repo is a placeholder: the successful mirror serves
// the temporary local repository through git http-backend instead.
func newExtensionGitFixture(t *testing.T) (*Downloader, func(string), *atomic.Int32, *atomic.Int32) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git executable required for HTTP Git fixture")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "repo.git")
	work := t.TempDir()
	command := func(args ...string) {
		t.Helper()
		cmd := exec.Command(git, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	command("-c", "init.defaultBranch=main", "init", "--bare", bare)
	command("clone", bare, work)
	command("-C", work, "config", "user.name", "Gopeed Test")
	command("-C", work, "config", "user.email", "test@example.com")
	publish := func(version string) {
		t.Helper()
		directory := filepath.Join(work, "nested", "extension")
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		manifest := map[string]any{"name": "mirror-test", "author": "test", "title": "Mirror test", "version": version, "repository": map[string]string{"url": "https://github.com/gopeed-test/repo", "directory": "nested/extension"}}
		body, _ := json.Marshal(manifest)
		if err := os.WriteFile(filepath.Join(directory, "manifest.json"), body, 0600); err != nil {
			t.Fatal(err)
		}
		command("-C", work, "add", ".")
		command("-C", work, "commit", "-m", version)
		command("-C", work, "push", "origin", "HEAD:main")
	}
	publish("1.0.0")
	var official, badMirror atomic.Int32
	backend := &cgi.Handler{Path: git, Args: []string{"http-backend"}, Env: []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1"}}
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/https://github.com/gopeed-test")
		r.RequestURI = r.URL.RequestURI()
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(mirror.Close)
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { badMirror.Add(1); http.Error(w, "mirror offline", 502) }))
	t.Cleanup(failed.Close)
	direct := &http.Transport{}
	t.Cleanup(direct.CloseIdleConnections)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect || r.URL.Hostname() == "github.com" {
			official.Add(1)
			http.Error(w, "official unavailable", 502)
			return
		}
		request := r.Clone(r.Context())
		request.RequestURI = ""
		response, err := direct.RoundTrip(request)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		defer response.Body.Close()
		for key, values := range response.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	}))
	t.Cleanup(proxy.Close)
	d := testBackendDownloader(t, &httpProtocol.FetcherManager{})
	config, _ := d.GetConfig()
	config.Proxy = &base.DownloaderProxyConfig{Enable: true, Scheme: "http", Host: strings.TrimPrefix(proxy.URL, "http://")}
	config.GithubMirror = &base.GithubMirrorConfig{Enabled: true, Mirrors: []base.GithubMirror{{Type: "ghProxy", URL: failed.URL}, {Type: "ghProxy", URL: mirror.URL}}}
	if err := d.PutConfig(config); err != nil {
		t.Fatal(err)
	}
	return d, publish, &official, &badMirror
}
func testInstallExtensionGitURL(t *testing.T, raw string) {
	d, _, official, badMirror := newExtensionGitFixture(t)
	extension, err := d.InstallExtensionByGit(raw)
	if err != nil {
		t.Fatal(err)
	}
	if extension.Version != "1.0.0" || extension.Identity != "test@mirror-test" {
		t.Fatalf("%+v", extension)
	}
	if official.Load() == 0 || badMirror.Load() == 0 {
		t.Fatal("ordered fallback was not exercised")
	}
	if _, err := os.Stat(filepath.Join(d.ExtensionPath(extension), "manifest.json")); err != nil {
		t.Fatal(err)
	}
}
func TestExtensionGitMirrorInstallAndUpdate(t *testing.T) {
	d, publish, official, badMirror := newExtensionGitFixture(t)
	extension, err := d.InstallExtensionByGit("https://github.com/gopeed-test/repo#nested/extension")
	if err != nil {
		t.Fatal(err)
	}
	publish("1.1.0")
	version, err := d.UpgradeCheckExtension(extension.Identity)
	if err != nil || version != "1.1.0" {
		t.Fatalf("%s %v", version, err)
	}
	if err = d.UpgradeExtension(extension.Identity); err != nil {
		t.Fatal(err)
	}
	updated, err := d.GetExtension(extension.Identity)
	if err != nil || updated.Version != "1.1.0" {
		t.Fatalf("%+v %v", updated, err)
	}
	if updated.Repository.Url != "https://github.com/gopeed-test/repo" {
		t.Fatal("mirror URL replaced canonical repository")
	}
	if official.Load() != 3 || badMirror.Load() != 3 {
		t.Fatalf("attempts official=%d mirror=%d", official.Load(), badMirror.Load())
	}
}

func TestExtensionGitShutdownWaitsForHandler(t *testing.T) {
	d, _, _, _ := newExtensionGitFixture(t)
	started := make(chan string, 1)
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	finished := make(chan error, 1)
	go func() {
		_, err := d.fetchExtensionByGit("https://github.com/gopeed-test/repo#nested/extension", func(path string, _ bool) (*Extension, error) {
			started <- path
			<-release
			return d.parseExtensionByPath(path)
		})
		finished <- err
	}()
	var path string
	select {
	case path = <-started:
	case err := <-finished:
		t.Fatalf("Git operation finished before handler: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Git handler did not start")
	}
	closed := make(chan error, 1)
	go func() { closed <- d.Close() }()
	select {
	case <-d.backgroundCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not cancel Git context")
	}
	select {
	case err := <-closed:
		t.Fatalf("shutdown returned before handler finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := os.Stat(filepath.Join(path, "manifest.json")); err != nil {
		t.Fatalf("shutdown removed files still used by handler: %v", err)
	}
	if _, err := d.InstallExtensionByGit("https://github.com/gopeed-test/repo"); err == nil || !strings.Contains(err.Error(), "downloader is closed") {
		t.Fatalf("expected new Git operations to be rejected during shutdown: %v", err)
	}
	close(release)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Git handler did not finish")
	}
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not finish after Git handler")
	}
}
