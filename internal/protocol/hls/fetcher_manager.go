package hls

import (
	"net/url"
	"path"
	"strings"

	"github.com/GopeedLab/gopeed/internal/fetcher"
)

type FetcherManager struct {
}

func (fm *FetcherManager) Name() string {
	return "hls"
}

// Filters registers the .m3u8 extension. The manager must be registered BEFORE
// the http manager in the downloader config: protocol routing picks the first
// manager whose filter matches, and the http filter claims every http(s) URL.
func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{
			Type:    fetcher.FilterTypeFile,
			Pattern: "M3U8",
		},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{}
}

func (fm *FetcherManager) ParseName(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	name := path.Base(parsed.Path)
	name = strings.TrimSuffix(name, path.Ext(name))
	if name == "" || name == "/" || name == "." {
		name = parsed.Hostname()
	}
	return name
}

func (fm *FetcherManager) AutoRename() bool {
	return true
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{
		SegmentConnections:    defaultSegmentConnections,
		MaxRetries:            defaultMaxRetries,
		TimeoutSeconds:        defaultTimeoutSeconds,
		PrefetchContentLength: true,
	}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (data any, err error) {
	_f := f.(*Fetcher)
	_f.mu.Lock()
	defer _f.mu.Unlock()
	if _f.state == nil {
		return nil, nil
	}
	// The engine's periodic checkpoint calls Store without holding the task
	// lock and serializes the returned value afterwards, while Resolve/Patch/
	// Start mutate the live state under the fetcher mutex: return a snapshot
	// copy taken under that mutex, mirroring the HTTP protocol's Store.
	return cloneState(_f.state), nil
}

func cloneState(state *fetcherState) *fetcherState {
	cloned := *state
	cloned.Segments = make([]*Segment, len(state.Segments))
	for i, seg := range state.Segments {
		s := *seg
		cloned.Segments[i] = &s
	}
	return &cloned
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherState{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		fb := &FetcherManager{}
		restored := fb.Build().(*Fetcher)
		restored.meta = meta
		if state, ok := v.(*fetcherState); ok && state.MediaURL != "" {
			restored.state = state
		}
		// Setup is invoked by the engine (restoreFetcher) after this builder
		// returns; ReqExtra parsing happens there as well.
		return restored
	}
}

func (fm *FetcherManager) Close() error {
	return nil
}
