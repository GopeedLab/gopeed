package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/imroc/req/v3"
)

type ImpersonationMode string

const (
	ImpersonationNative ImpersonationMode = "native"
	ImpersonationFixed  ImpersonationMode = "fixed"
	ImpersonationAuto   ImpersonationMode = "auto"
)

type Browser string

const (
	BrowserChrome  Browser = "chrome"
	BrowserFirefox Browser = "firefox"
	BrowserSafari  Browser = "safari"
)

type Options struct {
	Client        ClientOptions
	Transport     TransportOptions
	Impersonation ImpersonationOptions
}

type ClientOptions struct {
	Jar           http.CookieJar
	CheckRedirect func(req *http.Request, via []*http.Request) error
	Timeout       time.Duration
}

type TransportOptions struct {
	Proxy               func(*http.Request) (*url.URL, error)
	DialContext         func(ctx context.Context, network, address string) (net.Conn, error)
	TLSClientConfig     *tls.Config
	TLSHandshakeTimeout time.Duration
}

type ImpersonationOptions struct {
	Mode    ImpersonationMode
	Browser Browser
}

const autoSelectionTTL = 7 * 24 * time.Hour

var autoSelectedBrowsers = newBrowserSelectionCache(autoSelectionTTL, time.Now)

type browserSelection struct {
	browser   Browser
	expiresAt time.Time
}

type browserSelectionCache struct {
	mu      sync.Mutex
	entries map[string]browserSelection
	ttl     time.Duration
	now     func() time.Time
}

func newBrowserSelectionCache(ttl time.Duration, now func() time.Time) *browserSelectionCache {
	return &browserSelectionCache{
		entries: make(map[string]browserSelection),
		ttl:     ttl,
		now:     now,
	}
}

func (c *browserSelectionCache) load(origin string) (Browser, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	selection, ok := c.entries[origin]
	if !ok {
		return "", false
	}
	if !c.now().Before(selection.expiresAt) {
		delete(c.entries, origin)
		return "", false
	}
	return selection.browser, true
}

func (c *browserSelectionCache) store(origin string, browser Browser) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	for cachedOrigin, selection := range c.entries {
		if !now.Before(selection.expiresAt) {
			delete(c.entries, cachedOrigin)
		}
	}
	c.entries[origin] = browserSelection{
		browser:   browser,
		expiresAt: now.Add(c.ttl),
	}
}

func NewClient(options Options) (*http.Client, error) {
	var transport http.RoundTripper
	switch options.Impersonation.Mode {
	case "", ImpersonationNative:
		transport = newNativeTransport(options.Transport)
	case ImpersonationFixed:
		browserTransport, err := newBrowserTransport(options.Transport, options.Impersonation.Browser)
		if err != nil {
			return nil, err
		}
		transport = browserTransport
	case ImpersonationAuto:
		browserTransports := make(map[Browser]http.RoundTripper, 3)
		for _, browser := range []Browser{BrowserChrome, BrowserFirefox, BrowserSafari} {
			browserTransport, err := newBrowserTransport(options.Transport, browser)
			if err != nil {
				return nil, err
			}
			browserTransports[browser] = browserTransport
		}
		transport = &adaptiveTransport{
			native:   newNativeTransport(options.Transport),
			browsers: browserTransports,
			jar:      options.Client.Jar,
		}
	default:
		return nil, fmt.Errorf("unsupported impersonation mode %q", options.Impersonation.Mode)
	}

	return &http.Client{
		Transport:     transport,
		Jar:           options.Client.Jar,
		CheckRedirect: options.Client.CheckRedirect,
		Timeout:       options.Client.Timeout,
	}, nil
}

func newNativeTransport(options TransportOptions) *http.Transport {
	var tlsConfig *tls.Config
	if options.TLSClientConfig != nil {
		tlsConfig = options.TLSClientConfig.Clone()
	}
	return &http.Transport{
		Proxy:               options.Proxy,
		DialContext:         options.DialContext,
		TLSClientConfig:     tlsConfig,
		TLSHandshakeTimeout: options.TLSHandshakeTimeout,
		ForceAttemptHTTP2:   true,
	}
}

