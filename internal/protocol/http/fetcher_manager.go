package http

import (
	"net/url"
	"path"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// ============================================================================
// Fetcher Data (for Store/Restore)
// ============================================================================

type fetcherData struct {
	Connections          []*connection
	RedirectURL          string // Saved redirect URL for resume
	IfRange              string // Strong ETag or Last-Modified validator for safe resume
	RangeReprobeEligible bool   // Origin advertised Range before falling back to sequential mode
	Range                *bool  // Authoritative Range mode; nil for records saved by older versions
}

// ============================================================================
// Fetcher Manager
// ============================================================================

type FetcherManager struct {
}

func (fm *FetcherManager) Name() string {
	return "http"
}

func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: "HTTP",
		},
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: "HTTPS",
		},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{}
}

func (fm *FetcherManager) ParseName(u string) string {
	var name string
	url, err := url.Parse(u)
	if err != nil {
		return ""
	}
	name = path.Base(url.Path)
	if name == "" || name == "/" || name == "." {
		name = url.Hostname()
	}
	return name
}

func (fm *FetcherManager) AutoRename() bool {
	return true
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		Connections: 16,
	}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (data any, err error) {
	_f := f.(*Fetcher)
	_f.redirectLock.Lock()
	redirectURL := _f.redirectURL
	_f.redirectLock.Unlock()

	// Build an immutable snapshot while holding the same lock used by download
	// workers to update Range mode and connection progress. The storage layer
	// marshals this value after Store returns, so returning live pointers would
	// otherwise race with the active download and could persist a mixed state.
	_f.connMu.Lock()
	connections := snapshotConnectionsLocked(_f.connections)
	ifRange := _f.ifRange
	rangeReprobeEligible := _f.rangeReprobeEligible
	var rangeMode *bool
	if _f.meta != nil && _f.meta.Res != nil {
		value := _f.meta.Res.Range
		rangeMode = &value
	}
	_f.connMu.Unlock()

	return &fetcherData{
		Connections:          connections,
		RedirectURL:          redirectURL,
		IfRange:              ifRange,
		RangeReprobeEligible: rangeReprobeEligible,
		Range:                rangeMode,
	}, nil
}

// snapshotConnectionsLocked copies only the serialized connection state and
// deep-copies chunks. The caller must hold the fetcher's connMu.
func snapshotConnectionsLocked(connections []*connection) []*connection {
	snapshot := make([]*connection, len(connections))
	for i, conn := range connections {
		if conn == nil {
			continue
		}
		copyConn := &connection{
			ID:         conn.ID,
			Role:       conn.Role,
			State:      conn.State,
			Downloaded: conn.Downloaded,
			Completed:  conn.Completed,
		}
		if conn.Chunk != nil {
			copyChunk := *conn.Chunk
			copyConn.Chunk = &copyChunk
		}
		snapshot[i] = copyConn
	}
	return snapshot
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherData{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		fd := v.(*fetcherData)
		fb := &FetcherManager{}
		fetcher := fb.Build().(*Fetcher)
		fetcher.meta = meta
		base.ParseReqExtra[fhttp.ReqExtra](fetcher.meta.Req)
		base.ParseOptExtra[fhttp.OptsExtra](fetcher.meta.Opts)
		if len(fd.Connections) > 0 {
			fetcher.connections = fd.Connections
		}
		// Restore redirect URL for resume
		if fd.RedirectURL != "" {
			fetcher.redirectURL = fd.RedirectURL
		}
		fetcher.ifRange = fd.IfRange
		fetcher.rangeReprobeEligible = fd.RangeReprobeEligible
		if fd.Range != nil && fetcher.meta.Res != nil {
			// Range lives in the same persisted snapshot as Connections, making it
			// authoritative over task metadata that may have been saved just before
			// or after this record.
			fetcher.meta.Res.Range = *fd.Range
		}
		return fetcher
	}
}

func (fm *FetcherManager) Close() error {
	return nil
}
