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
// timer fires, it closes the underlying reader when it is an io.Closer (such
// as an HTTP response body), which unblocks the pending Read. That Read and
// every later one return context.DeadlineExceeded.
//
// A reader that is not an io.Closer cannot be interrupted; its timed-out Read
// still reports context.DeadlineExceeded, but only once it returns.
//
// TimeoutReader is not safe for concurrent use.
type TimeoutReader struct {
	reader   io.Reader
	timeout  time.Duration
	timer    *time.Timer
	timedOut bool
}

func NewTimeoutReader(r io.Reader, timeout time.Duration) *TimeoutReader {
	return &TimeoutReader{
		reader:  r,
		timeout: timeout,
	}
}

func (tr *TimeoutReader) Read(p []byte) (n int, err error) {
	if tr.timedOut {
		return 0, context.DeadlineExceeded
	}
	if tr.timer == nil {
		tr.timer = time.AfterFunc(tr.timeout, tr.expire)
	} else {
		tr.timer.Reset(tr.timeout)
	}

	n, err = tr.reader.Read(p)

	if !tr.timer.Stop() {
		// The timer fired during this Read and closed the reader, so err is
		// whatever the interrupted read produced. Report the timeout instead.
		tr.timedOut = true
		return n, context.DeadlineExceeded
	}
	return n, err
}

func (tr *TimeoutReader) expire() {
	if c, ok := tr.reader.(io.Closer); ok {
		c.Close()
	}
}
