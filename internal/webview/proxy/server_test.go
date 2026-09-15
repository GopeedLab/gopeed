package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/armon/go-socks5"
)

func TestRejectedProxyRequests(t *testing.T) {
	secret := "upstream-password"
	for _, tc := range []struct {
		name, method, target string
		status               int
	}{
		{"relative URL", "GET", "/relative", 400},
		{"unsupported URL", "GET", "ftp://example.com/file", 400},
		{"invalid CONNECT", "CONNECT", "example.com", 400},
		{"HTTP selection failure", "GET", "http://example.com/", 502},
		{"CONNECT selection failure", "CONNECT", "example.com:443", 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{selectProxy: func(*http.Request) (*url.URL, error) { return nil, errors.New(secret) }}
			s.transport = &http.Transport{Proxy: s.selectProxy}
			defer s.transport.CloseIdleConnections()
			response := httptest.NewRecorder()
			s.ServeHTTP(response, httptest.NewRequest(tc.method, tc.target, nil))
			if response.Code != tc.status || strings.Contains(response.Body.String(), secret) {
				t.Fatalf("unexpected proxy rejection: %d %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestTunnelRejectsUnsafeUpstreams(t *testing.T) {
	rejected := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "authentication required", http.StatusProxyAuthRequired)
	}))
	defer rejected.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("untrusted TLS proxy received a request")
	}))
	defer secure.Close()
	for _, upstream := range []string{"ftp://127.0.0.1", rejected.URL, secure.URL} {
		t.Run(upstream, func(t *testing.T) {
			selected, err := url.Parse(upstream)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			conn, err := dialTunnel(ctx, "example.com:443", selected)
			if conn != nil {
				conn.Close()
			}
			if err == nil {
				t.Fatal("unsafe upstream unexpectedly accepted")
			}
		})
	}
}

func TestRoutesAndAuthentication(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" {
			t.Error("proxy credentials leaked to origin")
		}
		w.Write([]byte("origin"))
	}))
	defer origin.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("secure")) }))
	defer secure.Close()
	upstream, err := New(func(*http.Request) (*url.URL, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()
	var authenticated atomic.Int32
	authProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "Basic dXNlcjpwYXNz" {
			http.Error(w, "authentication required", 407)
			return
		}
		authenticated.Add(1)
		upstream.ServeHTTP(w, r)
	}))
	defer authProxy.Close()
	socks, err := socks5.New(&socks5.Config{Credentials: socks5.StaticCredentials{"user": "pass"}})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go socks.Serve(listener)
	for _, route := range []string{"", authProxy.URL, "socks5://" + listener.Addr().String()} {
		t.Run(route, func(t *testing.T) {
			var selected *url.URL
			if route != "" {
				selected, _ = url.Parse(route)
				selected.User = url.UserPassword("user", "pass")
			}
			local, err := New(func(r *http.Request) (*url.URL, error) { return selected, nil })
			if err != nil {
				t.Fatal(err)
			}
			defer local.Close()
			proxyURL, _ := url.Parse(local.URL())
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
			for _, test := range []struct{ url, want string }{{origin.URL, "origin"}, {secure.URL, "secure"}} {
				response, err := client.Get(test.url)
				if err != nil {
					t.Fatal(err)
				}
				body, _ := io.ReadAll(response.Body)
				response.Body.Close()
				if string(body) != test.want {
					t.Fatalf("got status %d, body %q", response.StatusCode, body)
				}
			}
		})
	}
	if authenticated.Load() != 2 {
		t.Fatalf("expected authenticated HTTP and CONNECT, got %d", authenticated.Load())
	}
}

func TestRoutingChangesAndFailureDoesNotBypassProxy(t *testing.T) {
	var hits atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.Write([]byte("direct")) }))
	defer origin.Close()
	var useProxy atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("proxied")) }))
	upstreamURL, _ := url.Parse(upstream.URL)
	local, err := New(func(*http.Request) (*url.URL, error) {
		if useProxy.Load() {
			return upstreamURL, nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()
	endpoint, _ := url.Parse(local.URL())
	transport := &http.Transport{Proxy: http.ProxyURL(endpoint)}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	for _, want := range []string{"direct", "proxied", "upstream request failed\n"} {
		resp, err := client.Get(origin.URL)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != want {
			t.Fatalf("got %q want %q", body, want)
		}
		if want == "direct" {
			useProxy.Store(true)
		} else if want == "proxied" {
			upstream.Close()
		}
	}
	if hits.Load() != 1 {
		t.Fatal("failed proxy must not fall back to direct")
	}
}

func TestCloseTerminatesTunnels(t *testing.T) {
	echo, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		c, e := echo.Accept()
		if e == nil {
			defer c.Close()
			io.Copy(c, c)
		}
	}()
	s, err := New(func(*http.Request) (*url.URL, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	endpoint, _ := url.Parse(s.URL())
	c, err := dialTunnel(context.Background(), echo.Addr().String(), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	c.Write([]byte("hello\n"))
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil || line != "hello\n" {
		t.Fatalf("echo %q %v", line, err)
	}
	s.Close()
	if _, err := c.Read(make([]byte, 1)); err == nil {
		t.Fatal("tunnel stayed open")
	}
}

func TestSOCKSIngressPreservesRemoteDNS(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "CONNECT" || r.Host != "unresolved.invalid:80" {
			t.Errorf("lost destination: %s %s", r.Method, r.Host)
		}
		c, b, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer c.Close()
		b.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		b.Flush()
		request, err := http.ReadRequest(b.Reader)
		if err != nil {
			t.Error(err)
			return
		}
		request.Body.Close()
		b.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 7\r\nConnection: close\r\n\r\nproxied")
		b.Flush()
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	local, err := New(func(r *http.Request) (*url.URL, error) {
		if r.URL.Host != "unresolved.invalid:80" {
			t.Errorf("selector lost hostname: %s", r.URL)
		}
		return target, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()
	endpoint, _ := url.Parse(local.SOCKSURL())
	transport := &http.Transport{Proxy: http.ProxyURL(endpoint)}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	response, err := client.Get("http://unresolved.invalid/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if string(body) != "proxied" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestHTTPUpgradeIsForwarded(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "fixture" {
			http.Error(w, "upgrade lost", 400)
			return
		}
		c, b, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer c.Close()
		b.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: fixture\r\n\r\n")
		b.Flush()
		line, err := b.ReadString('\n')
		if err != nil {
			return
		}
		b.WriteString(line)
		b.Flush()
	}))
	defer upstream.Close()
	local, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()
	endpoint, _ := url.Parse(local.URL())
	c, err := net.Dial("tcp", endpoint.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	io.WriteString(c, "GET "+upstream.URL+"/ HTTP/1.1\r\nHost: fixture\r\nConnection: Upgrade\r\nUpgrade: fixture\r\n\r\n")
	reader := bufio.NewReader(c)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 101 {
		t.Fatalf("upgrade failed: %s", response.Status)
	}
	io.WriteString(c, "hello\n")
	line, err := reader.ReadString('\n')
	if err != nil || line != "hello\n" {
		t.Fatalf("upgraded echo %q %v", line, err)
	}
}
