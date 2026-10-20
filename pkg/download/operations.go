package download

import (
	"context"
	"errors"
)

// beginOperation binds API network work to runtime shutdown, including FFI
// calls that have no HTTP request context.
func (d *Downloader) beginOperation(parent context.Context) (context.Context, func(), error) {
	d.operationGate.Lock()
	defer d.operationGate.Unlock()
	if d.closed.Load() {
		return nil, nil, errors.New("downloader is closed")
	}
	d.operations.Add(1)
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(d.backgroundCtx, cancel)
	return ctx, func() { stop(); cancel(); d.operations.Done() }, nil
}
