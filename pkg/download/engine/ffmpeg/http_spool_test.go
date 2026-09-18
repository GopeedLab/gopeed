package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPSpoolRangeCrossBlocksAndRefetch(t *testing.T) {
	data := make([]byte, 4*SpoolBlockSize+57)
	for i := range data {
		data[i] = byte(i*31 + i/257)
	}
	var large atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var start, end int64
		fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end)
		if end-start+1 > SpoolBlockSize {
			large.Store(true)
		}
		w.Header().Set("ETag", `"stable"`)
		http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()
	input, err := OpenSpoolingHTTP(context.Background(), srv.Client(), HTTPSource{URL: srv.URL}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	h := input.(*HTTPSpool)
	select {
	case <-h.done:
	case <-time.After(10 * time.Second):
		t.Fatal("prefetch did not finish")
	}
	if !large.Load() {
		t.Fatal("did not fetch multiple blocks in one response")
	}
	for _, offset := range []int64{3*SpoolBlockSize - 123, 0, 2*SpoolBlockSize + 41, SpoolBlockSize - 5} {
		if _, err := h.Seek(offset, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		got := make([]byte, min(int64(2*SpoolBlockSize), int64(len(data))-offset))
		if _, err := io.ReadFull(h, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data[offset:offset+int64(len(got))]) {
			t.Fatal("incorrect random/cross-block read", offset)
		}
	}
	unique, received := h.Progress()
	if unique != int64(len(data)) || received <= unique {
		t.Fatal("refetch progress must not double-count stored bytes", unique, received)
	}
}

func TestHTTPSpoolSequentialAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		close(entered)
		<-r.Context().Done()
	}))
	defer srv.Close()
	in, err := OpenSpoolingHTTP(ctx, srv.Client(), HTTPSource{URL: srv.URL}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	if in.Seekable() {
		t.Fatal("200 response advertised seek")
	}
	got := make([]byte, 5)
	if _, err := io.ReadFull(in, got); err != nil || string(got) != "hello" {
		t.Fatal(string(got), err)
	}
	cancel()
	if err := in.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPSpoolRejectsBadRanges(t *testing.T) {
	for _, mode := range []string{"etag", "length", "offset", "status", "truncated", "encoding"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				first := calls.Add(1) == 1
				start, end := int64(0), int64(httpBlockSize-1)
				if !first {
					fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end)
				}
				tag := `"stable"`
				if !first && mode == "etag" {
					tag = `"changed"`
				}
				w.Header().Set("ETag", tag)
				if !first && mode == "status" {
					w.WriteHeader(200)
					return
				}
				if !first && mode == "offset" {
					start++
				}
				if !first && mode == "encoding" {
					w.Header().Set("Content-Encoding", "gzip")
				}
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, 2*SpoolBlockSize))
				length := end - start + 1
				if !first && mode == "length" {
					length++
				}
				w.Header().Set("Content-Length", fmt.Sprint(length))
				w.WriteHeader(206)
				if !first && mode == "truncated" {
					length /= 2
				}
				chunk := make([]byte, 64*1024)
				for length > 0 {
					n := min(int64(len(chunk)), length)
					if _, err := w.Write(chunk[:n]); err != nil {
						return
					}
					length -= n
				}
			}))
			defer srv.Close()
			in, err := OpenSpoolingHTTP(context.Background(), srv.Client(), HTTPSource{URL: srv.URL}, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()
			h := in.(*HTTPSpool)
			select {
			case err := <-h.Failures():
				if err == nil {
					t.Fatal("nil failure")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("bad range did not fail")
			}
		})
	}
}

func TestHTTPSpoolReadsPartialRangeBeforeResponseFinishes(t *testing.T) {
	ready, finish := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var start, end int64
		fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end)
		end = min(end, 4*1024*1024-1)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, 4*1024*1024))
		w.Header().Set("Content-Length", fmt.Sprint(end-start+1))
		w.WriteHeader(206)
		if start == 0 {
			w.Write(make([]byte, end+1))
			return
		}
		w.Write(bytes.Repeat([]byte{93}, 64*1024))
		w.(http.Flusher).Flush()
		close(ready)
		select {
		case <-finish:
			w.Write(make([]byte, end-start+1-64*1024))
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	input, err := OpenSpoolingHTTP(ctx, srv.Client(), HTTPSource{URL: srv.URL}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := input.Seek(1024*1024, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 1)
	if _, err := io.ReadFull(input, b); err != nil || b[0] != 93 {
		t.Fatal("waited for complete response", b, err)
	}
	close(finish)
}
