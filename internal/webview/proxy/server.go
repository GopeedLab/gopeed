// Package proxy provides the host-owned WebView forward proxy. Browser engines
// always use this endpoint; Gopeed decides the upstream route for each request.
package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-socks5"
	xproxy "golang.org/x/net/proxy"
)

type Selector func(*http.Request) (*url.URL, error)

type Server struct {
	listener      net.Listener
	socksListener net.Listener
	server        *http.Server
	transport     *http.Transport
	selectProxy   Selector
	mu            sync.Mutex
	connections   map[io.Closer]struct{}
	closed        bool
}

func New(selectProxy Selector) (*Server, error) {
	if selectProxy == nil {
		selectProxy = func(*http.Request) (*url.URL, error) { return nil, nil }
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{listener: listener, selectProxy: selectProxy, connections: make(map[io.Closer]struct{})}
	s.transport = &http.Transport{Proxy: selectProxy, DisableKeepAlives: true, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 30 * time.Second}
	s.server = &http.Server{Handler: s, ReadHeaderTimeout: 15 * time.Second}
	s.socksListener, err = net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		listener.Close()
		return nil, err
	}
	socks, err := socks5.New(&socks5.Config{Logger: log.New(io.Discard, "", 0), Resolver: unresolvedDNS{}, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		_, port, _ := net.SplitHostPort(address)
		scheme := "https"
		if port == "80" {
			scheme = "http"
		}
		request := &http.Request{URL: &url.URL{Scheme: scheme, Host: address}, Header: make(http.Header)}
		upstream, err := selectProxy(request)
		if err != nil {
			return nil, err
		}
		timed, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		conn, err := dialTunnel(timed, address, upstream)
		if err != nil {
			return nil, err
		}
		return s.track(conn)
	}})
	if err != nil {
		listener.Close()
		s.socksListener.Close()
		return nil, err
	}
	go s.server.Serve(listener)
	go socks.Serve(&trackedListener{Listener: s.socksListener, server: s})
	return s, nil
}

func (s *Server) URL() string { return "http://" + s.listener.Addr().String() }

func (s *Server) SOCKSURL() string { return "socks5://" + s.socksListener.Addr().String() }

func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	connections := make([]io.Closer, 0, len(s.connections))
	for c := range s.connections {
		connections = append(connections, c)
	}
	s.mu.Unlock()
	for _, c := range connections {
		c.Close()
	}
	s.socksListener.Close()
	s.transport.CloseIdleConnections()
	return s.server.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.connect(w, r)
		return
	}
	if !r.URL.IsAbs() || (r.URL.Scheme != "http" && r.URL.Scheme != "https") {
		http.Error(w, "absolute HTTP URL required", http.StatusBadRequest)
		return
	}
	out := r.Clone(r.Context())
	out.RequestURI = ""
	upgrade := out.Header.Get("Upgrade")
	stripHopHeaders(out.Header)
	if upgrade != "" {
		out.Header.Set("Connection", "Upgrade")
		out.Header.Set("Upgrade", upgrade)
	}
	resp, err := s.transport.RoundTrip(out)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSwitchingProtocols {
		remote, ok := resp.Body.(io.ReadWriteCloser)
		if !ok {
			http.Error(w, "invalid upstream upgrade", 502)
			return
		}
		client, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer client.Close()
		resp.Body = nil
		if err := resp.Write(buffer); err != nil {
			return
		}
		if err := buffer.Flush(); err != nil {
			return
		}
		s.bridge(client, buffer, remote)
		return
	}
	stripHopHeaders(resp.Header)
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func stripHopHeaders(h http.Header) {
	for _, key := range strings.Split(h.Get("Connection"), ",") {
		h.Del(strings.TrimSpace(key))
	}
	for _, key := range []string{"Connection", "Proxy-Connection", "Proxy-Authorization", "Proxy-Authenticate", "Keep-Alive", "TE", "Trailer", "Transfer-Encoding", "Upgrade"} {
		h.Del(key)
	}
}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	if _, _, err := net.SplitHostPort(r.Host); err != nil {
		http.Error(w, "invalid tunnel destination", 400)
		return
	}
	target := r.Clone(r.Context())
	target.URL = &url.URL{Scheme: "https", Host: r.Host}
	upstream, err := s.selectProxy(target)
	if err != nil {
		http.Error(w, "proxy selection failed", 502)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	remote, err := dialTunnel(ctx, r.Host, upstream)
	if err != nil {
		http.Error(w, "upstream connection failed", 502)
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
	s.bridge(client, buffer, remote)
}

func (s *Server) bridge(client net.Conn, buffer *bufio.ReadWriter, remote io.ReadWriteCloser) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.connections[client] = struct{}{}
	s.connections[remote] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.connections, client); delete(s.connections, remote); s.mu.Unlock() }()

	done := make(chan struct{})
	go func() { io.Copy(remote, buffer); remote.Close(); close(done) }()
	io.Copy(client, remote)
	client.Close()
	<-done
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.reader.Read(p) }

