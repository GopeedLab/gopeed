package ftp

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"

	pftp "github.com/GopeedLab/gopeed/pkg/protocol/ftp"
)

// ServerOptions are the knobs of the in-process FTP test server.
type ServerOptions struct {
	// TLS is pftp.TLSNone (or empty), pftp.TLSExplicit or pftp.TLSImplicit.
	// Explicit mode refuses plain logins, so a passing test proves TLS was used.
	TLS string
	// MaxLogins caps concurrent control connections. Beyond it the server
	// greets with "421 Too many connections" and hangs up. Zero means no cap.
	MaxLogins int
	// NoRest answers "502" to REST. It filters the plain control stream, so it
	// works with TLSNone and TLSImplicit only.
	NoRest bool
	// NoMLST disables MLST and MLSD, so clients fall back to SIZE and LIST.
	NoMLST bool
	// User and Pass are the only accepted credentials. An empty User accepts
	// any login, anonymous included.
	User string
	Pass string
}

// FTPServer is an ftpserverlib server over a temp directory.
type FTPServer struct {
	Addr string // host:port
	Root string // directory served as "/"
	// CA trusts the server's self-signed certificate.
	CA *x509.CertPool

	opts    ServerOptions
	tlsConf *tls.Config
	server  *ftpserver.FtpServer

	// ReadDelay is slept before every read of a transferred file, in
	// nanoseconds, to slow downloads down. Tests may change it at any time.
	ReadDelay atomic.Int64

	// DropAfter, when positive, makes the next transfer fail once after it
	// has sent that many bytes, as if the data connection were killed.
	DropAfter atomic.Int64

	mu      sync.Mutex
	active  int
	retrs   []int64 // offset of every RETR, 0 when no REST came first
	refused int
	logins  int
}

// StartFTPServer starts a server and stops it when the test ends.
func StartFTPServer(t *testing.T, opts ServerOptions) *FTPServer {
	t.Helper()
	s := &FTPServer{Root: t.TempDir(), opts: opts}
	s.tlsConf, s.CA = selfSignedTLS(t)

	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var ln net.Listener = base
	if opts.TLS == pftp.TLSImplicit {
		ln = tls.NewListener(ln, s.tlsConf)
	}
	ln = &limitListener{Listener: ln, srv: s}

	s.Addr = base.Addr().String()
	s.server = ftpserver.NewFtpServer(&mainDriver{srv: s, listener: ln})
	if err := s.server.Listen(); err != nil {
		t.Fatal(err)
	}
	go s.server.Serve()
	t.Cleanup(func() { _ = s.server.Stop() })
	return s
}

// WriteFile creates a file under the served root and returns its SHA-256.
func (s *FTPServer) WriteFile(t *testing.T, rel string, data []byte) string {
	t.Helper()
	name := filepath.Join(s.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return SHA256Hex(data)
}

// RetrOffsets returns the start offset of every RETR the server received.
func (s *FTPServer) RetrOffsets() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.retrs...)
}

// RestOffsets returns the non-zero RETR offsets, which only a REST sets.
func (s *FTPServer) RestOffsets() []int64 {
	var out []int64
	for _, off := range s.RetrOffsets() {
		if off != 0 {
			out = append(out, off)
		}
	}
	return out
}

// Refused returns how many connections were turned away with 421.
func (s *FTPServer) Refused() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refused
}

// Logins returns the number of successful logins.
func (s *FTPServer) Logins() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logins
}

func (s *FTPServer) acquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opts.MaxLogins > 0 && s.active >= s.opts.MaxLogins {
		s.refused++
		return false
	}
	s.active++
	return true
}

func (s *FTPServer) release() {
	s.mu.Lock()
	s.active--
	s.mu.Unlock()
}

// limitListener counts control connections and refuses those beyond
// MaxLogins with a 421 greeting, like vsftpd's max_clients and max_per_ip.
type limitListener struct {
	net.Listener
	srv *FTPServer
}

func (l *limitListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if !l.srv.acquire() {
			go func() {
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				_, _ = io.WriteString(c, "421 Too many connections\r\n")
				_ = c.Close()
			}()
			continue
		}
		var conn net.Conn = &countedConn{Conn: c, release: l.srv.release}
		if l.srv.opts.NoRest {
			conn = &noRestConn{countedConn: conn.(*countedConn), in: bufio.NewReader(c)}
		}
		return conn, nil
	}
}

type countedConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *countedConn) Close() error {
	c.once.Do(c.release)
	return c.Conn.Close()
}

// noRestConn answers REST itself with 502 and hides it from the server.
type noRestConn struct {
	*countedConn
	in      *bufio.Reader
	pending []byte
}

func (c *noRestConn) Read(p []byte) (int, error) {
	for len(c.pending) == 0 {
		line, err := c.in.ReadBytes('\n')
		if len(line) > 0 {
			if strings.HasPrefix(strings.ToUpper(string(line)), "REST ") {
				if _, werr := io.WriteString(c.Conn, "502 REST not implemented\r\n"); werr != nil {
					return 0, werr
				}
			} else {
				c.pending = line
			}
		}
		if err != nil {
			if len(c.pending) > 0 {
				break
			}
			return 0, err
		}
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

type mainDriver struct {
	srv      *FTPServer
	listener net.Listener
}

func (d *mainDriver) GetSettings() (*ftpserver.Settings, error) {
	settings := &ftpserver.Settings{
		Listener:    d.listener,
		DisableMLSD: d.srv.opts.NoMLST,
		DisableMLST: d.srv.opts.NoMLST,
	}
	switch d.srv.opts.TLS {
	case pftp.TLSExplicit:
		settings.TLSRequired = ftpserver.MandatoryEncryption
	case pftp.TLSImplicit:
		settings.TLSRequired = ftpserver.ImplicitEncryption
	}
	return settings, nil
}

func (d *mainDriver) ClientConnected(ftpserver.ClientContext) (string, error) {
	return "gopeed test server", nil
}

func (d *mainDriver) ClientDisconnected(ftpserver.ClientContext) {}

func (d *mainDriver) AuthUser(_ ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	if d.srv.opts.User != "" && (user != d.srv.opts.User || pass != d.srv.opts.Pass) {
		return nil, errors.New("bad credentials")
	}
	d.srv.mu.Lock()
	d.srv.logins++
	d.srv.mu.Unlock()
	return &clientDriver{Fs: afero.NewBasePathFs(afero.NewOsFs(), d.srv.Root), srv: d.srv}, nil
}

func (d *mainDriver) GetTLSConfig() (*tls.Config, error) {
	return d.srv.tlsConf, nil
}

type clientDriver struct {
	afero.Fs
	srv *FTPServer
}

// GetHandle sees the REST offset of every transfer. The server seeks the
// returned file to that offset itself.
func (d *clientDriver) GetHandle(name string, flags int, offset int64) (ftpserver.FileTransfer, error) {
	f, err := d.Fs.OpenFile(name, flags, 0o644)
	if err != nil {
		return nil, err
	}
	if flags == os.O_RDONLY {
		d.srv.mu.Lock()
		d.srv.retrs = append(d.srv.retrs, offset)
		d.srv.mu.Unlock()
	}
	return &slowFile{File: f, srv: d.srv, dropAfter: d.srv.DropAfter.Swap(0)}, nil
}

var errDropped = errors.New("test server dropped the transfer")

type slowFile struct {
	afero.File
	srv       *FTPServer
	sent      int64
	dropAfter int64
}

func (f *slowFile) Read(p []byte) (int, error) {
	if delay := time.Duration(f.srv.ReadDelay.Load()); delay > 0 {
		time.Sleep(delay)
		if len(p) > 16<<10 {
			p = p[:16<<10]
		}
	}
	if f.dropAfter > 0 {
		if f.sent >= f.dropAfter {
			return 0, errDropped
		}
		if rest := f.dropAfter - f.sent; int64(len(p)) > rest {
			p = p[:rest]
		}
	}
	n, err := f.File.Read(p)
	f.sent += int64(n)
	return n, err
}

// selfSignedTLS returns a server config with a fresh certificate for
// 127.0.0.1 and localhost, and a pool that trusts it.
func selfSignedTLS(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "gopeed ftp test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert}},
		MinVersion:   tls.VersionTLS12,
	}, pool
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func FileSHA256(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return SHA256Hex(data)
}

// RandomBytes returns n reproducible pseudo-random bytes.
func RandomBytes(n int, seed byte) []byte {
	out := make([]byte, n)
	x := uint32(seed) + 1
	for i := range out {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		out[i] = byte(x)
	}
	return out
}
