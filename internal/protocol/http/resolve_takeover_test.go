package http

import (
	"bytes"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// A server that limits connections refuses the first Range request with 429
// while Resolve keeps downloading. The retried request must not download the
// prefix Resolve already has.
func TestFetcher_RetriedTakeoverDoesNotRefetchPrefix(t *testing.T) {
	tests := []struct {
		name string
		// maxActive is the number of responses served at once.
		maxActive int64
		// busyFor refuses Range requests for this long, as if other tasks
		// held the server's connection slots.
		busyFor time.Duration
	}{
		{name: "resolve holds the only slot", maxActive: 1},
		{name: "slot frees mid-prefetch", maxActive: 3, busyFor: time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := bytes.Repeat([]byte("0123456789abcdef"), 256*1024) // 4 MiB, sent in about 2 s
			var active, served atomic.Int64
			busyUntil := time.Now().Add(tt.busyFor)
			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				if r.Header.Get(base.HttpHeaderRange) != "" && time.Now().Before(busyUntil) {
					w.WriteHeader(gohttp.StatusTooManyRequests)
					return
				}
				if active.Add(1) > tt.maxActive {
					active.Add(-1)
					w.WriteHeader(gohttp.StatusTooManyRequests)
					return
				}
				defer active.Add(-1)
				gohttp.ServeContent(&throttledWriter{ResponseWriter: w, served: &served}, r, "f.bin", time.Unix(0, 0), bytes.NewReader(payload))
			}))
			defer server.Close()

			f := buildFetcher()
			f.Meta().Opts = &base.Options{
				Path:  t.TempDir(),
				Name:  "f.bin",
				Extra: &http.OptsExtra{Connections: 4},
			}
			if err := f.Resolve(&base.Request{URL: server.URL + "/f.bin"}); err != nil {
				t.Fatal(err)
			}
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if err := f.Wait(); err != nil {
				t.Fatal(err)
			}

			got, err := os.ReadFile(f.meta.SingleFilepath())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatal("downloaded content does not match")
			}
			if ratio := float64(served.Load()) / float64(len(payload)); ratio > 1.1 {
				t.Fatalf("server sent %.2fx the file size, want at most 1.1x", ratio)
			}
		})
	}
}

// throttledWriter sends 16 KiB every 8 ms, about 2 MiB/s, and counts the bytes.
type throttledWriter struct {
	gohttp.ResponseWriter
	served *atomic.Int64
}

func (w *throttledWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		n := min(len(p), 16*1024)
		time.Sleep(8 * time.Millisecond)
		m, err := w.ResponseWriter.Write(p[:n])
		total += m
		w.served.Add(int64(m))
		if err != nil {
			return total, err
		}
		p = p[n:]
	}
	return total, nil
}
