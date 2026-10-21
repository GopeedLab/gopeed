package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/internal/githubmirror"
	"github.com/GopeedLab/gopeed/internal/httpclient"
)

func (d *Downloader) resourceClient(publicOnly bool) (*http.Client, error) {
	cfg, err := d.GetConfig()
	if err != nil {
		return nil, err
	}
	options := httpclient.Options{
		Client:    httpclient.ClientOptions{Timeout: 30 * time.Second},
		Transport: httpclient.TransportOptions{Proxy: cfg.Proxy.ToHandler(), TLSHandshakeTimeout: 10 * time.Second},
	}
	if publicOnly {
		// A configured upstream proxy is trusted, but target URLs are still checked.
		options.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return validatePublicURL(req.Context(), req.URL.String())
		}
		if cfg.Proxy == nil || !cfg.Proxy.Enable {
			options.Transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}
				ips, err := publicIPs(ctx, host)
				if err != nil {
					return nil, err
				}
				var last error
				for _, ip := range ips {
					conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
					if err == nil {
						return conn, nil
					}
					last = err
				}
				return nil, last
			}
		}
	}
	return httpclient.NewClient(options)
}
func publicIPs(ctx context.Context, host string) ([]net.IPAddr, error) {
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("target has no addresses")
	}
	for _, ip := range ips {
		if !ip.IP.IsGlobalUnicast() || ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsUnspecified() {
			return nil, errors.New("private or local resource target is not allowed")
		}
	}
	return ips, nil
}
func validatePublicURL(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return errors.New("invalid resource URL")
	}
	_, err = publicIPs(ctx, u.Hostname())
	return err
}
func (d *Downloader) fetchPublicResource(ctx context.Context, raw string, limit int64, validate func([]byte) error) ([]byte, error) {
	return d.fetchResource(ctx, raw, limit, true, validate)
}

// fetchResource races complete, validated responses. A successful HEAD alone
// cannot win, and failed candidates never discard a usable official response.
func (d *Downloader) fetchResource(ctx context.Context, raw string, limit int64, publicOnly bool, validate func([]byte) error) ([]byte, error) {
	cfg, err := d.GetConfig()
	if err != nil {
		return nil, err
	}
	client, err := d.resourceClient(publicOnly)
	if err != nil {
		return nil, err
	}
	defer client.CloseIdleConnections()
	return githubmirror.Race(ctx, cfg.GithubMirror, raw,
		githubmirror.Options[[]byte]{Kind: githubmirror.Resource, Timeout: 30 * time.Second},
		func(ctx context.Context, candidate string) ([]byte, error) {
			if publicOnly {
				if err := validatePublicURL(ctx, candidate); err != nil {
					return nil, err
				}
			}
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
			if err != nil {
				return nil, err
			}
			response, err := client.Do(request)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("resource returned HTTP %d", response.StatusCode)
			}
			body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
			if err != nil {
				return nil, err
			}
			if int64(len(body)) > limit {
				return nil, errors.New("resource exceeds size limit")
			}
			if len(body) == 0 {
				return nil, errors.New("empty resource")
			}
			if validate != nil {
				if err := validate(body); err != nil {
					return nil, err
				}
			}
			return body, nil
		},
	)
}

// githubResourceCandidates accepts official GitHub HTTPS resource URLs only.
func githubResourceCandidates(config *githubmirror.Config, raw string) ([]string, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" || strings.Contains(raw, "#") {
		return nil, ErrInvalidMirrorURL
	}
	u.Host = strings.ToLower(u.Host)
	switch u.Host {
	case "github.com", "raw.githubusercontent.com":
	default:
		return nil, ErrInvalidMirrorURL
	}
	return githubmirror.Candidates(config, u.String(), githubmirror.Resource), nil
}
