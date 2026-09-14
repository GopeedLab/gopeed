package hls

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/GopeedLab/gopeed/internal/httpclient"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

const maxPlaylistSize = 10 << 20 // 10MB, playlists are small text files

func (f *Fetcher) timeout() time.Duration {
	return time.Duration(f.config.TimeoutSeconds) * time.Second
}

// buildClient builds an HTTP client honoring the task proxy and TLS settings.
// The client has no total timeout on purpose: large segments over slow links
// must not be aborted; body stalls are handled by stallReader instead.
func (f *Fetcher) buildClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	client, err := httpclient.NewClient(httpclient.Options{
		Client: httpclient.ClientOptions{
			Jar: jar,
		},
		Transport: httpclient.TransportOptions{
			DialContext: (&net.Dialer{
				Timeout: f.timeout(),
			}).DialContext,
			Proxy: f.Ctl.GetProxy(f.meta.Req.Proxy),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: f.meta.Req.SkipVerifyCert,
			},
			TLSHandshakeTimeout: f.timeout(),
		},
		Impersonation: httpclient.ImpersonationOptions{
			Mode:    httpclient.ImpersonationAuto,
			Session: f.impSession,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("build HTTP client: %v", err))
	}
	return client
}

// buildRequest builds a request with task extra headers, the default user
// agent and an optional byte range.
func (f *Fetcher) buildRequest(ctx context.Context, method, u string, body io.Reader, byterange *ByteRange) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if f.extra != nil {
		for key, value := range f.extra.Header {
			req.Header.Set(key, value)
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}
	if byterange != nil {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", byterange.Offset, byterange.Offset+byterange.Length-1))
	}
	return req, nil
}

// stallReader aborts reads that see no data for the configured timeout.
type stallReader struct {
	rc      io.ReadCloser
	timeout time.Duration
}

func (r *stallReader) Read(p []byte) (int, error) {
	timer := time.AfterFunc(r.timeout, func() {
		r.rc.Close()
	})
	defer timer.Stop()
	return r.rc.Read(p)
}

func (r *stallReader) Close() error {
	return r.rc.Close()
}

// sha1Hex returns the hex SHA-1 digest of s.
func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
