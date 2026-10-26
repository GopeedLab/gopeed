package download

import (
	"context"
	"net"
	"net/http"
	"time"

	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

func init() {
	// go-git exposes a process-wide protocol registry. Register once during
	// initialization, rather than changing it during concurrent clones. Proxy
	// settings still come from each clone's ProxyOptions.
	transport := githttp.NewClient(newExtensionGitHTTPClient(10*time.Second, 60*time.Second))
	gitclient.InstallProtocol("http", transport)
	gitclient.InstallProtocol("https", transport)
}

func newExtensionGitHTTPClient(connectTimeout, idleTimeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: connectTimeout, KeepAlive: 30 * time.Second}
	// A socket read deadline must apply to one transfer, rather than a shared
	// HTTP/2 connection carrying traffic from other requests.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	return &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return &gitIdleConn{Conn: conn, timeout: idleTimeout}, nil
		},
		TLSHandshakeTimeout: connectTimeout,
		IdleConnTimeout:     idleTimeout,
		Protocols:           protocols,
	}}
}

type gitIdleConn struct {
	net.Conn
	timeout time.Duration
}

func (c *gitIdleConn) Read(p []byte) (int, error) {
	if err := c.Conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Read(p)
}

// Refresh an already-pending read when a pooled HTTP/1 connection starts the
// next request; time spent idle in the pool must not consume its read allowance.
func (c *gitIdleConn) Write(p []byte) (int, error) {
	if err := c.Conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}
