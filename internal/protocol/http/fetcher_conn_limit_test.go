package http

import (
	"bytes"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// TestFetcher_ConnectionLimitForbidden simulates a server that allows only a
// few concurrent connections and answers extra ones with 403. The refused
// chunks must be retried once a slot frees up instead of being dropped.
func TestFetcher_ConnectionLimitForbidden(t *testing.T) {
	const maxActive = 2

	payload := make([]byte, 8*1024*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	var active atomic.Int32
	var refused atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if active.Add(1) > maxActive {
			active.Add(-1)
			refused.Add(1)
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		defer active.Add(-1)
		serveRangeContent(&slowResponseWriter{ResponseWriter: w}, r, payload)
	}))
	defer server.Close()

	downloadDir := t.TempDir()
	f := buildFetcher()
	f.Meta().Opts = &base.Options{
		Path: downloadDir,
		Name: "conn-limit.data",
		Extra: &http.OptsExtra{
			Connections: 8,
		},
	}
	if err := f.Resolve(&base.Request{URL: server.URL + "/conn-limit.data"}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- f.Wait()
	}()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("download failed: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("download did not finish when the server refused extra connections with 403")
	}

	if refused.Load() == 0 {
		t.Fatal("server never refused a connection, test did not exercise the limit")
	}
	got, err := os.ReadFile(filepath.Join(downloadDir, "conn-limit.data"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded file is corrupt: got %d bytes, want %d", len(got), len(payload))
	}
}

// TestFetcher_ForbiddenWithoutActiveConnectionFails simulates a link that
// expires during the download: only the first two requests are served and
// every later one gets 403. Once no connection is receiving data anymore the
// download must fail instead of finishing with missing chunks.
func TestFetcher_ForbiddenWithoutActiveConnectionFails(t *testing.T) {
	payload := make([]byte, 8*1024*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if requests.Add(1) > 2 {
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		serveRangeContent(&slowResponseWriter{ResponseWriter: w}, r, payload)
	}))
	defer server.Close()

	f := buildFetcher()
	f.Meta().Opts = &base.Options{
		Path: t.TempDir(),
		Name: "expired.data",
		Extra: &http.OptsExtra{
			Connections: 8,
		},
	}
	if err := f.Resolve(&base.Request{URL: server.URL + "/expired.data"}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- f.Wait()
	}()
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "403") {
			t.Fatalf("Wait() error = %v, want an error with status 403", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("download did not fail after every connection was refused with 403")
	}
}

// slowResponseWriter throttles writes so connections overlap long enough to hit the limit.
type slowResponseWriter struct {
	gohttp.ResponseWriter
}

func (w *slowResponseWriter) Write(p []byte) (int, error) {
	written := 0
	for written < len(p) {
		end := min(written+32*1024, len(p))
		n, err := w.ResponseWriter.Write(p[written:end])
		written += n
		if err != nil {
			return written, err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return written, nil
}
