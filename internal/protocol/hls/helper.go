package hls

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/GopeedLab/gopeed/internal/httpclient"
	"github.com/GopeedLab/gopeed/pkg/base"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

const maxPlaylistSize = 10 << 20 // 10MB, playlists are small text files

func (f *Fetcher) timeout() time.Duration {
	return time.Duration(f.config.TimeoutSeconds) * time.Second
}

// buildClient builds an HTTP client honoring the task proxy and TLS settings.
// The client has no total timeout on purpose: large segments over slow links
// must not be aborted; body stalls are handled by stallReader instead. It is
// only called while the fetcher is quiescent (Resolve/Start), never from
// inside running worker goroutines.
func (f *Fetcher) buildClient() *http.Client {
	var proxy *base.RequestProxy
	skipVerify := false
	if req := f.meta.Req; req != nil {
		proxy = req.Proxy
		skipVerify = req.SkipVerifyCert
	}
	jar, _ := cookiejar.New(nil)
	client, err := httpclient.NewClient(httpclient.Options{
		Client: httpclient.ClientOptions{
			Jar: jar,
		},
		Transport: httpclient.TransportOptions{
			DialContext: (&net.Dialer{
				Timeout: f.timeout(),
			}).DialContext,
			Proxy: f.Ctl.GetProxy(proxy),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipVerify,
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
// agent and an optional byte range. extra may be nil.
func buildRequest(ctx context.Context, extra *ReqExtra, method, u string, body io.Reader, byterange *ByteRange) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if extra != nil {
		for key, value := range extra.Header {
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
