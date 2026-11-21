// Package speedlimit holds the downloader-wide download and upload limits
// shared by every protocol.
package speedlimit

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// minBurst keeps the burst large enough for one BT block or HTTP read even
// when the limit is very low.
const minBurst = 64 << 10

// Limits are the shared limiters. A nil *Limits or limiter means unlimited.
type Limits struct {
	Download *rate.Limiter
	Upload   *rate.Limiter
}

// New returns unlimited limiters.
func New() *Limits {
	return &Limits{Download: rate.NewLimiter(rate.Inf, minBurst), Upload: rate.NewLimiter(rate.Inf, minBurst)}
}

// Set applies bytes-per-second limits; zero or less means unlimited. It takes
// effect for transfers already in progress.
func (l *Limits) Set(download, upload int64) {
	if l == nil {
		return
	}
	set(l.Download, download)
	set(l.Upload, upload)
}

func set(l *rate.Limiter, bytesPerSecond int64) {
	if bytesPerSecond <= 0 {
		l.SetLimit(rate.Inf)
		l.SetBurst(minBurst)
		return
	}
	l.SetLimit(rate.Limit(bytesPerSecond))
	l.SetBurst(int(max(bytesPerSecond, minBurst)))
}

// Wait blocks until n bytes may pass the limiter or ctx is done.
func Wait(ctx context.Context, l *rate.Limiter, n int) error {
	if l == nil {
		return nil
	}
	for n > 0 {
		if l.Limit() == rate.Inf {
			return nil
		}
		step := min(n, l.Burst())
		if err := l.WaitN(ctx, step); err != nil {
			return err
		}
		n -= step
	}
	return nil
}

// Reader limits the bytes read from r. Reads are not cut short: the limiter is
// charged after each read, so a slow limit delays the next read instead of
// the transfer of bytes already received.
func Reader(ctx context.Context, r io.Reader, l *rate.Limiter) io.Reader {
	if l == nil {
		return r
	}
	return &reader{ctx: ctx, r: r, l: l}
}

type reader struct {
	ctx context.Context
	r   io.Reader
	l   *rate.Limiter
}

func (r *reader) Read(p []byte) (int, error) {
	if r.l.Limit() != rate.Inf && len(p) > r.l.Burst() {
		p = p[:r.l.Burst()]
	}
	n, err := r.r.Read(p)
	if n > 0 {
		if werr := Wait(r.ctx, r.l, n); werr != nil && err == nil {
			err = werr
		}
	}
	return n, err
}
