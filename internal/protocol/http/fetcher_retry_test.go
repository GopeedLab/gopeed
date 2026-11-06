package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// fastRetryUnit keeps the retry delays of a test short: 1 to 5 of these.
const fastRetryUnit = 5 * time.Millisecond

func testPayload(size int) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	return payload
}

// rangeStart returns the first byte a request asks for, or -1 without Range.
func rangeStart(r *gohttp.Request) int64 {
	var start int64
	if _, err := fmt.Sscanf(r.Header.Get(base.HttpHeaderRange), "bytes=%d-", &start); err != nil {
		return -1
	}
	return start
}

// slowResponseWriter throttles writes so connections overlap: 32 KiB per
// pause, 5 ms unless set.
type slowResponseWriter struct {
	gohttp.ResponseWriter
	pause time.Duration
}

func (w *slowResponseWriter) Write(p []byte) (int, error) {
	pause := w.pause
	if pause == 0 {
		pause = 5 * time.Millisecond
	}
	written := 0
	for written < len(p) {
		end := min(written+32*1024, len(p))
		n, err := w.ResponseWriter.Write(p[written:end])
		written += n
		if err != nil {
			return written, err
		}
		if f, ok := w.ResponseWriter.(gohttp.Flusher); ok {
			f.Flush()
		}
		time.Sleep(pause)
	}
	return written, nil
}

// restoredTwoChunks returns a fetcher resumed with two unstarted chunks of size/2 each, below
// the size work stealing splits, so each chunk is fetched by its own connection. Its retry
// delays use unit and it gets maxRetries retry rounds.
func restoredTwoChunks(t *testing.T, url string, size int64, unit time.Duration, maxRetries int) (*Fetcher, string) {
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
	f.Setup(retryController(t, maxRetries))
	f.retryUnit = unit
	return f, filepath.Join(dir, "slot.bin")
}

// retryController serves the default HTTP configuration with maxRetries retry rounds.
func retryController(t *testing.T, maxRetries int) *controller.Controller {
	cfg := new(FetcherManager).DefaultConfig().(*config)
	cfg.MaxRetries = maxRetries
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		if err := json.Unmarshal([]byte(test.ToJson(cfg)), v); err != nil {
			t.Error(err)
		}
	}
	return ctl
}

// newRetryFetcher returns a fresh fetcher with maxRetries retry rounds of unit-based delays.
func newRetryFetcher(t *testing.T, unit time.Duration, maxRetries int) *Fetcher {
	f := new(FetcherManager).Build().(*Fetcher)
	f.Setup(retryController(t, maxRetries))
	f.retryUnit = unit
	return f
}

func waitResult(t *testing.T, f *Fetcher, timeout time.Duration) error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- f.Wait() }()
	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		t.Fatal("Wait() did not return")
		return nil
	}
}

func waitDownload(t *testing.T, f *Fetcher, file string, payload []byte) {
	t.Helper()
	if err := waitResult(t, f, 30*time.Second); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded content differs from the server's")
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// failingStatuses is the mix of answers a refused request gets in these tests.
var failingStatuses = []int{
	gohttp.StatusForbidden,
	gohttp.StatusNotFound,
	gohttp.StatusInternalServerError,
	gohttp.StatusTooManyRequests,
	gohttp.StatusGone,
	gohttp.StatusUnauthorized,
	gohttp.StatusBadRequest,
	gohttp.StatusServiceUnavailable,
	0, // the connection is closed before any response
}

func writeFailure(w gohttp.ResponseWriter, status int) {
	if status == 0 {
		if conn, _, err := w.(gohttp.Hijacker).Hijack(); err == nil {
			conn.Close()
		}
		return
	}
	w.WriteHeader(status)
}

// ---------------------------------------------------------------------------
// A peer downloading normally
// ---------------------------------------------------------------------------

// TestFetcher_ConnectionLimitForbidden simulates a server that allows only a
// few concurrent connections and answers extra ones with 403. The refused
// chunks must be retried once a slot frees up instead of being dropped.
func TestFetcher_ConnectionLimitForbidden(t *testing.T) {
	const maxActive = 2
	payload := testPayload(8 * 1024 * 1024)

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
		Path:  downloadDir,
		Name:  "conn-limit.data",
		Extra: &fhttp.OptsExtra{Connections: 8},
	}
	if err := f.Resolve(&base.Request{URL: server.URL + "/conn-limit.data"}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, filepath.Join(downloadDir, "conn-limit.data"), payload)
	if refused.Load() == 0 {
		t.Fatal("server never refused a connection, test did not exercise the limit")
	}
}

