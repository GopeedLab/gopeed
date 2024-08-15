package fetcher

import (
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/pkg/base"
	"path"
	"strings"
)

// Fetcher defines the interface for a download protocol.
// Each download task will have a corresponding Fetcher instance for the management of the download task
type Fetcher interface {
	Setup(ctl *controller.Controller)
	// Resolve resource info from request
	Resolve(req *base.Request) error
	// Create ready to download, but not started
	Create(opts *base.Options) error
	Start() error
	Pause() error
	Close() error

	// Stats refreshes health statistics and returns the latest information
	Stats() any
	// Meta returns the meta information of the download.
	Meta() *FetcherMeta
	// Progress returns the progress of the download.
	Progress() Progress
	// Wait for the download to complete, this method will block until the download is done.
	Wait() error
}

type Uploader interface {
	Upload() error
	UploadedBytes() int64
	WaitUpload() error
}

// FetcherMeta defines the meta information of a fetcher.
type FetcherMeta struct {
	Req  *base.Request  `json:"req"`
	Res  *base.Resource `json:"res"`
	Opts *base.Options  `json:"opts"`
}

// FolderPath return the folder path of the meta info.
func (m *FetcherMeta) FolderPath() string {
	// check if rename folder
	folder := m.Res.Name
	if m.Opts.Name != "" {
		folder = m.Opts.Name
	}
	return path.Join(m.Opts.Path, folder)
}

// SingleFilepath return the single file path of the meta info.
func (m *FetcherMeta) SingleFilepath() string {
	// check if rename file
	file := m.Res.Files[0]
	fileName := file.Name
	if m.Opts.Name != "" {
		fileName = m.Opts.Name
	}
	return path.Join(m.Opts.Path, file.Path, fileName)
}

// RootDirPath return the root dir path of the task file.
func (m *FetcherMeta) RootDirPath() string {
	if m.Res.Name != "" {
		return m.FolderPath()
	} else {
		return m.Opts.Path
	}
}

type FilterType int

const (
	// FilterTypeUrl url type, pattern is the scheme, e.g. http://github.com -> http
	FilterTypeUrl FilterType = iota
	// FilterTypeFile file type, pattern is the file extension name, e.g. test.torrent -> torrent
	FilterTypeFile
	// FilterTypeBase64 base64 data type, pattern is the data mime type, e.g. data:application/x-bittorrent;base64 -> application/x-bittorrent
	FilterTypeBase64
)

type SchemeFilter struct {
	Type    FilterType
	Pattern string
}

func (s *SchemeFilter) Match(uri string) bool {
	uriUpper := strings.ToUpper(uri)
	patternUpper := strings.ToUpper(s.Pattern)
	switch s.Type {
	case FilterTypeUrl:
		return strings.HasPrefix(uriUpper, patternUpper+":")
	case FilterTypeFile:
		return strings.HasSuffix(uriUpper, "."+patternUpper)
	case FilterTypeBase64:
		return strings.HasPrefix(uriUpper, "DATA:"+patternUpper+";BASE64,")
	}
	return false
}

// FetcherManager manage and control the fetcher
type FetcherManager interface {
	// Name return the name of the protocol.
	Name() string
	// Filters registers the supported schemes.
	Filters() []*SchemeFilter
	// Build returns a new fetcher.
	Build() Fetcher
	// ParseName name displayed when the task is not yet resolved, parsed from the request URL
	ParseName(u string) string
	// AutoRename returns whether the fetcher need renaming the download file when has the same name file.
	AutoRename() bool

	// DefaultConfig returns the default configuration of the protocol.
	DefaultConfig() any
	// Store fetcher
	Store(fetcher Fetcher) (any, error)
	// Restore fetcher
	Restore() (v any, f func(meta *FetcherMeta, v any) Fetcher)
	// Close the fetcher manager, release resources.
	Close() error
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
