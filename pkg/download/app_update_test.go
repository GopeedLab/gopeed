package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestAppUpdateAssetName(t *testing.T) {
	for _, test := range []struct{ channel, arch, want string }{
		{"windowsInstaller", "x64", "Gopeed-v2.0.0-beta.2-windows-amd64.zip"},
		{"windowsPortable", "ia32", "Gopeed-v2.0.0-beta.2-windows-386-portable.zip"},
		{"macosDmg", "arm64", "Gopeed-v2.0.0-beta.2-macos-arm64.dmg"},
		{"linuxDeb", "riscv64", "Gopeed-v2.0.0-beta.2-linux-riscv64.deb"},
		{"androidApk", "arm", "Gopeed-v2.0.0-beta.2-android-armeabi-v7a.apk"},
		{"androidApk", "arm64", "Gopeed-v2.0.0-beta.2-android-arm64-v8a.apk"},
		{"androidApk", "x64", "Gopeed-v2.0.0-beta.2-android-x86_64.apk"},
		{"androidApk", "ia32", "Gopeed-v2.0.0-beta.2-android.apk"},
		{"linuxFlathub", "x64", ""}, {"linuxSnap", "x64", ""},
		{"linuxAppImage", "x64", ""}, {"linuxRpm", "x64", ""},
		{"iosIpa", "arm64", ""}, {"docker", "x64", ""}, {"", "", ""},
	} {
		t.Run(test.channel+"/"+test.arch, func(t *testing.T) {
			got, err := appUpdateAssetName("v2.0.0-beta.2", test.channel, test.arch)
			if err != nil || got != test.want {
				t.Fatalf("asset=%q error=%v", got, err)
			}
		})
	}
	for _, test := range []struct{ channel, arch string }{
		{"https://example.com/package.zip", "arm64"}, {"macosDmg", "../arm64"}, {"macosDmg", ""},
	} {
		if _, err := appUpdateAssetName("2.0.0", test.channel, test.arch); err == nil {
			t.Fatalf("invalid target accepted: %+v", test)
		}
	}
}

func TestAppReleaseProviderFallback(t *testing.T) {
	githubFailure := errors.New("GitHub unavailable")
	siteFailure := errors.New("site unavailable")
	for _, test := range []struct {
		name               string
		githubErr, siteErr error
		wantCalls          int
	}{
		{"GitHub success including an empty release list stops fallback", nil, nil, 1},
		{"GitHub failure falls back to the independent official provider", githubFailure, nil, 2},
		{"both provider failures are preserved", githubFailure, siteFailure, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []string
			body, err := fetchAppReleaseData(context.Background(), func(_ context.Context, raw string) ([]byte, error) {
				calls = append(calls, raw)
				if raw == githubReleasesURL {
					return []byte("[]"), test.githubErr
				}
				if raw != gopeedSiteURL+"/api/releases?per_page=10" {
					t.Fatalf("unexpected provider: %s", raw)
				}
				return []byte("[]"), test.siteErr
			})
			if len(calls) != test.wantCalls || (test.siteErr == nil && (err != nil || string(body) != "[]")) {
				t.Fatalf("calls=%v body=%s err=%v", calls, body, err)
			}
			if test.siteErr != nil && (!errors.Is(err, githubFailure) || !errors.Is(err, siteFailure)) {
				t.Fatalf("provider errors lost: %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := fetchAppReleaseData(ctx, func(context.Context, string) ([]byte, error) {
		calls++
		cancel()
		return nil, context.Canceled
	})
	if calls != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request tried fallback: calls=%d err=%v", calls, err)
	}
}

// The local HTTP proxy records real outbound targets without connecting to
// external services. A usable GitHub mirror must avoid the website fallback.
func TestCheckAppUpdateUsesMirrorsBeforeWebsiteFallback(t *testing.T) {
	for _, test := range []struct {
		name         string
		status       int
		body         string
		wantFallback bool
	}{
		{"usable API mirror", http.StatusOK, `[{"tag_name":"v9.0.0"}]`, false},
		{"unsupported API mirror", http.StatusNotFound, "not supported", true},
		{"invalid API mirror response", http.StatusOK, "<html>landing page</html>", true},
		{"non-release JSON array", http.StatusOK, `[{"message":"API unsupported"}]`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var targets []string
			var mirrorPaths []string
			officialStarted := make(chan struct{})
			var officialOnce sync.Once
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodConnect {
					mu.Lock()
					targets = append(targets, r.Host)
					mu.Unlock()
					if r.Host == "api.github.com:443" {
						officialOnce.Do(func() { close(officialStarted) })
					}
					http.Error(w, "provider unavailable", http.StatusBadGateway)
					return
				}
				mu.Lock()
				mirrorPaths = append(mirrorPaths, r.URL.RequestURI())
				mu.Unlock()
				// Keep both official and mirror attempts observable before success
				// cancels the race; all waiting is bound to the request lifetime.
				select {
				case <-officialStarted:
				case <-r.Context().Done():
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			d := testBackendDownloader(t, &httpProtocol.FetcherManager{})
			cfg, _ := d.GetConfig()
			cfg.Proxy = &base.DownloaderProxyConfig{Enable: true, Scheme: "http", Host: strings.TrimPrefix(server.URL, "http://")}
			cfg.GithubMirror = &base.GithubMirrorConfig{Enabled: true, Mirrors: []base.GithubMirror{
				{Type: "jsdelivr", URL: server.URL + "/cdn"},
				{Type: "ghProxy", URL: server.URL + "/proxy"},
			}}
			if err := d.PutConfig(cfg); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			result, err := d.CheckAppUpdate(ctx, AppUpdateRequest{CurrentVersion: "1.2.3", Channel: "windowsPortable", Architecture: "x64"})
			if test.wantFallback {
				if err == nil || result != nil {
					t.Fatalf("failed providers returned update: %+v %v", result, err)
				}
			} else if err != nil || result == nil || result.Version != "9.0.0" {
				t.Fatalf("usable API mirror was not selected: %+v %v", result, err)
			}
			if !test.wantFallback {
				asset := "Gopeed-v9.0.0-windows-amd64-portable.zip"
				official := "https://github.com/GopeedLab/gopeed/releases/download/v9.0.0/" + asset
				if result.AssetName != asset || !reflect.DeepEqual(result.DownloadURLs, []string{official, server.URL + "/proxy/" + official}) {
					t.Fatalf("client package/candidates missing from check: %+v", result)
				}
			}
			wantTargets := []string{"api.github.com:443"}
			if test.wantFallback {
				wantTargets = append(wantTargets, "gopeed.com:443")
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(targets, wantTargets) || !reflect.DeepEqual(mirrorPaths, []string{"/proxy/" + githubReleasesURL}) {
				t.Fatalf("provider targets=%v mirror requests=%v", targets, mirrorPaths)
			}
		})
	}
}