// While one connection downloads, the other one's failures of any kind are
// retried without using the retry budget: here one retry round is allowed,
// and the chunk fails with a mix of statuses and a dropped connection many
// more times than that before the server lets it through.
func TestFetcher_MixedFailuresWhilePeerHealthy(t *testing.T) {
	payload := testPayload(512 * 1024)
	half := int64(len(payload) / 2)
	const failures = 12
	var bRequests atomic.Int32
	release := make(chan struct{})
	var releaseOnce sync.Once
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if rangeStart(r) == half {
			n := int(bRequests.Add(1))
			if n <= failures {
				if n == failures {
					releaseOnce.Do(func() { close(release) })
				}
				writeFailure(w, failingStatuses[int(n-1)%len(failingStatuses)])
				return
			}
			serveRangeContent(w, r, payload)
			return
		}
		// The first chunk sends its headers and part of its body, then holds
		// the rest until the second chunk has failed often enough.
		w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes 0-%d/%d", half-1, len(payload)))
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprint(half))
		w.WriteHeader(gohttp.StatusPartialContent)
		_, _ = w.Write(payload[:1024])
		w.(gohttp.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write(payload[1024:half])
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 1)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	if got := bRequests.Load(); got <= failures {
		t.Fatalf("second chunk was requested %d times, want more than %d", got, failures)
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

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), time.Second, defaultMaxRetries)
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

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), time.Second, defaultMaxRetries)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	mu.Lock()
	defer mu.Unlock()
	if refused != 2 {
		t.Fatalf("refused %d requests, want 2", refused)
	}
}

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

// ---------------------------------------------------------------------------
// Complete outages
// ---------------------------------------------------------------------------

// With every request failing, each retry round gives each unfinished chunk one
// request and counts once for the task: two chunks and three rounds make
// exactly eight requests, then the task fails.
func TestFetcher_OutageRemainsFailed(t *testing.T) {
	const size, rounds = 512 * 1024, 3
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		requests.Add(1)
		w.WriteHeader(gohttp.StatusServiceUnavailable)
	}))
	defer server.Close()

	f, _ := restoredTwoChunks(t, server.URL+"/slot.bin", size, fastRetryUnit, rounds)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	err := waitResult(t, f, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("Wait() = %v, want a 503 failure", err)
	}
	if got, want := requests.Load(), int32(2*(1+rounds)); got != want {
		t.Fatalf("server got %d requests, want %d (first try and %d rounds for 2 chunks)", got, want, rounds)
	}
	if f.getState() != stateError {
		t.Fatalf("state = %v, want stateError", f.getState())
	}
}

