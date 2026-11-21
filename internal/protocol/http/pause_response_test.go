package http

import (
	gohttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestPauseInterruptsResolveResponse(t *testing.T) {
	for _, prefetch := range []bool{false, true} {
		name := "reused response"
		if prefetch {
			name = "prefetch"
		}
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				w.Header().Set("Content-Length", "1048576")
				if prefetch {
					w.Header().Set("Accept-Ranges", "bytes")
				}
				_, _ = w.Write([]byte("prefix"))
				w.(gohttp.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer server.Close()
			f := buildFetcher()
			f.Meta().Opts = &base.Options{Path: t.TempDir(), Name: "paused.bin"}
			if err := f.Resolve(&base.Request{URL: server.URL}); err != nil {
				t.Fatal(err)
			}
			if !prefetch {
				if err := f.Start(); err != nil {
					t.Fatal(err)
				}
			}
			deadline := time.Now().Add(2 * time.Second)
			for {
				f.resolveRespLock.Lock()
				ready := f.resolveResp == nil
				f.resolveRespLock.Unlock()
				if prefetch {
					ready = f.prefetchSize.Load() > 0
				}
				if ready {
					break
				}
				if time.Now().After(deadline) {
					server.CloseClientConnections()
					t.Fatal("response consumer did not start")
				}
				time.Sleep(time.Millisecond)
			}
			done := make(chan error, 1)
			go func() { done <- f.Pause() }()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				server.CloseClientConnections()
				<-done
				t.Fatal("Pause did not cancel the blocked Resolve response")
			}
		})
	}
}
