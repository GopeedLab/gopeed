package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
	"golang.org/x/net/proxy"

	pftp "github.com/GopeedLab/gopeed/pkg/protocol/ftp"
)

const (
	connectTimeout = 15 * time.Second
	readTimeout    = 15 * time.Second
	quitTimeout    = 2 * time.Second
)

// target is a parsed FTP URL. It holds the password, so it must never be
// formatted into an error or a log line.
type target struct {
	host    string // host name or IP, the TLS server name
	addr    string // host:port
	user    string
	pass    string
	path    string // path relative to the login directory, "" for the directory itself
	tlsMode string
}

// parseTarget reads an ftp, ftps or ftpes URL. The path follows RFC 1738: it
// is relative to the login directory, and an encoded leading slash
// (ftp://host/%2Fpub) makes it absolute.
func parseTarget(rawURL, optTLS string) (*target, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		// url.Error repeats the whole URL, password included.
		return nil, errors.New("ftp: invalid URL")
	}
	t := &target{}
	switch strings.ToLower(u.Scheme) {
	case "ftps":
		t.tlsMode = pftp.TLSImplicit
	case "ftpes":
		t.tlsMode = pftp.TLSExplicit
	case "ftp":
		switch strings.ToLower(optTLS) {
		case "", pftp.TLSNone:
			t.tlsMode = pftp.TLSNone
		case pftp.TLSExplicit:
			t.tlsMode = pftp.TLSExplicit
		case pftp.TLSImplicit:
			t.tlsMode = pftp.TLSImplicit
		default:
			return nil, fmt.Errorf("ftp: unknown tls mode %q", optTLS)
		}
	default:
		return nil, fmt.Errorf("ftp: unsupported scheme %q", u.Scheme)
	}
	t.host = u.Hostname()
	if t.host == "" {
		return nil, errors.New("ftp: URL has no host")
	}
	port := u.Port()
	if port == "" {
		port = "21"
		if t.tlsMode == pftp.TLSImplicit {
			port = "990"
		}
	}
	t.addr = net.JoinHostPort(t.host, port)
	if u.User != nil {
		// Username and Password are already percent-decoded.
		t.user = u.User.Username()
		t.pass, _ = u.User.Password()
	}
	if t.user == "" {
		t.user, t.pass = "anonymous", "anonymous@"
	}
	t.path = strings.TrimPrefix(u.Path, "/")
	return t, nil
}

type dialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// session is one logged-in control connection. It owns the network
// connections it opens, so abort can unblock any read or write in progress.
type session struct {
	*ftp.ServerConn

	ctrl     net.Conn // control connection as dialed, under any TLS layer
	dataHost string   // where data connections go

	mu     sync.Mutex
	data   net.Conn // current data connection as dialed
	closed bool
	stop   func() bool
}

// deadline bounds the next control-connection exchange. The control
// connection is idle during a transfer, so every exchange sets a new one.
func (s *session) deadline() {
	_ = s.ctrl.SetDeadline(time.Now().Add(readTimeout))
}

// clearDeadline lets the control connection stay idle through a transfer.
func (s *session) clearDeadline() {
	_ = s.ctrl.SetDeadline(time.Time{})
}

// abort closes every connection at once. Pending calls fail immediately.
func (s *session) abort() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	_ = s.ctrl.Close()
	if s.data != nil {
		_ = s.data.Close()
	}
}

// quit logs out and waits, briefly, for the server to hang up. Servers that
// cap logins count a session until they have seen it end, so the next login
// would otherwise be refused.
func (s *session) quit() {
	if s.stop != nil {
		s.stop()
	}
	_ = s.ctrl.SetDeadline(time.Now().Add(quitTimeout))
	if s.ServerConn != nil {
		_ = s.ServerConn.Quit()
	}
	s.abort()
}

func (s *session) setData(c net.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		_ = c.Close()
		return net.ErrClosed
	}
	s.data = c
	return nil
}

// ctrlConn is the control connection handed to the ftp library.
type ctrlConn struct {
	net.Conn
}

