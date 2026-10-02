package ftp

import (
	"crypto/tls"
	"net/url"
	"path"
	"strings"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	pftp "github.com/GopeedLab/gopeed/pkg/protocol/ftp"
)

// fetcherData is what a task saves to resume later. Segments has the same
// Begin, End and Downloaded fields every version writes; the other fields are
// optional, so a save without them still restores.
type fetcherData struct {
	Segments []*segment
	// MaxConns is the login ceiling the server enforced, 0 when none was hit.
	MaxConns int `json:",omitempty"`
	// NoRest records that the server refused REST, so the task downloads each
	// file in one sequential stream.
	NoRest bool `json:",omitempty"`
}

// FetcherManager handles ftp://, ftps:// (implicit TLS) and ftpes:// (explicit
// TLS) URLs.
type FetcherManager struct {
	// baseTLS is the base for every TLS connection. Nil means the system
	// roots; the package's tests set it to trust their own certificate. The
	// task's skipVerifyCert still turns verification off.
	baseTLS *tls.Config
}

func (fm *FetcherManager) Name() string {
	return "ftp"
}

func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{Type: fetcher.FilterTypeUrl, Pattern: "FTP"},
		{Type: fetcher.FilterTypeUrl, Pattern: "FTPS"},
		{Type: fetcher.FilterTypeUrl, Pattern: "FTPES"},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{manager: fm}
}

// ParseName names a task before it is resolved. It never includes the
// userinfo of the URL.
func (fm *FetcherManager) ParseName(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	name := path.Base(strings.TrimSuffix(parsed.Path, "/"))
	if name == "" || name == "/" || name == "." {
		name = parsed.Hostname()
	}
	return name
}

func (fm *FetcherManager) AutoRename() bool {
	return true
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{Connections: defaultConnections}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (any, error) {
	ff := f.(*Fetcher)
	ff.mu.Lock()
	defer ff.mu.Unlock()
	segments := make([]*segment, len(ff.segments))
	for i, s := range ff.segments {
		c := *s
		segments[i] = &c
	}
	return &fetcherData{
		Segments: segments,
		MaxConns: ff.maxConns,
		NoRest:   ff.noRest,
	}, nil
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherData{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		fd := v.(*fetcherData)
		ff := fm.Build().(*Fetcher)
		ff.meta = meta
		if meta != nil && meta.Opts != nil {
			_ = base.ParseOptExtra[pftp.OptsExtra](meta.Opts)
		}
		ff.segments = fd.Segments
		ff.maxConns = fd.MaxConns
		ff.noRest = fd.NoRest
		return ff
	}
}

func (fm *FetcherManager) Close() error {
	return nil
}
