// Package debrid provides a unified interface for debrid services (TorBox,
// Real-Debrid, etc.).  A debrid service caches torrents in the cloud and
// exposes them as fast HTTP download links, bypassing peer-to-peer entirely.
package debrid

import (
	"fmt"

	"github.com/GopeedLab/gopeed/internal/debrid/realdebrid"
	"github.com/GopeedLab/gopeed/internal/debrid/torbox"
	"github.com/GopeedLab/gopeed/internal/debrid/types"
)

// Re-export shared types so existing callers (e.g. pkg/download) can still
// use debrid.Config, debrid.Service, etc. without a separate import.
type (
	ServiceName = types.ServiceName
	File        = types.File
	Service     = types.Service
	Config      = types.Config
)

const (
	ServiceTorBox     = types.ServiceTorBox
	ServiceRealDebrid = types.ServiceRealDebrid
	ServiceNone       = types.ServiceNone
)

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