func newBrowserTransport(options TransportOptions, browser Browser) (http.RoundTripper, error) {
	client := req.NewClient()
	client.SetProxy(options.Proxy)
	if options.DialContext != nil {
		client.SetDial(options.DialContext)
	}
	client.SetTLSHandshakeTimeout(options.TLSHandshakeTimeout)
	if options.TLSClientConfig != nil {
		tlsConfig := options.TLSClientConfig.Clone()
		if len(tlsConfig.NextProtos) == 0 {
			tlsConfig.NextProtos = []string{"h2", "http/1.1"}
		}
		client.SetTLSClientConfig(tlsConfig)
	}

	switch browser {
	case BrowserChrome:
		client.ImpersonateChrome()
	case BrowserFirefox:
		client.ImpersonateFirefox()
	case BrowserSafari:
		client.ImpersonateSafari()
	default:
		return nil, fmt.Errorf("unsupported browser %q", browser)
	}

	return &defaultHeaderTransport{
		next:     client.GetClient().Transport,
		defaults: client.Headers.Clone(),
	}, nil
}

type defaultHeaderTransport struct {
	next     http.RoundTripper
	defaults http.Header
}

func (t *defaultHeaderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header = request.Header.Clone()
	for key, values := range t.defaults {
		if len(request.Header.Values(key)) == 0 {
			request.Header[key] = append([]string(nil), values...)
		}
	}
	return t.next.RoundTrip(request)
}

func (t *defaultHeaderTransport) CloseIdleConnections() {
	closeIdleConnections(t.next)
}

type adaptiveTransport struct {
	native   http.RoundTripper
	browsers map[Browser]http.RoundTripper
	jar      http.CookieJar
}

func (t *adaptiveTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	origin := request.URL.Scheme + "://" + request.URL.Host
	if selected, ok := autoSelectedBrowsers.load(origin); ok {
		return t.browsers[selected].RoundTrip(request)
	}

	response, err := t.native.RoundTrip(request)
	if !shouldFallback(response, err) {
		return response, err
	}

	retry, ok := cloneRequestForRetry(request)
	if !ok {
		return response, err
	}
	if response != nil {
		applyResponseCookies(t.jar, request, retry, response)
		_ = response.Body.Close()
	}

	browser := browserFromUserAgent(request.UserAgent())
	response, err = t.browsers[browser].RoundTrip(retry)
	if isSuccessfulFallback(response, err) {
		autoSelectedBrowsers.store(origin, browser)
	}
	return response, err
}

func applyResponseCookies(jar http.CookieJar, request, retry *http.Request, response *http.Response) {
	if jar == nil {
		return
	}
	jar.SetCookies(request.URL, response.Cookies())
	jarCookies := jar.Cookies(request.URL)
	jarCookieNames := make(map[string]struct{}, len(jarCookies))
	for _, cookie := range jarCookies {
		jarCookieNames[cookie.Name] = struct{}{}
	}
	existingCookies := retry.Cookies()
	retry.Header.Del("Cookie")
	for _, cookie := range existingCookies {
		if _, replaced := jarCookieNames[cookie.Name]; !replaced {
			retry.AddCookie(cookie)
		}
	}
	for _, cookie := range jarCookies {
		retry.AddCookie(cookie)
	}
}

func (t *adaptiveTransport) CloseIdleConnections() {
	closeIdleConnections(t.native)
	for _, transport := range t.browsers {
		closeIdleConnections(transport)
	}
}

func closeIdleConnections(transport http.RoundTripper) {
	if closer, ok := transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func shouldFallback(response *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if response == nil {
		return false
	}
	return response.StatusCode == http.StatusForbidden ||
		strings.EqualFold(response.Header.Get("Cf-Mitigated"), "challenge")
}

func isSuccessfulFallback(response *http.Response, err error) bool {
	return err == nil && response != nil &&
		response.StatusCode >= http.StatusOK &&
		response.StatusCode < http.StatusBadRequest
}

func cloneRequestForRetry(request *http.Request) (*http.Request, bool) {
	if !isIdempotent(request) {
		return nil, false
	}
	retry := request.Clone(request.Context())
	if request.Body == nil || request.Body == http.NoBody {
		return retry, true
	}
	if request.GetBody == nil {
		return nil, false
	}
	body, err := request.GetBody()
	if err != nil {
		return nil, false
	}
	retry.Body = body
	return retry, true
}

func isIdempotent(request *http.Request) bool {
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return request.Header.Get("Idempotency-Key") != "" ||
		request.Header.Get("X-Idempotency-Key") != ""
}

func browserFromUserAgent(userAgent string) Browser {
	userAgent = strings.ToLower(userAgent)
	switch {
	case strings.Contains(userAgent, "firefox"):
		return BrowserFirefox
	case strings.Contains(userAgent, "safari") &&
		!strings.Contains(userAgent, "chrome") &&
		!strings.Contains(userAgent, "chromium"):
		return BrowserSafari
	default:
		return BrowserChrome
	}
}