// RemoteAddr must be a *net.TCPAddr: the library asserts it. A connection
// through a proxy may report something else, and its address is not used
// for data connections anyway (see dialData).
func (c ctrlConn) RemoteAddr() net.Addr {
	if a, ok := c.Conn.RemoteAddr().(*net.TCPAddr); ok {
		return a
	}
	return &net.TCPAddr{}
}

// Close is called by the library's Quit right after it sends QUIT. Reading
// until the server hangs up (bounded by the deadline quit set) tells us the
// server has ended the session.
func (c ctrlConn) Close() error {
	_, _ = io.Copy(io.Discard, c.Conn)
	return c.Conn.Close()
}

// dial opens a control connection, upgrades it to TLS when asked, and logs in.
//
// The library is given a dial function for every connection. That is how
// data connections go through the same proxy as the control connection, but
// it also means the library does no TLS on them: with a dial function it
// neither wraps an implicit-TLS control connection nor any data connection.
// We do both here. Explicit TLS (AUTH TLS) is still done by the library, and
// so are PBSZ and PROT P after login. For the same reason the library's
// DialWithTimeout and DialWithContext would have no effect: connectTimeout
// and ctx are applied to our own dials instead.
func (f *Fetcher) dial(ctx context.Context, t *target) (*session, error) {
	netDial, proxied, err := f.netDialer(t)
	if err != nil {
		return nil, err
	}
	var tlsConf *tls.Config
	if t.tlsMode != pftp.TLSNone {
		tlsConf = f.tlsConfig(t.host)
	}

	dialCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	raw, err := netDial(dialCtx, "tcp", t.addr)
	if err != nil {
		return nil, err
	}
	ctrl := raw
	if t.tlsMode == pftp.TLSImplicit {
		tc := tls.Client(raw, tlsConf)
		if err := tc.HandshakeContext(dialCtx); err != nil {
			_ = raw.Close()
			return nil, err
		}
		ctrl = tc
	}

	s := &session{ctrl: raw, dataHost: t.host}
	if a, ok := raw.RemoteAddr().(*net.TCPAddr); ok && !proxied {
		// Same IP as the control connection, as the library itself does,
		// so a name with several addresses cannot send us elsewhere.
		s.dataHost = a.IP.String()
	}
	s.stop = context.AfterFunc(ctx, s.abort)
	s.deadline()

	handedOut := false
	opts := []ftp.DialOption{
		ftp.DialWithShutTimeout(readTimeout),
		ftp.DialWithDialFunc(func(network, addr string) (net.Conn, error) {
			if !handedOut {
				// Dial asks for the control connection first, once.
				handedOut = true
				return ctrlConn{ctrl}, nil
			}
			return s.dialData(ctx, netDial, tlsConf, addr)
		}),
	}
	switch t.tlsMode {
	case pftp.TLSImplicit:
		opts = append(opts, ftp.DialWithTLS(tlsConf))
	case pftp.TLSExplicit:
		opts = append(opts, ftp.DialWithExplicitTLS(tlsConf))
	}
	c, err := ftp.Dial(t.addr, opts...)
	if err != nil {
		s.stop()
		s.abort()
		return nil, err
	}
	s.ServerConn = c
	s.deadline()
	if err := c.Login(t.user, t.pass); err != nil {
		s.stop()
		s.abort()
		return nil, err
	}
	return s, nil
}

// dialData opens a passive data connection. The library takes the host from
// the control connection's peer, which through a proxy is the proxy, so only
// the port is used here.
func (s *session) dialData(ctx context.Context, netDial dialContextFunc, tlsConf *tls.Config, addr string) (net.Conn, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	dialCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	raw, err := netDial(dialCtx, "tcp", net.JoinHostPort(s.dataHost, port))
	if err != nil {
		return nil, err
	}
	if err := s.setData(raw); err != nil {
		return nil, err
	}
	if tlsConf != nil {
		// The handshake runs on the first read, as in the library, because
		// some servers only start it once the transfer command is answered.
		return tls.Client(raw, tlsConf), nil
	}
	return raw, nil
}

