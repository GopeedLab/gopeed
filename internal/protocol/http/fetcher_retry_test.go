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
// delays use unit and each connection gets maxRetries retries of its own.
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

// retryController serves the default HTTP configuration with maxRetries retries per connection.
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

// newRetryFetcher returns a fresh fetcher with maxRetries retries per connection and unit-based delays.
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
// retried without using its allowance: here one retry is allowed,
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

// With every request failing, each chunk's connection spends its own
// allowance: two chunks with three retries each make exactly eight requests,
// then the task fails.
func TestFetcher_OutageRemainsFailed(t *testing.T) {
	const size, retries = 512 * 1024, 3
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		requests.Add(1)
		w.WriteHeader(gohttp.StatusServiceUnavailable)
	}))
	defer server.Close()

	f, _ := restoredTwoChunks(t, server.URL+"/slot.bin", size, fastRetryUnit, retries)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	err := waitResult(t, f, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("Wait() = %v, want a 503 failure", err)
	}
	if got, want := requests.Load(), int32(2*(1+retries)); got != want {
		t.Fatalf("server got %d requests, want %d (first try and %d retries for each of 2 chunks)", got, want, retries)
	}
	if f.getState() != stateError {
		t.Fatalf("state = %v, want stateError", f.getState())
	}
}

// The delays of a connection's retries grow 1, 2, 3... units during a
// complete outage, and the task fails after the configured retries.
func TestFetcher_OutageRetryDelaysGrow(t *testing.T) {
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
		t.Fatalf("server got %d requests, want 5 (first try and 4 retries)", len(times))
	}
	for i := 1; i < len(times); i++ {
		gap := times[i].Sub(times[i-1])
		want := time.Duration(i) * unit
		if gap < want || gap > want+400*time.Millisecond {
			t.Fatalf("gap before retry %d = %v, want about %v", i, gap, want)
		}
	}
}

