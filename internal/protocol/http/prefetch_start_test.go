package http

import (
	"bytes"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// Start can run after the Resolve prefetch wrote every byte but before it set
// prefetchDone. The prefetched body must still be copied to the target.
func TestFetcher_StartBeforePrefetchDoneCopiesFullBody(t *testing.T) {
	payload := []byte("prefetched before Start")
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		gohttp.ServeContent(w, r, "f.txt", time.Time{}, bytes.NewReader(payload))
	}))
	defer server.Close()

	f := buildFetcher()
	f.Meta().Opts = &base.Options{Path: t.TempDir(), Name: "f.txt", Extra: &http.OptsExtra{Connections: 1}}
	if err := f.Resolve(&base.Request{URL: server.URL + "/f.txt"}); err != nil {
		t.Fatal(err)
	}
	for !f.prefetchDone.Load() {
		time.Sleep(time.Millisecond)
	}
	// Reopen the window between the last write and prefetchDone.
	f.prefetchDone.Store(false)
	go func() {
		time.Sleep(50 * time.Millisecond)
		f.prefetchDone.Store(true)
	}()

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
		t.Fatalf("content = %q, want %q", got, payload)
	}
}
