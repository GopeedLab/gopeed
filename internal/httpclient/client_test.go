package httpclient_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/httpclient"
)

func TestFixedChromeUsesBrowserTLSFingerprintAndHTTP2(t *testing.T) {
	server := newBrowserFingerprintServer(t)

	client, err := httpclient.NewClient(httpclient.Options{
		Transport: httpclient.TransportOptions{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Impersonation: httpclient.ImpersonationOptions{
			Mode:    httpclient.ImpersonationFixed,
			Browser: httpclient.BrowserChrome,
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if response.ProtoMajor != 2 {
		t.Fatalf("protocol = %q, want HTTP/2", response.Proto)
	}
}

func TestAutoFallsBackFromNativeToBrowserFingerprint(t *testing.T) {
	server := newBrowserFingerprintServer(t)

	client, err := httpclient.NewClient(httpclient.Options{
		Transport: httpclient.TransportOptions{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Impersonation: httpclient.ImpersonationOptions{
			Mode: httpclient.ImpersonationAuto,
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()

	if response.ProtoMajor != 2 {
		t.Fatalf("protocol = %q, want HTTP/2", response.Proto)
	}
	if got, want := server.browserHellos(), []bool{false, true}; !equalBools(got, want) {
		t.Fatalf("browser ClientHellos = %v, want %v", got, want)
	}
}

func TestAutoChoosesBrowserFingerprintFromUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		wantBrowser httpclient.Browser
	}{
		{
			name:        "chrome",
			userAgent:   "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36",
			wantBrowser: httpclient.BrowserChrome,
		},
		{
			name:        "firefox",
			userAgent:   "Mozilla/5.0 Firefox/120.0",
			wantBrowser: httpclient.BrowserFirefox,
		},
		{
			name:        "safari",
			userAgent:   "Mozilla/5.0 Version/16.6 Safari/605.1.15",
			wantBrowser: httpclient.BrowserSafari,
		},
		{
			name:        "unknown defaults to chrome",
			userAgent:   "Gopeed/1.0",
			wantBrowser: httpclient.BrowserChrome,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newProfileFingerprintServer(t, test.wantBrowser)
			client, err := httpclient.NewClient(httpclient.Options{
				Transport: httpclient.TransportOptions{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
				Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationAuto},
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			request.Header.Set("User-Agent", test.userAgent)
			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			response.Body.Close()

			if response.ProtoMajor != 2 {
				t.Fatalf("protocol = %q, want HTTP/2", response.Proto)
			}
			if got, want := server.profilesSeen(), []string{"native", string(test.wantBrowser)}; !equalStrings(got, want) {
				t.Fatalf("profiles = %v, want %v", got, want)
			}
		})
	}
}

func TestAutoFallsBackOnForbiddenAndPinsBrowserPerOrigin(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Header.Get("Sec-Ch-Ua") == "" {
			http.Error(w, "browser required", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationAuto},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	for range 2 {
		request, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		request.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36")
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
	}

	if requests.Load() != 3 {
		t.Fatalf("requests = %d, want 3 (native + fallback + pinned)", requests.Load())
	}
}

func TestAutoSharesSelectedBrowserAcrossClientsGlobally(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Header.Get("Sec-Ch-Ua") == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	for range 2 {
		client, err := httpclient.NewClient(httpclient.Options{
			Impersonation: httpclient.ImpersonationOptions{
				Mode: httpclient.ImpersonationAuto,
			},
		})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		request, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		request.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36")
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		response.Body.Close()
	}

	if requests.Load() != 3 {
		t.Fatalf("requests = %d, want 3 (native + fallback + shared pinned)", requests.Load())
	}
}

func TestGlobalAutoSelectionDoesNotAffectNativeMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Sec-Ch-Ua") == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	autoClient, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationAuto},
	})
	if err != nil {
		t.Fatalf("NewClient(auto) error = %v", err)
	}
	autoResponse, err := autoClient.Get(server.URL)
	if err != nil {
		t.Fatalf("auto Get() error = %v", err)
	}
	autoResponse.Body.Close()
	if autoResponse.StatusCode != http.StatusOK {
		t.Fatalf("auto status = %d, want %d", autoResponse.StatusCode, http.StatusOK)
	}

	nativeClient, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationNative},
	})
	if err != nil {
		t.Fatalf("NewClient(native) error = %v", err)
	}
	nativeResponse, err := nativeClient.Get(server.URL)
	if err != nil {
		t.Fatalf("native Get() error = %v", err)
	}
	nativeResponse.Body.Close()
	if nativeResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("native status = %d, want %d", nativeResponse.StatusCode, http.StatusForbidden)
	}
}

func TestAutoAppliesFallbackResponseCookiesBeforeRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Sec-Ch-Ua") == "" {
			http.SetCookie(w, &http.Cookie{Name: "challenge", Value: "passed", Path: "/"})
			w.WriteHeader(http.StatusForbidden)
			return
		}
		cookie, err := request.Cookie("challenge")
		if err != nil || cookie.Value != "passed" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	client, err := httpclient.NewClient(httpclient.Options{
		Client:        httpclient.ClientOptions{Jar: jar},
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationAuto},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func TestAutoDoesNotRetryRequestWithNonReplayableBody(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationAuto},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	bodyReader, bodyWriter := io.Pipe()
	t.Cleanup(func() {
		bodyReader.Close()
		bodyWriter.Close()
	})
	go func() {
		_, _ = bodyWriter.Write([]byte("payload"))
		_ = bodyWriter.Close()
	}()
	request, err := http.NewRequest(http.MethodPost, server.URL, bodyReader)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	response.Body.Close()
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func TestAutoDoesNotRetryReplayablePost(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationAuto},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	response.Body.Close()
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func TestNativeUsesStandardTransportWithHTTP2(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Transport: httpclient.TransportOptions{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Impersonation: httpclient.ImpersonationOptions{
			Mode: httpclient.ImpersonationNative,
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer response.Body.Close()

	if response.ProtoMajor != 2 {
		t.Fatalf("protocol = %q, want HTTP/2", response.Proto)
	}
}

func TestClientOptionsPreserveCookiesAcrossRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/start":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "gopeed", Path: "/"})
			http.Redirect(w, request, "/download", http.StatusFound)
		case "/download":
			cookie, err := request.Cookie("session")
			if err != nil || cookie.Value != "gopeed" {
				http.Error(w, "cookie missing", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, request)
		}
	}))
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	var redirects atomic.Int32
	client, err := httpclient.NewClient(httpclient.Options{
		Client: httpclient.ClientOptions{
			Jar: jar,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				return nil
			},
		},
		Impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationNative},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Get(server.URL + "/start")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if redirects.Load() != 1 {
		t.Fatalf("redirect calls = %d, want 1", redirects.Load())
	}
}

func TestClientTimeoutCancelsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Client: httpclient.ClientOptions{Timeout: 50 * time.Millisecond},
		Impersonation: httpclient.ImpersonationOptions{
			Mode: httpclient.ImpersonationNative,
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Get(server.URL)
	if err == nil {
		t.Fatal("Get() error = nil, want timeout")
	}
	var networkError net.Error
	if !errors.As(err, &networkError) || !networkError.Timeout() {
		t.Fatalf("Get() error = %v, want net.Error timeout", err)
	}
}

func TestTransportOptionsUseProxyInNativeAndFixedModes(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Host != "download.example" {
			http.Error(w, "unexpected target", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(proxy.Close)
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	tests := []struct {
		name          string
		impersonation httpclient.ImpersonationOptions
	}{
		{
			name:          "native",
			impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationNative},
		},
		{
			name: "fixed chrome",
			impersonation: httpclient.ImpersonationOptions{
				Mode:    httpclient.ImpersonationFixed,
				Browser: httpclient.BrowserChrome,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := httpclient.NewClient(httpclient.Options{
				Transport:     httpclient.TransportOptions{Proxy: http.ProxyURL(proxyURL)},
				Impersonation: test.impersonation,
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			response, err := client.Get("http://download.example/file")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestTransportOptionsUseDialContextInNativeAndFixedModes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name          string
		impersonation httpclient.ImpersonationOptions
	}{
		{
			name:          "native",
			impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationNative},
		},
		{
			name: "fixed chrome",
			impersonation: httpclient.ImpersonationOptions{
				Mode:    httpclient.ImpersonationFixed,
				Browser: httpclient.BrowserChrome,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			dialer := &net.Dialer{}
			client, err := httpclient.NewClient(httpclient.Options{
				Transport: httpclient.TransportOptions{
					DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						calls.Add(1)
						return dialer.DialContext(ctx, network, address)
					},
				},
				Impersonation: test.impersonation,
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			response, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			response.Body.Close()
			if calls.Load() == 0 {
				t.Fatal("DialContext() was not called")
			}
		})
	}
}

func TestTransportOptionsApplyTLSHandshakeTimeoutInNativeAndFixedModes(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	closed := make(chan struct{})
	t.Cleanup(func() {
		close(closed)
		listener.Close()
	})
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				<-closed
				connection.Close()
			}()
		}
	}()

	tests := []struct {
		name          string
		impersonation httpclient.ImpersonationOptions
	}{
		{
			name:          "native",
			impersonation: httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationNative},
		},
		{
			name: "fixed chrome",
			impersonation: httpclient.ImpersonationOptions{
				Mode:    httpclient.ImpersonationFixed,
				Browser: httpclient.BrowserChrome,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := httpclient.NewClient(httpclient.Options{
				Client: httpclient.ClientOptions{Timeout: time.Second},
				Transport: httpclient.TransportOptions{
					TLSHandshakeTimeout: 50 * time.Millisecond,
				},
				Impersonation: test.impersonation,
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			started := time.Now()
			_, err = client.Get("https://" + listener.Addr().String())
			if err == nil {
				t.Fatal("Get() error = nil, want TLS handshake timeout")
			}
			if elapsed := time.Since(started); elapsed >= 500*time.Millisecond {
				t.Fatalf("TLS handshake took %s, want less than 500ms", elapsed)
			}
		})
	}
}

func TestFixedBrowserAddsProfileHeadersWithoutOverwritingExplicitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Seen-User-Agent", request.UserAgent())
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{
			Mode:    httpclient.ImpersonationFixed,
			Browser: httpclient.BrowserChrome,
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	response.Body.Close()
	if userAgent := response.Header.Get("Seen-User-Agent"); !strings.Contains(userAgent, "Chrome/120") {
		t.Fatalf("profile User-Agent = %q, want Chrome/120", userAgent)
	}

	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("User-Agent", "Gopeed-Custom/1.0")
	response, err = client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	response.Body.Close()
	if userAgent := response.Header.Get("Seen-User-Agent"); userAgent != "Gopeed-Custom/1.0" {
		t.Fatalf("explicit User-Agent = %q, want Gopeed-Custom/1.0", userAgent)
	}
}

func TestFixedClientCloseIdleConnectionsReachesBrowserTransport(t *testing.T) {
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	client, err := httpclient.NewClient(httpclient.Options{
		Impersonation: httpclient.ImpersonationOptions{
			Mode:    httpclient.ImpersonationFixed,
			Browser: httpclient.BrowserChrome,
		},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	for attempt := range 2 {
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if attempt == 0 {
			client.CloseIdleConnections()
		}
	}

	if connections.Load() != 2 {
		t.Fatalf("connections = %d, want 2", connections.Load())
	}
}

type browserFingerprintServer struct {
	*httptest.Server

	mu     sync.Mutex
	hellos []bool
}

func newBrowserFingerprintServer(t *testing.T) *browserFingerprintServer {
	t.Helper()

	server := &browserFingerprintServer{Server: httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))}
	server.EnableHTTP2 = true
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			browser := false
			for _, cipherSuite := range hello.CipherSuites {
				if isGREASE(cipherSuite) {
					browser = true
					break
				}
			}
			server.mu.Lock()
			server.hellos = append(server.hellos, browser)
			server.mu.Unlock()
			if browser {
				return nil, nil
			}
			return nil, errors.New("browser TLS fingerprint required")
		},
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func (s *browserFingerprintServer) browserHellos() []bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]bool(nil), s.hellos...)
}