// The delays of the retry rounds grow 1, 2, 3... units after a complete
// outage, and the task fails after the configured rounds.
func TestFetcher_OutageRoundDelaysGrow(t *testing.T) {
	const unit = 40 * time.Millisecond
	var mu sync.Mutex
	var times []time.Time
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		mu.Lock()
		times = append(times, time.Now())
		mu.Unlock()
		w.WriteHeader(gohttp.StatusBadGateway)
	}))
	defer server.Close()

	// One chunk: restore two, finish the first.
	f, _ := restoredTwoChunks(t, server.URL+"/slot.bin", 512*1024, unit, 4)
	f.connections[0].Completed = true
	f.connections[0].State = connCompleted
	f.connections[0].Chunk.Downloaded = 256 * 1024
	f.connections[0].Downloaded = 256 * 1024
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitResult(t, f, 30*time.Second); err == nil {
		t.Fatal("Wait() succeeded during an outage")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(times) != 5 {
		t.Fatalf("server got %d requests, want 5 (first try and 4 rounds)", len(times))
	}
	for i := 1; i < len(times); i++ {
		gap := times[i].Sub(times[i-1])
		want := time.Duration(i) * unit
		if gap < want || gap > want+400*time.Millisecond {
			t.Fatalf("gap before round %d = %v, want about %v", i, gap, want)
		}
	}
}

// An outage that ends within the retry rounds: the download completes.
func TestFetcher_OutageRecovers(t *testing.T) {
	payload := testPayload(512 * 1024)
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		// Both chunks fail their first try and the first two rounds.
		if n := requests.Add(1); n <= 6 {
			writeFailure(w, failingStatuses[int(n-1)%len(failingStatuses)])
			return
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 3)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
}

// Every chunk fails until one of them gets through: that valid response resets
// the budget of the whole task, not only of the chunk that recovered. Two
// rounds are allowed. The first chunk recovers in the second round; the
// second chunk keeps failing until after the first one has completed, and then
// needs up to two more rounds, which only the reset gives it.
func TestFetcher_OneRecoveryResetsTaskBudget(t *testing.T) {
	payload := testPayload(512 * 1024)
	var aRequests, bFailuresAfterA atomic.Int32
	var aDone atomic.Bool
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if rangeStart(r) == 0 {
			// First try and first round fail, the second round gets through.
			if aRequests.Add(1) <= 2 {
				w.WriteHeader(gohttp.StatusServiceUnavailable)
				return
			}
			serveRangeContent(w, r, payload)
			aDone.Store(true)
			return
		}
		if !aDone.Load() || bFailuresAfterA.Add(1) <= 2 {
			w.WriteHeader(gohttp.StatusBadGateway)
			return
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 2)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	if got := aRequests.Load(); got != 3 {
		t.Fatalf("first chunk was requested %d times, want 3", got)
	}
}

// ---------------------------------------------------------------------------
// Resume and restart
// ---------------------------------------------------------------------------

// A restarted download gets a fresh retry budget: the first attempt used its
// rounds up and failed, and the second attempt meets an outage that clears
// well within its own rounds.
func TestFetcher_RestartResetsRetryBudget(t *testing.T) {
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
	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), 20*time.Millisecond, defaultMaxRetries)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitResult(t, f, 30*time.Second); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("first attempt = %v, want terminal 403", err)
	}
	// The origin refuses for a moment longer, well within the five rounds (300 ms).
	recoverAt.Store(time.Now().Add(100 * time.Millisecond).UnixNano())
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
}

// A paused download resumes with a fresh retry budget. The first attempt uses
// both of its rounds, and is paused while its last requests wait for response
// headers; Pause stops them. After the resume, the outage needs both rounds
// again.
func TestFetcher_ResumeResetsRetryBudget(t *testing.T) {
	payload := testPayload(512 * 1024)
	var resumed atomic.Bool
	var requests atomic.Int32
	held := make(chan struct{}, 4)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		n := requests.Add(1)
		if !resumed.Load() && n > 4 {
			// The second round's requests: wait for headers until cancelled.
			held <- struct{}{}
			<-r.Context().Done()
			return
		}
		if n <= 4 {
			w.WriteHeader(gohttp.StatusInternalServerError)
			return
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 2)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-held:
	case <-time.After(10 * time.Second):
		t.Fatal("the second round did not start")
	}
	pauseStart := time.Now()
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(pauseStart); took > 5*time.Second {
		t.Fatalf("Pause took %v with requests waiting for headers", took)
	}

	resumed.Store(true)
	requests.Store(0)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
}