// tlsConfig returns the TLS settings for one server. Certificates are
// verified unless the request sets skipVerifyCert, as for HTTP. Data
// connections resume the control connection's TLS session through the shared
// cache, which servers such as vsftpd require by default.
func (f *Fetcher) tlsConfig(host string) *tls.Config {
	cfg := &tls.Config{}
	if f.manager != nil && f.manager.TLSConfig != nil {
		cfg = f.manager.TLSConfig.Clone()
	}
	cfg.ServerName = host
	if f.meta.Req != nil && f.meta.Req.SkipVerifyCert {
		cfg.InsecureSkipVerify = true
	}
	if cfg.ClientSessionCache == nil {
		f.sessionCacheOnce.Do(func() { f.sessionCache = tls.NewLRUClientSessionCache(0) })
		cfg.ClientSessionCache = f.sessionCache
	}
	return cfg
}

// netDialer picks how to reach the server, from the same proxy settings the
// HTTP protocol reads (the task's proxy, otherwise the global one).
func (f *Fetcher) netDialer(t *target) (dialContextFunc, bool, error) {
	direct := (&net.Dialer{Timeout: connectTimeout}).DialContext
	pu, err := f.proxyURL(t)
	if err != nil {
		return nil, false, err
	}
	if pu == nil {
		return direct, false, nil
	}
	switch strings.ToLower(pu.Scheme) {
	case "socks5", "socks5h":
		d, err := proxy.FromURL(pu, &net.Dialer{Timeout: connectTimeout})
		if err != nil {
			return nil, false, fmt.Errorf("ftp: socks5 proxy: %w", err)
		}
		if cd, ok := d.(proxy.ContextDialer); ok {
			return cd.DialContext, true, nil
		}
		return func(_ context.Context, network, addr string) (net.Conn, error) {
			return d.Dial(network, addr)
		}, true, nil
	default:
		// An HTTP proxy is not used. Every passive data connection goes to
		// a port the server picks, and HTTP proxies usually allow CONNECT to
		// port 443 only, so the transfer would fail after the login worked.
		f.warnOnce.Do(func() {
			if l := f.logger(); l != nil {
				l.Warn().Msgf("ftp: the %s proxy at %s cannot carry FTP data connections (HTTP proxy), connecting directly; use a SOCKS5 proxy for FTP", strings.ToLower(pu.Scheme), pu.Host)
			}
		})
		return direct, false, nil
	}
}

func (f *Fetcher) proxyURL(t *target) (*url.URL, error) {
	if f.ctl == nil || f.ctl.GetProxy == nil {
		return nil, nil
	}
	handler := f.ctl.GetProxy(f.meta.Req.Proxy)
	if handler == nil {
		return nil, nil
	}
	// The proxy chooser reads only the URL; the credentials stay out.
	req := &http.Request{URL: &url.URL{Scheme: "ftp", Host: t.addr, Path: "/" + t.path}, Header: http.Header{}}
	pu, err := handler(req)
	if err != nil {
		return nil, errors.New("ftp: cannot read the proxy settings")
	}
	return pu, nil
}

// isLoginRefusal tells a server that is full apart from bad credentials.
// 421 always means "not now". 530 means a refused login; it is only a
// capacity limit when these credentials have worked before. The library
// drops the code when USER itself is refused, so a "too many" message counts
// as well.
func isLoginRefusal(err error, credentialsWorked bool) bool {
	var te *textproto.Error
	if errors.As(err, &te) {
		switch te.Code {
		case ftp.StatusNotAvailable:
			return true
		case ftp.StatusNotLoggedIn:
			return credentialsWorked || strings.Contains(strings.ToLower(te.Msg), "too many")
		}
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "too many")
}

// isPermanent reports errors a retry cannot fix: a 5xx reply and a
// certificate that does not verify.
func isPermanent(err error) bool {
	var te *textproto.Error
	if errors.As(err, &te) {
		return te.Code >= 500
	}
	var verifyErr *tls.CertificateVerificationError
	if errors.As(err, &verifyErr) {
		return true
	}
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		return true
	}
	return errors.Is(err, errSizeChanged)
}

func replyCode(err error) int {
	var te *textproto.Error
	if errors.As(err, &te) {
		return te.Code
	}
	return 0
}