func dialTunnel(ctx context.Context, target string, upstream *url.URL) (net.Conn, error) {
	dialer := &net.Dialer{}
	if upstream == nil {
		return dialer.DialContext(ctx, "tcp", target)
	}
	if upstream.Scheme == "socks5" || upstream.Scheme == "socks5h" {
		u := *upstream
		u.Scheme = "socks5"
		d, err := xproxy.FromURL(&u, dialer)
		if err != nil {
			return nil, err
		}
		return d.(xproxy.ContextDialer).DialContext(ctx, "tcp", target)
	}
	if upstream.Scheme != "http" && upstream.Scheme != "https" {
		return nil, errors.New("unsupported proxy scheme")
	}
	host := upstream.Host
	if upstream.Port() == "" {
		port := "80"
		if upstream.Scheme == "https" {
			port = "443"
		}
		host = net.JoinHostPort(upstream.Hostname(), port)
	}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			conn.Close()
		}
	}()
	if deadline, exists := ctx.Deadline(); exists {
		conn.SetDeadline(deadline)
	}
	if upstream.Scheme == "https" {
		secure := tls.Client(conn, &tls.Config{ServerName: upstream.Hostname(), MinVersion: tls.VersionTLS12})
		if err := secure.HandshakeContext(ctx); err != nil {
			return nil, err
		}
		conn = secure
	}
	request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: target}, Host: target, Header: make(http.Header)}
	if upstream.User != nil {
		password, _ := upstream.User.Password()
		request.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(upstream.User.Username()+":"+password)))
	}
	if err := request.Write(conn); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		response.Body.Close()
		return nil, fmt.Errorf("upstream CONNECT status %d", response.StatusCode)
	}
	conn.SetDeadline(time.Time{})
	ok = true
	return &bufferedConn{Conn: conn, reader: reader}, nil
}

// Leave hostname resolution to the selected upstream (SOCKS remote DNS / CONNECT).
type unresolvedDNS struct{}

func (unresolvedDNS) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	return ctx, nil, nil
}

type trackedListener struct {
	net.Listener
	server *Server
}

func (l *trackedListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return l.server.track(c)
}

type trackedConn struct {
	net.Conn
	server *Server
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.server.mu.Lock()
	delete(c.server.connections, c)
	c.server.mu.Unlock()
	return err
}
func (s *Server) track(c net.Conn) (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		c.Close()
		return nil, net.ErrClosed
	}
	tracked := &trackedConn{Conn: c, server: s}
	s.connections[tracked] = struct{}{}
	return tracked, nil
}

func (c *trackedConn) CloseWrite() error  { return closeWrite(c.Conn) }
func (c *bufferedConn) CloseWrite() error { return closeWrite(c.Conn) }
func closeWrite(c net.Conn) error {
	if writer, ok := c.(interface{ CloseWrite() error }); ok {
		return writer.CloseWrite()
	}
	return c.Close()
}
