// Package types holds the shared interfaces and value types for debrid
// services.  It has no dependencies on the service implementations, which
// allows both the parent debrid package and each sub-package (torbox,
// realdebrid, …) to import from here without creating an import cycle.
package types

import "context"

// ServiceName identifies a debrid provider.
type ServiceName string

const (
	ServiceTorBox     ServiceName = "torbox"
	ServiceRealDebrid ServiceName = "realdebrid"
	ServiceNone       ServiceName = ""
)

// File is a single downloadable file resolved by a debrid service.
type File struct {
	Name string
	Size int64
	URL  string // Direct HTTP download URL
}

// Service is the interface every debrid provider must implement.
type Service interface {
	// Name returns the canonical service identifier.
	Name() ServiceName
	// Resolve takes a magnet URI or .torrent URL and returns the list of
	// download-ready HTTP files.  Implementations should poll until the
	// content is cached (up to the supplied context deadline).
	Resolve(ctx context.Context, magnetOrTorrent string) ([]File, error)
}

// Config holds per-service credentials and the user's active selection.
// It is stored inside DownloaderStoreConfig.
type Config struct {
	// Active is the service that will intercept magnet/torrent links.
	// Empty string means debrid is disabled.
	Active        ServiceName `json:"active"`
	TorBoxKey     string      `json:"torBoxKey"`
	RealDebridKey string      `json:"realDebridKey"`
}
