// Package debrid provides a unified interface for debrid services (TorBox,
// Real-Debrid, etc.).  A debrid service caches torrents in the cloud and
// exposes them as fast HTTP download links, bypassing peer-to-peer entirely.
package debrid

import (
	"context"
	"fmt"

	"github.com/GopeedLab/gopeed/internal/debrid/realdebrid"
	"github.com/GopeedLab/gopeed/internal/debrid/torbox"
)

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

// New returns a ready-to-use Service for the given config, or an error if the
// active service is unknown or its API key is missing.
func New(cfg *Config) (Service, error) {
	if cfg == nil || cfg.Active == ServiceNone {
		return nil, fmt.Errorf("debrid: no active service configured")
	}
	switch cfg.Active {
	case ServiceTorBox:
		if cfg.TorBoxKey == "" {
			return nil, fmt.Errorf("debrid: TorBox API key not set")
		}
		return torbox.New(cfg.TorBoxKey), nil
	case ServiceRealDebrid:
		if cfg.RealDebridKey == "" {
			return nil, fmt.Errorf("debrid: Real-Debrid API key not set")
		}
		return realdebrid.New(cfg.RealDebridKey), nil
	default:
		return nil, fmt.Errorf("debrid: unknown service %q", cfg.Active)
	}
}
