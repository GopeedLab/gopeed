package http

import (
	"bytes"
	gohttp "net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A restarted download gets a fresh 403 retry budget: the first attempt used
// its retries up and failed, and the second attempt meets a 403 that clears
// well within its own retries.
func TestFetcher_RestartResetsForbiddenRetryBudget(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789abcdef"), 32*1024)
	var recoverAt atomic.Int64
	recoverAt.Store(1<<63 - 1)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if time.Now().UnixNano() < recoverAt.Load() {
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		gohttp.ServeContent(w, r, "slot.bin", time.Time{}, bytes.NewReader(payload))
	}))
	defer server.Close()
	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)))
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("first attempt = %v, want terminal 403", err)
	}
	// The origin refuses for a moment longer, well within the three retries.
	recoverAt.Store(time.Now().Add(200 * time.Millisecond).UnixNano())
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
}