// Pause stops a connection waiting for its next retry at once, however long
// the delay.
func TestFetcher_PauseStopsRetryWait(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		requests.Add(1)
		w.WriteHeader(gohttp.StatusServiceUnavailable)
	}))
	defer server.Close()

	f, _ := restoredTwoChunks(t, server.URL+"/slot.bin", 512*1024, time.Hour, defaultMaxRetries)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "both chunks to fail", func() bool { return requests.Load() == 2 })
	start := time.Now()
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("Pause took %v during an hour-long retry delay", took)
	}
}

// After the retries run out the task fails but keeps what it downloaded, and a
// new Start continues from there.
func TestFetcher_FailedTaskKeepsDownloadedData(t *testing.T) {
	payload := testPayload(512 * 1024)
	half := int64(len(payload) / 2)
	var down atomic.Bool
	var resumedFrom atomic.Int64
	resumedFrom.Store(-1)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		start := rangeStart(r)
		if down.Load() {
			w.WriteHeader(gohttp.StatusServiceUnavailable)
			return
		}
		if start == 0 {
			// Send 100 KiB of the first chunk, then the link goes down.
			w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes 0-%d/%d", half-1, len(payload)))
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprint(half))
			w.WriteHeader(gohttp.StatusPartialContent)
			_, _ = w.Write(payload[:100*1024])
			w.(gohttp.Flusher).Flush()
			down.Store(true)
			if conn, _, err := w.(gohttp.Hijacker).Hijack(); err == nil {
				conn.Close()
			}
			return
		}
		if start > 0 && start < half {
			resumedFrom.CompareAndSwap(-1, start)
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 2)
	// Only the first chunk is left.
	f.connections[1].Completed = true
	f.connections[1].State = connCompleted
	f.connections[1].Chunk.Downloaded = half
	f.connections[1].Downloaded = half
	defer f.Close()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(make([]byte, half), payload[half:]...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitResult(t, f, 30*time.Second); err == nil {
		t.Fatal("Wait() succeeded after the link went down")
	}
	if got := f.connections[0].Chunk.Downloaded; got < 100*1024 {
		t.Fatalf("first chunk progress after the failure = %d, want the 100 KiB downloaded kept", got)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:100*1024], payload[:100*1024]) {
		t.Fatal("the downloaded part of the file was lost")
	}

	down.Store(false)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	if from := resumedFrom.Load(); from < 100*1024 {
		t.Fatalf("restart requested the chunk from %d, want from the kept progress", from)
	}
}

// ---------------------------------------------------------------------------
// Completion and other request paths
// ---------------------------------------------------------------------------

// A slow-start batch in which one connection succeeds and the other keeps
// failing: once the failing one gives up, every worker has exited without the
// batch completing. Wait must still return. About 1 MB/s per connection lets
// the fetcher measure speeds, so that no finished connection takes over part
// of the failing one's chunk (which would complete the batch by itself).
func TestFetcher_WaitReturnsWhenAllWorkersExit(t *testing.T) {
	payload := testPayload(4 * 1024 * 1024)
	// Slow start splits the file in two, then the first half again: the
	// second half is the chunk that keeps failing.
	refuseFrom := int64(len(payload) / 2)
	var resolved atomic.Bool
	var refused atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if resolved.CompareAndSwap(false, true) {
			// Resolve: a slow full response, so Start finds the file unfinished.
			w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprint(len(payload)))
			w.WriteHeader(gohttp.StatusOK)
			for offset := 0; offset < len(payload); offset += 16 * 1024 {
				if _, err := w.Write(payload[offset : offset+16*1024]); err != nil {
					return
				}
				w.(gohttp.Flusher).Flush()
				time.Sleep(10 * time.Millisecond)
			}
			return
		}
		if rangeStart(r) >= refuseFrom {
			refused.Add(1)
			w.WriteHeader(gohttp.StatusServiceUnavailable)
			return
		}
		serveRangeContent(&slowResponseWriter{ResponseWriter: w, pause: 30 * time.Millisecond}, r, payload)
	}))
	defer server.Close()

	f := newRetryFetcher(t, 20*time.Millisecond, 1)
	f.Meta().Opts = &base.Options{Path: t.TempDir(), Name: "batch.data", Extra: &fhttp.OptsExtra{Connections: 3}}
	if err := f.Resolve(&base.Request{URL: server.URL + "/batch.data"}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	err := waitResult(t, f, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("Wait() = %v, want a 503 failure", err)
	}
	if refused.Load() == 0 {
		t.Fatal("no chunk was refused, the test did not exercise the failing worker")
	}
}

