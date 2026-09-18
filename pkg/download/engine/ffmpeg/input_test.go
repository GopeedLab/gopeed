package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
