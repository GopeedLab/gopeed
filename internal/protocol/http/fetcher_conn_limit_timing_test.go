package http

import (
	"bytes"
	"encoding/json"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// oneSlotServer serves payload to one request at a time and answers any other request with
// 403, like a host that limits each client to one connection. hold, when set, runs before the
// accepted request's response headers are written; refuse, when set, runs before a 403 is
// written.
type oneSlotServer struct {
	payload []byte
	mu      sync.Mutex
	active  int
	served  int
	refused int
	hold    func(served int)
	refuse  func(refused int)
}

func (s *oneSlotServer) ServeHTTP(w gohttp.ResponseWriter, r *gohttp.Request) {
	s.mu.Lock()
	if s.active >= 1 {
		s.refused++
		n := s.refused
		s.mu.Unlock()
		if s.refuse != nil {
			s.refuse(n)
		}
		w.WriteHeader(gohttp.StatusForbidden)
		return
	}
	s.active++
	s.served++
	n := s.served
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.active--
		s.mu.Unlock()
	}()
	if s.hold != nil {
		s.hold(n)
	}
	gohttp.ServeContent(w, r, "slot.bin", time.Time{}, bytes.NewReader(s.payload))
}

// restoredTwoChunks returns a fetcher resumed with two unstarted chunks of size/2 each, below
// the size work stealing splits, so each chunk is fetched by its own connection.
func restoredTwoChunks(t *testing.T, url string, size int64) (*Fetcher, string) {
	t.Helper()
	dir := t.TempDir()
	half := size / 2
	conns := []*connection{
		{ID: 0, Role: rolePrimary, State: connNotStarted, Chunk: newChunk(0, half-1)},
		{ID: 1, State: connNotStarted, Chunk: newChunk(half, size-1)},
	}
	rangeMode := true
	saved := &fetcherData{Connections: conns, Range: &rangeMode, ResourceSize: &size, FileSize: &size}
	meta := &fetcher.FetcherMeta{
		Req:  &base.Request{URL: url},
		Res:  &base.Resource{Size: size, Range: true, Files: []*base.FileInfo{{Name: "slot.bin", Size: size}}},
		Opts: &base.Options{Path: dir, Name: "slot.bin", Extra: &fhttp.OptsExtra{Connections: 2}},
	}
	fm := new(FetcherManager)
	_, restore := fm.Restore()
	f := restore(meta, saved).(*Fetcher)
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		if err := json.Unmarshal([]byte(test.ToJson(fm.DefaultConfig())), v); err != nil {
			t.Error(err)
		}
	}
	f.Setup(ctl)
	return f, filepath.Join(dir, "slot.bin")
}

func waitDownload(t *testing.T, f *Fetcher, file string, payload []byte) {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- f.Wait() }()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("download failed: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("download did not finish")
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded content differs from the server's")
	}
}

// A 403 that arrives while the other request holds the server's only slot but has not got
// its response headers yet: the other connection is not downloading yet, but the refused
// chunk must still be retried once the slot is free.
func TestFetcher_ConnectionLimitForbiddenWhilePeerAwaitsHeaders(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789abcdef"), 32*1024)
	refused := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	s := &oneSlotServer{payload: payload}
	s.hold = func(served int) {
		if served == 1 {
			<-release
		}
	}
	s.refuse = func(int) { once.Do(func() { close(refused) }) }
	server := httptest.NewServer(s)
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)))
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-refused:
	case <-time.After(10 * time.Second):
		t.Fatal("the server refused no request")
	}
	// Let the refused connection handle its 403 before the slot frees up.
	time.Sleep(300 * time.Millisecond)
	close(release)
	waitDownload(t, f, file, payload)
}

// A 403 handled after the last downloading connection has finished: the refusal was a
// connection limit when the server sent it, and the chunk must be retried. The HTTP client
// repeats a 403 once at once (browser impersonation); the server is still counting the
// finished connection then, so that answer is 403 too.
func TestFetcher_ConnectionLimitForbiddenAfterLastPeerFinished(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789abcdef"), 32*1024)
	firstDone := make(chan struct{})
	var mu sync.Mutex
	served, refused := 0, 0
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		mu.Lock()
		first := served == 0
		refuse := !first && refused < 2
		if first {
			served++
		} else if refuse {
			refused++
		} else {
			served++
		}
		n := refused
		mu.Unlock()
		if refuse {
			if n == 1 {
				// Answer only once the accepted request is over and the client has seen it end.
				<-firstDone
				time.Sleep(300 * time.Millisecond)
			}
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		gohttp.ServeContent(w, r, "slot.bin", time.Time{}, bytes.NewReader(payload))
		if first {
			close(firstDone)
		}
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)))
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	if refused != 2 {
		t.Fatalf("refused %d requests, want 2", refused)
	}
}
