package fetcher

import (
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/pkg/base"
)

// Fetcher defines the interface for a download protocol.
// One fetcher for each download task
type Fetcher interface {
	// Name return the name of the protocol.
	Name() string

	Setup(ctl *controller.Controller) error
	// Resolve resource info from request
	Resolve(req *base.Request) error
	// Create ready to download, but not started
	Create(opts *base.Options) error
	Start() error
	Pause() error
	Continue() error
	Close() error

	// Meta returns the meta information of the download.
	Meta() *FetcherMeta
	// Progress returns the progress of the download.
	Progress() Progress
	// Wait for the download to complete, this method will block until the download is done.
	Wait() error
}

// FetcherMeta defines the meta information of a fetcher.
type FetcherMeta struct {
	Req  *base.Request
	Res  *base.Resource
	Opts *base.Options
}

// FetcherBuilder defines the interface for a fetcher builder.
type FetcherBuilder interface {
	// Schemes returns the schemes supported by the fetcher.
	Schemes() []string
	// Build returns a new fetcher.
	Build() Fetcher

	// Store fetcher
	Store(fetcher Fetcher) (any, error)
	// Restore fetcher
	Restore() (v any, f func(meta *FetcherMeta, v any) Fetcher)
}

type DefaultFetcher struct {
	Ctl    *controller.Controller
	Meta   *FetcherMeta
	DoneCh chan error
}

func (f *DefaultFetcher) Setup(ctl *controller.Controller) (err error) {
	f.Ctl = ctl
	f.DoneCh = make(chan error, 1)
	return
}

func (f *DefaultFetcher) Wait() (err error) {
	return <-f.DoneCh
}

// Progress is a map of the progress of each file in the torrent.
type Progress []int64

// TotalDownloaded returns the total downloaded bytes.
func (p Progress) TotalDownloaded() int64 {
	total := int64(0)
	for _, d := range p {
		total += d
	}
	return total
}
