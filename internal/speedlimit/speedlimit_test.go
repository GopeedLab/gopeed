package speedlimit

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestReaderUnlimitedByDefault(t *testing.T) {
	l := New()
	start := time.Now()
	n, err := io.Copy(io.Discard, Reader(context.Background(), bytes.NewReader(make([]byte, 8<<20)), l.Download))
	if err != nil || n != 8<<20 {
		t.Fatalf("copy: n=%d err=%v", n, err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("unlimited copy took %v", time.Since(start))
	}
}

func TestReaderAppliesLimitAndLiveChange(t *testing.T) {
	l := New()
	l.Set(64<<10, 0)
	// The first burst passes at once; the next 128 KiB need about 2 s.
	start := time.Now()
	n, err := io.Copy(io.Discard, Reader(context.Background(), bytes.NewReader(make([]byte, 192<<10)), l.Download))
	if err != nil || n != 192<<10 {
		t.Fatalf("copy: n=%d err=%v", n, err)
	}
	if d := time.Since(start); d < 1500*time.Millisecond || d > 4*time.Second {
		t.Fatalf("limited copy took %v, want about 2s", d)
	}
	l.Set(0, 0)
	start = time.Now()
	if _, err := io.Copy(io.Discard, Reader(context.Background(), bytes.NewReader(make([]byte, 8<<20)), l.Download)); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("limit removal not applied, took %v", time.Since(start))
	}
}

func TestReaderStopsOnCancel(t *testing.T) {
	l := New()
	l.Set(1, 0)
	ctx, cancel := context.WithCancel(context.Background())
	r := Reader(ctx, bytes.NewReader(make([]byte, 1<<20)), l.Download)
	buf := make([]byte, 1<<20)
	if _, err := r.Read(buf); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := r.Read(buf); err == nil {
		t.Fatal("read after cancel did not fail")
	}
}

func TestNilLimiterIsUnlimited(t *testing.T) {
	if err := Wait(context.Background(), nil, 1<<30); err != nil {
		t.Fatal(err)
	}
	var l *Limits
	l.Set(1, 1)
}
