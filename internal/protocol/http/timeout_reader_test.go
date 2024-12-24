package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestTimeoutReader_Read(t *testing.T) {
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)
	timeoutReader := NewTimeoutReader(reader, 1*time.Second)

	buf := make([]byte, len(data))
	n, err := timeoutReader.Read(buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected to read %d bytes, read %d", len(data), n)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("expected %s, got %s", data, buf)
	}
}

func TestTimeoutReader_ReadTimeout(t *testing.T) {
	reader := &slowReader{delay: 2 * time.Second}
	timeoutReader := NewTimeoutReader(reader, 1*time.Second)

	buf := make([]byte, 8192)
	_, err := timeoutReader.Read(buf)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
	}
}

type slowReader struct {
	delay time.Duration
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(sr.delay)
	return 0, io.EOF
}