// An outage that ends within the connections' allowances: the download completes.
func TestFetcher_OutageRecovers(t *testing.T) {
	payload := testPayload(512 * 1024)
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		// Both chunks fail their first try and their first two retries.
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
// the allowance of every unfinished connection, not only of the one that
// recovered. Two retries are allowed. The first chunk recovers on its second
// retry; the second chunk keeps failing until after the first one has
// completed, and then needs two more retries, which only the reset gives it.
func TestFetcher_OneRecoveryResetsEveryAllowance(t *testing.T) {
	payload := testPayload(512 * 1024)
	var aRequests, bFailuresAfterA atomic.Int32
	var aDone atomic.Bool
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if rangeStart(r) == 0 {
			// The first try and the first retry fail, the second retry gets through.
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
// Per-connection retries
// ---------------------------------------------------------------------------

// A connection that fails fast retries at its own pace while another one is
// still waiting for its response headers: the slow request is answered only
// after the fast connection has retried three times, so waiting for it would
// stall the download.
func TestFetcher_FastFailureRetriesBeforeSlowPeerResponds(t *testing.T) {
	payload := testPayload(512 * 1024)
	half := int64(len(payload) / 2)
	const fastFailures = 3
	var fastRequests atomic.Int32
	retried := make(chan struct{})
	var retriedOnce sync.Once
	var slowTimedOut atomic.Bool
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if rangeStart(r) == half {
			n := fastRequests.Add(1)
			if n == fastFailures+1 {
				retriedOnce.Do(func() { close(retried) })
			}
			if n <= fastFailures {
				w.WriteHeader(gohttp.StatusServiceUnavailable)
				return
			}
			serveRangeContent(w, r, payload)
			return
		}
		// The slow request holds its headers until the fast connection has
		// retried past its failures.
		select {
		case <-retried:
		case <-time.After(10 * time.Second):
			slowTimedOut.Store(true)
		case <-r.Context().Done():
			return
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, defaultMaxRetries)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	if slowTimedOut.Load() {
		t.Fatal("the fast connection waited for the slow one's headers instead of retrying")
	}
}

// Connections failing at the same time each spend only their own allowance.
// Two retries are allowed; both chunks fail their first try and their first
// retry while nothing is healthy, four failures in all, and are served on
// their second retry. A shared task counter would have run out.
func TestFetcher_SimultaneousFailuresChargeOwnAllowance(t *testing.T) {
	payload := testPayload(512 * 1024)
	half := int64(len(payload) / 2)
	const failuresEach = 2
	var mu sync.Mutex
	requests := map[int64]int{}
	bothFailed := make(chan struct{})
	var bothFailedOnce sync.Once
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		start := rangeStart(r)
		mu.Lock()
		requests[start]++
		n := requests[start]
		if requests[0] > failuresEach && requests[half] > failuresEach {
			bothFailedOnce.Do(func() { close(bothFailed) })
		}
		mu.Unlock()
		if n <= failuresEach {
			w.WriteHeader(gohttp.StatusBadGateway)
			return
		}
		// Hold the first success until the other chunk has failed as often,
		// so that every failure happens with no healthy peer.
		select {
		case <-bothFailed:
		case <-r.Context().Done():
			return
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, failuresEach)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	mu.Lock()
	defer mu.Unlock()
	for start, n := range requests {
		if n != failuresEach+1 {
			t.Fatalf("chunk at %d was requested %d times, want %d", start, n, failuresEach+1)
		}
	}
}

// A chunk whose connection has used up its allowance is kept while a slower
// request may still succeed. One retry is allowed: the second chunk fails its
// first try and its retry and then waits, without further requests, until the
// first chunk's request, held at its headers until then, succeeds. That
// success gives the second chunk a new allowance, and it completes.
func TestFetcher_LateSuccessRestoresExhaustedChunk(t *testing.T) {
	payload := testPayload(512 * 1024)
	half := int64(len(payload) / 2)
	var spentRequests, requestsWhileParked atomic.Int32
	spent := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if rangeStart(r) == half {
			switch n := spentRequests.Add(1); {
			case n <= 2:
				if n == 2 {
					close(spent)
				}
				w.WriteHeader(gohttp.StatusServiceUnavailable)
				return
			default:
				select {
				case <-release:
				default:
					requestsWhileParked.Add(1)
				}
				serveRangeContent(w, r, payload)
				return
			}
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		serveRangeContent(w, r, payload)
	}))
	defer server.Close()

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 1)
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-spent:
	case <-time.After(10 * time.Second):
		t.Fatal("the second chunk did not use up its allowance")
	}
	// Far longer than the retry delays: an exhausted chunk must not retry, nor
	// fail the task, while its peer is still in flight.
	time.Sleep(200 * time.Millisecond)
	if f.getState() == stateError {
		t.Fatal("the task failed while a request could still recover it")
	}
	close(release)
	waitDownload(t, f, file, payload)
	if n := requestsWhileParked.Load(); n != 0 {
		t.Fatalf("the exhausted chunk sent %d requests before its peer succeeded, want 0", n)
	}
	if n := spentRequests.Load(); n != 3 {
		t.Fatalf("the second chunk was requested %d times, want 3", n)
	}
}

// ---------------------------------------------------------------------------
// Resume and restart
// ---------------------------------------------------------------------------

// A restarted download gets fresh allowances: the first attempt used its
// retries up and failed, and the second attempt meets an outage that clears
// well within its own retries.
func TestFetcher_RestartResetsRetryAllowances(t *testing.T) {
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
	// The origin refuses for a moment longer, well within the five retries (300 ms).
	recoverAt.Store(time.Now().Add(100 * time.Millisecond).UnixNano())
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
}

