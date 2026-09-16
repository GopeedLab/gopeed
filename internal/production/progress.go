// Package production associates generated resources with their input progress.
package production

import (
	"context"
	"sync"
)

// Source counts unique input bytes and total received bytes (including refetches).
// Implementations must support concurrent snapshots.
type Source interface {
	Progress() (downloaded, received int64)
}

// Reader resolves its optional producer before response headers are sent.
// A nil producer preserves ordinary Blob behavior.
type Reader interface {
	Production() Source
	WaitProduction(context.Context) error
}
type Registry struct {
	mu      sync.Mutex
	sources map[string]Source
}

func (r *Registry) Set(id string, source Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sources == nil {
		r.sources = make(map[string]Source)
	}
	r.sources[id] = source
}
func (r *Registry) Get(id string) Source { r.mu.Lock(); defer r.mu.Unlock(); return r.sources[id] }
func (r *Registry) Remove(id string)     { r.mu.Lock(); defer r.mu.Unlock(); delete(r.sources, id) }