// Resolve follows the same policy: it is retried with growing delays and
// fails once the rounds are used up.
func TestFetcher_ResolveRetries(t *testing.T) {
	t.Run("recovers", func(t *testing.T) {
		var requests atomic.Int32
		server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
			if n := requests.Add(1); n <= 3 {
				writeFailure(w, []int{gohttp.StatusTooManyRequests, gohttp.StatusInternalServerError, 0}[n-1])
				return
			}
			_, _ = w.Write([]byte("resolved"))
		}))
		defer server.Close()
		f := newRetryFetcher(t, fastRetryUnit, 3)
		defer f.Close()
		if err := f.Resolve(&base.Request{URL: server.URL + "/file.txt"}); err != nil {
			t.Fatal(err)
		}
		if got := f.Meta().Res.Size; got != int64(len("resolved")) {
			t.Fatalf("size = %d", got)
		}
	})
	t.Run("gives up", func(t *testing.T) {
		var requests atomic.Int32
		server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
			requests.Add(1)
			w.WriteHeader(gohttp.StatusNotFound)
		}))
		defer server.Close()
		f := newRetryFetcher(t, fastRetryUnit, 2)
		defer f.Close()
		err := f.Resolve(&base.Request{URL: server.URL + "/missing.txt"})
		if re := extractRequestError(err); re == nil || re.Code != gohttp.StatusNotFound {
			t.Fatalf("Resolve() = %v, want 404", err)
		}
		if got := requests.Load(); got != 3 {
			t.Fatalf("server got %d requests, want 3 (first try and 2 rounds)", got)
		}
	})
	t.Run("invalid request is not retried", func(t *testing.T) {
		f := newRetryFetcher(t, time.Hour, defaultMaxRetries)
		defer f.Close()
		start := time.Now()
		err := f.Resolve(&base.Request{URL: "http://127.0.0.1:1/x", Extra: &fhttp.ReqExtra{Method: "BAD METHOD"}})
		if err == nil {
			t.Fatal("Resolve() succeeded with an invalid method")
		}
		if time.Since(start) > 5*time.Second {
			t.Fatal("an invalid request was retried")
		}
	})
	t.Run("close stops the retries", func(t *testing.T) {
		var requests atomic.Int32
		server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
			requests.Add(1)
			w.WriteHeader(gohttp.StatusBadGateway)
		}))
		defer server.Close()
		f := newRetryFetcher(t, time.Hour, defaultMaxRetries)
		errCh := make(chan error, 1)
		go func() { errCh <- f.Resolve(&base.Request{URL: server.URL + "/x"}) }()
		waitFor(t, "the first request", func() bool { return requests.Load() == 1 })
		f.Close()
		select {
		case err := <-errCh:
			if err == nil {
				t.Fatal("Resolve() succeeded after Close")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Close did not stop the Resolve retries")
		}
	})
}

