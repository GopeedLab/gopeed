package http

import (
	"context"
	"io"
	"time"
)

// TimeoutReader fails a Read that makes no progress within timeout.
//
// One timer per reader is armed when Read starts and stopped before it
// returns, so time the caller spends between reads is not counted. If the
// timer fires, it calls cancel, which should cancel the HTTP request the
// reader belongs to so the transport unblocks the pending Read. Read still
// waits for the underlying Read to return; that Read and every later one
// return context.DeadlineExceeded. Errors from any other cancellation are
// returned unchanged, and closing the response body is left to the caller.
//
// With a nil cancel the pending Read cannot be interrupted; it still reports
// context.DeadlineExceeded, but only once it returns.
//
// TimeoutReader is not safe for concurrent use.
type TimeoutReader struct {
	reader   io.Reader
	timeout  time.Duration
	cancel   context.CancelFunc
	timer    *time.Timer
	timedOut bool
}

func NewTimeoutReader(r io.Reader, timeout time.Duration, cancel context.CancelFunc) *TimeoutReader {
	if cancel == nil {
		cancel = func() {}
	}
	return &TimeoutReader{
		reader:  r,
		timeout: timeout,
		cancel:  cancel,
	}
}

func (tr *TimeoutReader) Read(p []byte) (n int, err error) {
	if tr.timedOut {
		return 0, context.DeadlineExceeded
	}
	if tr.timer == nil {
		tr.timer = time.AfterFunc(tr.timeout, tr.cancel)
	} else {
		tr.timer.Reset(tr.timeout)
	}

	n, err = tr.reader.Read(p)

	if !tr.timer.Stop() {
		// The timer fired during this Read and cancelled the request, so err is
		// whatever the interrupted read produced. Report the timeout instead.
		tr.timedOut = true
		return n, context.DeadlineExceeded
	}
	return n, err
}