// A paused download resumes with fresh allowances. In the first attempt each
// connection uses both of its retries, and is paused while its last requests
// wait for response headers; Pause stops them. After the resume, the outage
// needs both retries again.
func TestFetcher_ResumeResetsRetryAllowances(t *testing.T) {
	payload := testPayload(512 * 1024)
	var resumed atomic.Bool
	var requests atomic.Int32
	held := make(chan struct{}, 4)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		n := requests.Add(1)
		if !resumed.Load() && n > 4 {
			// The second retries: wait for headers until cancelled.
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
		t.Fatal("the second retries did not start")
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

// Resolve follows the same rules on its own request sequence: it is retried
// with growing delays and fails once its allowance is used up.
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
			t.Fatalf("server got %d requests, want 3 (first try and 2 retries)", got)
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
			// Resolve, then the first try and three retries.
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

// ---------------------------------------------------------------------------
// Response header timeout
// ---------------------------------------------------------------------------

// hangUntilCancelled never answers: it waits until the client gives up.
func hangUntilCancelled(r *gohttp.Request) {
	<-r.Context().Done()
}

// A request whose headers never arrive times out and, next to a healthy peer,
// is retried for free: no retry is allowed on the allowance here.
func TestFetcher_HeaderTimeoutRetriedWhilePeerHealthy(t *testing.T) {
	payload := testPayload(512 * 1024)
	half := int64(len(payload) / 2)
	var bRequests atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if rangeStart(r) == half {
			if bRequests.Add(1) == 1 {
				hangUntilCancelled(r)
				return
			}
			close(release)
			serveRangeContent(w, r, payload)
			return
		}
		// The first chunk downloads until the second one has been retried.
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

	f, file := restoredTwoChunks(t, server.URL+"/slot.bin", int64(len(payload)), fastRetryUnit, 0)
	f.headerTimeout = 100 * time.Millisecond
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDownload(t, f, file, payload)
	if got := bRequests.Load(); got != 2 {
		t.Fatalf("second chunk was requested %d times, want 2", got)
	}
}

// When every request hangs, each one times out, every connection spends its
// allowance, and the task fails with the timeout.
func TestFetcher_HeaderTimeoutOutageFails(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		requests.Add(1)
		hangUntilCancelled(r)
	}))
	defer server.Close()

	f, _ := restoredTwoChunks(t, server.URL+"/slot.bin", 512*1024, fastRetryUnit, 2)
	f.headerTimeout = 50 * time.Millisecond
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	err := waitResult(t, f, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), errResponseHeaderTimeout.Error()) {
		t.Fatalf("Wait() = %v, want a response header timeout", err)
	}
	// Two chunks: the first try and two retries each.
	if got := requests.Load(); got != 6 {
		t.Fatalf("server got %d requests, want 6", got)
	}
}

// Pause stops a request waiting for its headers at once, long before the
// header timeout.
func TestFetcher_PauseStopsHeaderWait(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		requests.Add(1)
		hangUntilCancelled(r)
	}))
	defer server.Close()

	f, _ := restoredTwoChunks(t, server.URL+"/slot.bin", 512*1024, time.Hour, defaultMaxRetries)
	f.headerTimeout = time.Hour
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "both requests", func() bool { return requests.Load() == 2 })
	start := time.Now()
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("Pause took %v while requests waited for headers", took)
	}
}

