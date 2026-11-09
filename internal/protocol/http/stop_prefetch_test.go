package http

import (
	"bytes"
	"fmt"
	gohttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// A stalled Resolve body must not keep the prefetch goroutine alive after
// stopPrefetchAndCopyData returns, or it writes to a file we already removed.
func TestFetcher_StopPrefetchUnblocksStalledRead(t *testing.T) {
	prefix := bytes.Repeat([]byte("p"), 64*1024)
	stall := make(chan struct{})
	defer close(stall)

	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(prefix)*4))
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(prefix)
		w.(gohttp.Flusher).Flush()
		select {
		case <-stall:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()

	f := buildFetcher()
	f.Meta().Opts = &base.Options{
		Path:  t.TempDir(),
		Name:  "stalled.data",
		Extra: &http.OptsExtra{Connections: 1},
	}
	if err := f.Resolve(&base.Request{URL: server.URL + "/stalled.data"}); err != nil {
		t.Fatal(err)
	}
	for f.prefetchSize.Load() < int64(len(prefix)) {
		time.Sleep(time.Millisecond)
	}

	start := time.Now()
	f.stopPrefetch()
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("stopPrefetch took %v, want it to interrupt the stalled read", elapsed)
	}
	if !f.prefetchDone.Load() {
		t.Fatal("prefetch goroutine still running after stopPrefetch")
	}
}
