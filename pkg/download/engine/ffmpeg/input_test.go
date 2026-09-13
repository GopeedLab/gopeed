package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPRange(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 300000)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requests.Add(1)
		if r.Header.Get("X-Test") != "secret" {
			t.Error("missing headers")
		}
		if r.Header.Get("Accept-Encoding") != "identity" {
			t.Error("missing identity encoding")
		}
		if count > 1 && r.Header.Get("If-Range") != `"v1"` {
			t.Error("missing If-Range")
		}
		w.Header().Set("ETag", `"v1"`)
		http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()
	in, err := OpenHTTP(context.Background(), srv.Client(), HTTPSource{URL: srv.URL, Headers: map[string]string{"X-Test": "secret"}})
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	if !in.Seekable() || in.Size() != int64(len(data)) {
		t.Fatal("range not detected")
	}
	for _, off := range []int64{0, int64(len(data) - 16), 12, int64(httpBlockSize + 35)} {
		if _, err = in.Seek(off, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		b := make([]byte, 16)
		if _, err = io.ReadFull(in, b); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, data[off:off+16]) {
			t.Fatalf("wrong bytes at %d", off)
		}
	}
}

func TestHTTPSequentialAndValidation(t *testing.T) {
	for _, kind := range []string{"sequential", "bad-range", "changed", "truncated", "unauthorized"} {
		t.Run(kind, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count := requests.Add(1)
				switch kind {
				case "sequential":
					w.Write([]byte("media"))
				case "unauthorized":
					w.WriteHeader(403)
				case "bad-range":
					w.Header().Set("Content-Range", "bytes 2-6/10")
					w.WriteHeader(206)
					w.Write([]byte("media"))
				case "truncated":
					w.Header().Set("Content-Range", "bytes 0-9/10")
					w.WriteHeader(206)
					w.Write([]byte("media"))
				case "changed":
					if count == 1 {
						w.Header().Set("Content-Range", "bytes 0-4/10")
						w.Header().Set("ETag", `"v1"`)
						w.WriteHeader(206)
					}
					w.Write([]byte("media"))
				}
			}))
			defer srv.Close()
			in, err := OpenHTTP(context.Background(), srv.Client(), HTTPSource{URL: srv.URL})
			if kind == "sequential" {
				if err != nil {
					t.Fatal(err)
				}
				defer in.Close()
				if in.Seekable() {
					t.Fatal("unexpected seeking")
				}
				if _, err = in.Seek(0, 0); err != ErrSeek {
					t.Fatal(err)
				}
				b, err := io.ReadAll(in)
				if err != nil || string(b) != "media" || requests.Load() != 1 {
					t.Fatalf("%q %v count %d", b, err, requests.Load())
				}
			} else if kind == "changed" {
				if err != nil {
					t.Fatal(err)
				}
				defer in.Close()
				in.Seek(5, 0)
				if _, err = in.Read(make([]byte, 2)); err == nil {
					t.Fatal("accepted 200 after range")
				}
			} else if err == nil {
				in.Close()
				t.Fatal("accepted invalid response")
			}
		})
	}
}

func TestStreamRing(t *testing.T) {
	s := NewStreamInput(context.Background())
	defer s.Close()
	// Start near the boundary to exercise wrap-around without a huge fixture.
	s.read = StreamBufferLimit - 3
	s.written = s.read
	if _, err := s.Write([]byte("abcdefgh")); err != nil {
		t.Fatal(err)
	}
	s.End(nil)
	b, err := io.ReadAll(s)
	if err != nil || string(b) != "abcdefgh" {
		t.Fatalf("%q %v", b, err)
	}
	name := s.file.Name()
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(name); err == nil {
		t.Fatal("temporary file leaked")
	}
}

func TestOutputArguments(t *testing.T) {
	a, err := Arguments("", []string{"-shortest", "-metadata", "title=test"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(a, " "), "-f mp4 pipe:1") {
		t.Fatal(a)
	}
	for _, args := range [][]string{{"-i", "/etc/passwd"}, {"-report"}, {"-c:v", "libx264"}, {"-metadata"}, {"other.mp4"}} {
		if _, err := Arguments("mp4", args); err == nil {
			t.Fatal(fmt.Sprint(args))
		}
	}
}

func TestStreamBackpressureAndCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewStreamInput(ctx)
	defer s.Close()
	f, err := os.CreateTemp(t.TempDir(), "ring")
	if err != nil {
		t.Fatal(err)
	}
	s.file = f
	if err = f.Truncate(StreamBufferLimit); err != nil {
		t.Fatal(err)
	}
	s.written = StreamBufferLimit
	finished := make(chan error, 1)
	go func() { _, err := s.Write([]byte{42}); finished <- err }()
	select {
	case err := <-finished:
		t.Fatalf("did not apply backpressure: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	if _, err = s.Read(make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("writer did not resume")
	}
	go func() { _, err := s.Write([]byte{43}); finished <- err }()
	cancel()
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("cancelled write succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("writer did not cancel")
	}
}
