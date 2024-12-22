package http

import (
	"context"
	"io"
	"time"
)

type TimeoutReader struct {
	reader  io.Reader
	timeout time.Duration
}

func NewTimeoutReader(r io.Reader, timeout time.Duration) *TimeoutReader {
	return &TimeoutReader{
		reader:  r,
		timeout: timeout,
	}
}

func (tr *TimeoutReader) Read(p []byte) (n int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), tr.timeout)
	defer cancel()

	done := make(chan struct{})
	var readErr error
	var bytesRead int

	go func() {
		bytesRead, readErr = tr.reader.Read(p)
		close(done)
	}()

	select {
	case <-done:
		return bytesRead, readErr
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