// A download without Range support and without the Resolve response (here
// discarded by a pause) uses the sequential fallback, under the same policy.
func TestFetcher_SequentialFallbackRetries(t *testing.T) {
	payload := testPayload(256 * 1024)
	for _, tc := range []struct {
		name     string
		failures int32
		wantErr  bool
	}{
		{name: "recovers", failures: 3},
		{name: "gives up", failures: 1 << 30, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				n := requests.Add(1)
				if n > 1 && n <= 1+tc.failures {
					writeFailure(w, failingStatuses[int(n)%len(failingStatuses)])
					return
				}
				w.Header().Set(base.HttpHeaderContentLength, fmt.Sprint(len(payload)))
				_, _ = w.Write(payload)
			}))
			defer server.Close()

			dir := t.TempDir()
			f := newRetryFetcher(t, fastRetryUnit, 3)
			f.Meta().Opts = &base.Options{Path: dir, Name: "seq.data", Extra: &fhttp.OptsExtra{Connections: 4}}
			if err := f.Resolve(&base.Request{URL: server.URL + "/seq.data"}); err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if f.Meta().Res.Range {
				t.Fatal("the test server must not advertise Range")
			}
			if err := f.Pause(); err != nil {
				t.Fatal(err)
			}
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if !tc.wantErr {
				waitDownload(t, f, filepath.Join(dir, "seq.data"), payload)
				return
			}
			if err := waitResult(t, f, 30*time.Second); err == nil {
				t.Fatal("Wait() succeeded during an outage")
			}
			// Resolve, then the first try and three rounds.
			if got := requests.Load(); got != 5 {
				t.Fatalf("server got %d requests, want 5", got)
			}
		})
	}
}

// Errors that retrying cannot fix end the connection at once, without using
// the retry policy.
func TestFetcher_TerminalErrorsAreNotRetried(t *testing.T) {
	t.Run("local write error", func(t *testing.T) {
		payload := testPayload(64 * 1024)
		var requests atomic.Int32
		server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
			requests.Add(1)
			serveRangeContent(w, r, payload)
		}))
		defer server.Close()

		f := newRetryFetcher(t, time.Hour, defaultMaxRetries)
		f.meta.Req = &base.Request{URL: server.URL}
		f.meta.Res = &base.Resource{Range: true, Size: int64(len(payload))}
		path := filepath.Join(t.TempDir(), "readonly.data")
		if err := os.WriteFile(path, make([]byte, len(payload)), 0o644); err != nil {
			t.Fatal(err)
		}
		readOnly, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer readOnly.Close()
		f.file = readOnly

		conn := &connection{ID: 1, State: connNotStarted, Chunk: newChunk(0, int64(len(payload))-1), ctx: context.Background()}
		f.connections = []*connection{conn}
		f.wg.Add(1)
		done := make(chan struct{})
		go func() {
			f.runConnection(conn)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("a write error was retried")
		}
		var te *terminalError
		if !conn.failed || !errors.As(conn.lastErr, &te) {
			t.Fatalf("connection failed = %v, err = %v; want a terminal write error", conn.failed, conn.lastErr)
		}
		if got := requests.Load(); got != 1 {
			t.Fatalf("server got %d requests, want 1", got)
		}
	})
	t.Run("invalid request", func(t *testing.T) {
		f := newRetryFetcher(t, time.Hour, defaultMaxRetries)
		f.meta.Req = &base.Request{URL: "http://127.0.0.1:1/x", Extra: &fhttp.ReqExtra{Method: "BAD METHOD"}}
		f.meta.Res = &base.Resource{Range: true, Size: 1024}
		conn := &connection{ID: 1, State: connNotStarted, Chunk: newChunk(0, 1023), ctx: context.Background()}
		f.connections = []*connection{conn}
		f.wg.Add(1)
		done := make(chan struct{})
		go func() {
			f.runConnection(conn)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("an invalid request was retried")
		}
		if !conn.failed || !isTerminalError(conn.lastErr) {
			t.Fatalf("connection failed = %v, err = %v; want a terminal error", conn.failed, conn.lastErr)
		}
	})
}