func equalBools(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type profileFingerprintServer struct {
	*httptest.Server

	mu       sync.Mutex
	profiles []string
}

func newProfileFingerprintServer(t *testing.T, required httpclient.Browser) *profileFingerprintServer {
	t.Helper()

	server := &profileFingerprintServer{Server: httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))}
	server.EnableHTTP2 = true
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			profile := classifyProfile(hello.CipherSuites)
			server.mu.Lock()
			server.profiles = append(server.profiles, profile)
			server.mu.Unlock()
			if profile == string(required) {
				return nil, nil
			}
			return nil, errors.New("different browser TLS fingerprint required")
		},
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func (s *profileFingerprintServer) profilesSeen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.profiles...)
}

func classifyProfile(cipherSuites []uint16) string {
	if len(cipherSuites) >= 5 && isGREASE(cipherSuites[0]) {
		switch cipherSuites[4] {
		case tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
			return string(httpclient.BrowserChrome)
		case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
			return string(httpclient.BrowserSafari)
		}
	}
	if len(cipherSuites) >= 3 &&
		cipherSuites[0] == tls.TLS_AES_128_GCM_SHA256 &&
		cipherSuites[1] == tls.TLS_CHACHA20_POLY1305_SHA256 &&
		cipherSuites[2] == tls.TLS_AES_256_GCM_SHA384 {
		return string(httpclient.BrowserFirefox)
	}
	return "native"
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isGREASE(value uint16) bool {
	return value&0x0f0f == 0x0a0a && byte(value) == byte(value>>8)
}