// The header timeout also covers the browser impersonation retry that
// internal/httpclient makes after a 403: here that retry never gets headers.
func TestFetcher_HeaderTimeoutCoversImpersonationRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		hangUntilCancelled(r)
	}))
	defer server.Close()

	f := newRetryFetcher(t, fastRetryUnit, 0)
	f.headerTimeout = 200 * time.Millisecond
	defer f.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- f.Resolve(&base.Request{URL: server.URL + "/file"}) }()
	select {
	case err := <-errCh:
		if !errors.Is(err, errResponseHeaderTimeout) {
			t.Fatalf("Resolve() = %v, want a response header timeout", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the impersonation retry ignored the header timeout")
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("server got %d requests, want the 403 and its browser retry", got)
	}
}

// ---------------------------------------------------------------------------
// Expired link
// ---------------------------------------------------------------------------

// The link expires during the download: the first requests are served, every
// later one gets 403. The task must fail after its retries, report the
// 403, and keep what it has downloaded.
func TestFetcher_ExpiredLinkFailsAndKeepsData(t *testing.T) {
	payload := testPayload(4 * 1024 * 1024)
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		// The Resolve request and the first Range request are served.
		if requests.Add(1) > 2 {
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		serveRangeContent(&slowResponseWriter{ResponseWriter: w}, r, payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	f := newRetryFetcher(t, fastRetryUnit, defaultMaxRetries)
	f.Meta().Opts = &base.Options{Path: dir, Name: "expired.data", Extra: &fhttp.OptsExtra{Connections: 8}}
	if err := f.Resolve(&base.Request{URL: server.URL + "/expired.data"}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	err := waitResult(t, f, 60*time.Second)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("Wait() = %v, want a 403 failure", err)
	}
	if f.getState() != stateError {
		t.Fatalf("state = %v, want stateError", f.getState())
	}

	got, readErr := os.ReadFile(filepath.Join(dir, "expired.data"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	var kept int64
	f.connMu.Lock()
	defer f.connMu.Unlock()
	for _, conn := range f.connections {
		begin, n := conn.Chunk.Begin, conn.Chunk.Downloaded
		if n > 0 && !bytes.Equal(got[begin:begin+n], payload[begin:begin+n]) {
			t.Fatalf("connection %d: the downloaded bytes %d-%d were not kept", conn.ID, begin, begin+n-1)
		}
		kept += n
	}
	if kept == 0 || kept >= int64(len(payload)) {
		t.Fatalf("kept %d of %d bytes, want a partial download", kept, len(payload))
	}
}

// ---------------------------------------------------------------------------
// Without Range support: the kept Resolve response
// ---------------------------------------------------------------------------

// cutResponse sends a 200 with the given headers and body, then drops the
// connection before the response is complete.
func cutResponse(t *testing.T, w gohttp.ResponseWriter, header string, body []byte) {
	conn, rw, err := w.(gohttp.Hijacker).Hijack()
	if err != nil {
		t.Errorf("hijack response: %v", err)
		return
	}
	defer conn.Close()
	_, _ = fmt.Fprint(rw, "HTTP/1.1 200 OK\r\n"+header+"Connection: close\r\n\r\n")
	_, _ = rw.Write(body)
	_ = rw.Flush()
}

// The Resolve response of a resource without Range support breaks part way:
// the download requests the whole resource again and rewrites the file from
// the start. With an unknown length the stale bytes of the first response,
// here longer than the resource, must be truncated.
func TestFetcher_ResolveBodyCutRetriesFromStart(t *testing.T) {
	payload := testPayload(256 * 1024)
	for _, tc := range []struct {
		name   string
		header string
		first  []byte
	}{
		{
			name:   "known size",
			header: fmt.Sprintf("Content-Length: %d\r\n", len(payload)),
			first:  payload[:100*1024],
		},
		{
			name:   "unknown size",
			header: "Transfer-Encoding: chunked\r\n",
			first:  append([]byte(fmt.Sprintf("%x\r\n", 300*1024)), bytes.Repeat([]byte("x"), 300*1024)...),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				if requests.Add(1) == 1 {
					cutResponse(t, w, tc.header, tc.first)
					return
				}
				if r.Header.Get(base.HttpHeaderRange) != "" {
					t.Errorf("re-request asked for Range %q, want the whole resource", r.Header.Get(base.HttpHeaderRange))
				}
				_, _ = w.Write(payload)
			}))
			defer server.Close()

			dir := t.TempDir()
			f := newRetryFetcher(t, fastRetryUnit, 2)
			f.Meta().Opts = &base.Options{Path: dir, Name: "cut.data", Extra: &fhttp.OptsExtra{Connections: 4}}
			if err := f.Resolve(&base.Request{URL: server.URL + "/cut.data"}); err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if f.Meta().Res.Range {
				t.Fatal("the test server must not advertise Range")
			}
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			waitDownload(t, f, filepath.Join(dir, "cut.data"), payload)
			if got := requests.Load(); got != 2 {
				t.Fatalf("server got %d requests, want the Resolve request and one re-request", got)
			}
		})
	}
}

// A one-time URL: the Resolve response breaks part way and every later
// request is refused with 403. The download fails after its retries with
// the 403, without a special case for such URLs.
func TestFetcher_OneTimeURLFailsAfterRetries(t *testing.T) {
	payload := testPayload(256 * 1024)
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if requests.Add(1) == 1 {
			cutResponse(t, w, fmt.Sprintf("Content-Length: %d\r\n", len(payload)), payload[:64*1024])
			return
		}
		w.WriteHeader(gohttp.StatusForbidden)
	}))
	defer server.Close()

	dir := t.TempDir()
	f := newRetryFetcher(t, fastRetryUnit, 2)
	f.Meta().Opts = &base.Options{Path: dir, Name: "once.data"}
	if err := f.Resolve(&base.Request{URL: server.URL + "/once.data"}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	err := waitResult(t, f, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("Wait() = %v, want a 403 failure", err)
	}
	if f.getState() != stateError {
		t.Fatalf("state = %v, want stateError", f.getState())
	}
	// Resolve, then one request per retry. The first 403 is repeated
	// once by the browser impersonation of internal/httpclient, which then
	// stops trying it for this URL.
	if got := requests.Load(); got != 4 {
		t.Fatalf("server got %d requests, want 4", got)
	}
}
