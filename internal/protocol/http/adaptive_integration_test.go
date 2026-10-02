package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// ============================================================================
// Test server: per-request rate, total cap, and a connection limit with 429.
// ============================================================================

const adaptivePiece = 16 * 1024

type adaptiveTestServer struct {
	data     []byte
	perConn  int64        // bytes/s for each request
	totalCap atomic.Int64 // bytes/s for all requests together, 0 means no cap
	maxConns atomic.Int32 // concurrent requests above this get 429, 0 means no limit
	noRange  bool
	// ignoreRange advertises byte ranges but answers every request with the
	// whole file and 200.
	ignoreRange bool

	active   atomic.Int32
	rejected atomic.Int32

	mu       sync.Mutex
	nextFree time.Time // pacing clock shared by all requests for the total cap

	srv *httptest.Server
}

func newAdaptiveTestServer(t *testing.T, size int, perConn, totalCap int64, maxConns int32, noRange bool) *adaptiveTestServer {
	t.Helper()
	s := &adaptiveTestServer{data: adaptiveTestData(size), perConn: perConn, noRange: noRange}
	s.totalCap.Store(totalCap)
	s.maxConns.Store(maxConns)
	s.srv = httptest.NewServer(gohttp.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

func adaptiveTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}
	return data
}

func (s *adaptiveTestServer) url() string {
	return s.srv.URL + "/adaptive.bin"
}

// reserve books one piece on the shared clock and returns when it may be sent.
func (s *adaptiveTestServer) reserve(n int) time.Time {
	limit := s.totalCap.Load()
	now := time.Now()
	if limit <= 0 {
		return now
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	at := s.nextFree
	if at.Before(now) {
		at = now
	}
	s.nextFree = at.Add(time.Duration(float64(n) / float64(limit) * float64(time.Second)))
	return at
}

func (s *adaptiveTestServer) serve(w gohttp.ResponseWriter, r *gohttp.Request) {
	n := s.active.Add(1)
	defer s.active.Add(-1)
	if limit := s.maxConns.Load(); limit > 0 && n > limit {
		s.rejected.Add(1)
		w.WriteHeader(gohttp.StatusTooManyRequests)
		return
	}

	start, end := int64(0), int64(len(s.data)-1)
	status := gohttp.StatusOK
	if !s.noRange {
		w.Header().Set("Accept-Ranges", "bytes")
		if spec := r.Header.Get("Range"); spec != "" && !s.ignoreRange {
			bounds := strings.SplitN(strings.TrimPrefix(spec, "bytes="), "-", 2)
			start, _ = strconv.ParseInt(bounds[0], 10, 64)
			if len(bounds) == 2 && bounds[1] != "" {
				end, _ = strconv.ParseInt(bounds[1], 10, 64)
			}
			if end > int64(len(s.data)-1) {
				end = int64(len(s.data) - 1)
			}
			status = gohttp.StatusPartialContent
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(s.data)))
		}
	}
	w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	w.WriteHeader(status)

	connNext := time.Now()
	for pos := start; pos <= end; {
		size := min(int64(adaptivePiece), end-pos+1)
		now := time.Now()
		if connNext.Before(now) {
			connNext = now // no burst credit for time spent waiting on the cap
		}
		at := connNext
		if shared := s.reserve(int(size)); shared.After(at) {
			at = shared
		}
		if d := time.Until(at); d > 0 {
			time.Sleep(d)
		}
		if _, err := w.Write(s.data[pos : pos+size]); err != nil {
			return
		}
		pos += size
		connNext = at.Add(time.Duration(float64(size) / float64(s.perConn) * float64(time.Second)))
	}
}

// ============================================================================
// Helpers
// ============================================================================

// useFastAdaptiveClock shortens the controller's windows so the integration
// tests converge in seconds. The ratios between window, evaluation and probe
// stay those of the real 2 s / 30 s clock closely enough for the behaviour.
func useFastAdaptiveClock(t *testing.T) {
	t.Helper()
	tick, probe := adaptiveTickInterval, adaptiveProbeInterval
	adaptiveTickInterval, adaptiveProbeInterval = 300*time.Millisecond, 3*time.Second
	t.Cleanup(func() { adaptiveTickInterval, adaptiveProbeInterval = tick, probe })
}

func boolPtr(v bool) *bool { return &v }

func newAdaptiveFetcher(t *testing.T, cfg config, s *adaptiveTestServer, extra *http.OptsExtra) (*Fetcher, string) {
	t.Helper()
	dir := t.TempDir()
	f := buildConfigFetcher(cfg).(*Fetcher)
	var optsExtra any // a typed nil would read as "set"
	if extra != nil {
		optsExtra = extra
	}
	err := f.Resolve(&base.Request{URL: s.url()}, &base.Options{
		Name:  "adaptive.bin",
		Path:  dir,
		Extra: optsExtra,
	})
	if err != nil {
		t.Fatal(err)
	}
	return f, filepath.Join(dir, "adaptive.bin")
}

func assertAdaptiveFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("downloaded file differs from the served data (got %d bytes, want %d)", len(got), len(want))
	}
}

// connTimeline samples the server's concurrent requests every 100 ms.
type connTimeline struct {
	mu      sync.Mutex
	samples []int
	stop    chan struct{}
	done    chan struct{}
}

func sampleConnections(s *adaptiveTestServer) *connTimeline {
	tl := &connTimeline{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(tl.done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-tl.stop:
				return
			case <-ticker.C:
				tl.mu.Lock()
				tl.samples = append(tl.samples, int(s.active.Load()))
				tl.mu.Unlock()
			}
		}
	}()
	return tl
}

func (tl *connTimeline) finish() []int {
	close(tl.stop)
	<-tl.done
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return append([]int(nil), tl.samples...)
}

// snapshot returns the samples taken so far.
func (tl *connTimeline) snapshot() []int {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return append([]int(nil), tl.samples...)
}

// settled is the most common connection count in samples[from:to].
func settled(samples []int, from, to int) int {
	if to > len(samples) {
		to = len(samples)
	}
	if from >= to {
		return -1
	}
	counts := map[int]int{}
	for _, v := range samples[from:to] {
		counts[v]++
	}
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	best := -1
	for _, k := range keys {
		if best < 0 || counts[k] > counts[best] {
			best = k
		}
	}
	return best
}

// waitFor polls cond every 50 ms until it holds or the timeout passes.
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %v waiting for %s", timeout, what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func waitDone(t *testing.T, f fetcher.Fetcher, timeout time.Duration) error {
	t.Helper()
	ch := make(chan error, 1)
	go func() { ch <- f.Wait() }()
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		t.Fatalf("download did not finish within %v", timeout)
		return nil
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestAdaptiveIntegrationSettlesAtTotalCap(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 64<<20, 1<<20, 6<<20, 0, false)
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)

	tl := sampleConnections(s)
	started := time.Now()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitDone(t, f, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	samples := tl.finish()
	t.Logf("took %v; connections every 100 ms: %v", time.Since(started).Round(time.Millisecond), samples)

	// Skip the growth (first 4 s) and the tail where chunks run out (last 1.5 s).
	got := settled(samples, 40, len(samples)-15)
	t.Logf("settled at %d connections", got)
	if got < 5 || got > 7 {
		t.Fatalf("settled at %d connections, want 6 +- 1", got)
	}
	if f.adaptive == nil {
		t.Fatal("adaptive controller was not used")
	}
	assertAdaptiveFile(t, path, s.data)
}

func TestAdaptiveIntegrationConnectionLimit429(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 40<<20, 1<<20, 0, 4, false)
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)

	tl := sampleConnections(s)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitDone(t, f, 60*time.Second); err != nil {
		t.Fatalf("a 429 connection limit must not fail the task: %v", err)
	}
	samples := tl.finish()
	t.Logf("connections every 100 ms: %v; rejected with 429: %d", samples, s.rejected.Load())

	got := settled(samples, 30, len(samples)-15)
	t.Logf("settled at %d connections", got)
	if got != 4 {
		t.Fatalf("settled at %d connections, want 4", got)
	}
	// A 429 on a follow-up request after a range split is retried before
	// anything is parked, so the count stays at the limit most of the time.
	steady := samples[30 : len(samples)-15]
	at4 := 0
	for _, n := range steady {
		if n == 4 {
			at4++
		}
	}
	t.Logf("at 4 connections for %d%% of the steady part", 100*at4/len(steady))
	if at4*10 < len(steady)*7 {
		t.Fatalf("at 4 connections for only %d of %d samples, want at least 70%%", at4, len(steady))
	}
	if s.rejected.Load() == 0 {
		t.Fatal("the server never refused a connection, so the limit was not found")
	}
	for i, c := range f.Stats().Snapshot.(*http.Stats).Connections {
		if c.Failed {
			t.Fatalf("connection %d reports a failure; a parked connection is not a failure", i)
		}
	}
	assertAdaptiveFile(t, path, s.data)
}

func TestAdaptiveIntegrationSlowdownGivesBack(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 64<<20, 1<<20, 6<<20, 0, false)
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)

	tl := sampleConnections(s)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 15*time.Second, "growth to 6 connections", func() bool { return s.active.Load() >= 6 })
	time.Sleep(time.Second)

	// The server now gives only 2 MB/s in total.
	s.totalCap.Store(2 << 20)
	mark := len(tl.snapshot())
	defer func() {
		if t.Failed() {
			t.Logf("after the slowdown: %v", tl.snapshot()[mark:])
		}
	}()
	waitFor(t, 10*time.Second, "connections given back", func() bool {
		recent := tl.snapshot()[mark:]
		return len(recent) >= 10 && settled(recent, len(recent)-10, len(recent)) <= 3
	})
	gaveBack := tl.snapshot()[mark:]
	t.Logf("after the slowdown: %v", gaveBack)

	// Let the rest finish quickly.
	s.totalCap.Store(0)
	if err := waitDone(t, f, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	tl.finish()
	assertAdaptiveFile(t, path, s.data)
}

func TestAdaptiveIntegrationPauseAndResumeGrowsAgain(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 64<<20, 1<<20, 6<<20, 0, false)
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 15*time.Second, "growth to 5 connections", func() bool { return s.active.Load() >= 5 })

	// Store, pause and restore through JSON, as the downloader does.
	fm := new(FetcherManager)
	data, err := fm.Store(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 5*time.Second, "the server to see the pause", func() bool { return s.active.Load() == 0 })

	rawData, _ := json.Marshal(data)
	rawMeta, _ := json.Marshal(f.Meta())
	var meta fetcher.FetcherMeta
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatal(err)
	}
	v, restore := fm.Restore()
	if err := json.Unmarshal(rawData, v); err != nil {
		t.Fatal(err)
	}
	resumed := restore(&meta, v).(*Fetcher)
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		raw, _ := json.Marshal(config{Connections: 16, Adaptive: true})
		json.Unmarshal(raw, v)
	}
	resumed.Setup(ctl)

	tl := sampleConnections(s)
	if err := resumed.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 15*time.Second, "growth after resume", func() bool { return s.active.Load() >= 5 })
	time.Sleep(200 * time.Millisecond) // let the sampler record it
	early := tl.snapshot()
	t.Logf("connections after resume: %v", early)
	// The controller measures one connection for two 300 ms windows before it
	// adds the second, so the first 500 ms of traffic show one request. (A
	// restored fetcher first waits up to 10 s in stopPrefetchAndCopyData for a
	// prefetch that never ran; that delay predates this change.)
	first := 0
	for first < len(early) && early[first] == 0 {
		first++
	}
	for i := first; i < len(early) && i < first+5; i++ {
		if early[i] > 1 {
			t.Fatalf("resume ran %d connections %d ms after its first request, want growth from one", early[i], (i-first)*100)
		}
	}
	if m := maxOf(early); m < 5 {
		t.Fatalf("resume grew to %d connections, want at least 5", m)
	}
	if err := waitDone(t, resumed, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	tl.finish()
	assertAdaptiveFile(t, path, s.data)
}

func TestAdaptiveIntegrationNoRangeIsSequential(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 4<<20, 8<<20, 0, 0, true)
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)
	if f.Meta().Res.Range {
		t.Fatal("setup: the server must not advertise Range")
	}
	tl := sampleConnections(s)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitDone(t, f, 30*time.Second); err != nil {
		t.Fatalf("a server without Range must download without error: %v", err)
	}
	samples := tl.finish()
	for _, n := range samples {
		if n > 1 {
			t.Fatalf("saw %d concurrent requests, want one sequential download", n)
		}
	}
	if n := len(f.Stats().Snapshot.(*http.Stats).Connections); n != 1 {
		t.Fatalf("got %d connections, want 1", n)
	}
	assertAdaptiveFile(t, path, s.data)
}

// With adaptive off, the slow-start batches are today's: 1, 2, 4, then the
// rest up to the configured 16 (the batch size squares and is capped).
func TestAdaptiveOffKeepsSlowStartSequence(t *testing.T) {
	cases := []struct {
		name  string
		cfg   config
		extra *http.OptsExtra
	}{
		{"global off", config{Connections: 16}, nil},
		{"task off overrides global on", config{Connections: 16, Adaptive: true}, &http.OptsExtra{Adaptive: boolPtr(false)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newAdaptiveTestServer(t, 64<<20, 4<<20, 0, 0, false)
			f, path := newAdaptiveFetcher(t, tc.cfg, s, tc.extra)
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if err := waitDone(t, f, 60*time.Second); err != nil {
				t.Fatal(err)
			}
			if f.adaptive != nil {
				t.Fatal("adaptive controller ran although adaptive is off")
			}
			f.slowStart.mu.Lock()
			got := append([]int(nil), f.slowStart.batchHistory...)
			f.slowStart.mu.Unlock()
			if fmt.Sprint(got) != fmt.Sprint([]int{1, 2, 4, 9}) {
				t.Fatalf("slow-start batches = %v, want [1 2 4 9]", got)
			}
			conns := f.Stats().Snapshot.(*http.Stats).Connections
			if len(conns) != 16 {
				t.Fatalf("got %d connections, want 16", len(conns))
			}
			for i, c := range f.connections {
				if c.ID != i {
					t.Fatalf("connection %d has ID %d, want IDs 0..15 in order", i, c.ID)
				}
			}
			assertAdaptiveFile(t, path, s.data)
		})
	}
}

// A record written before adaptive connections existed has no new fields.
// It must resume, with adaptive on and off.
func TestAdaptiveResumesPreChangeStoreBlob(t *testing.T) {
	for _, adaptive := range []bool{false, true} {
		t.Run(fmt.Sprintf("adaptive=%v", adaptive), func(t *testing.T) {
			useFastAdaptiveClock(t)
			const size = 8 << 20
			s := newAdaptiveTestServer(t, size, 8<<20, 0, 0, false)
			dir := t.TempDir()
			path := filepath.Join(dir, "adaptive.bin")

			// Four ranges, each a quarter downloaded, as an older version left them.
			partial := make([]byte, size)
			quarter := int64(size / 4)
			var conns []string
			for i := int64(0); i < 4; i++ {
				begin, end, done := i*quarter, (i+1)*quarter-1, quarter/4
				copy(partial[begin:begin+done], s.data[begin:begin+done])
				role := 2
				if i == 0 {
					role = 1
				}
				conns = append(conns, fmt.Sprintf(`{"ID":%d,"Role":%d,"State":2,"Chunk":{"Begin":%d,"End":%d,"Downloaded":%d},"Downloaded":%d,"Completed":false}`,
					i, role, begin, end, done, done))
			}
			if err := os.WriteFile(path, partial, 0o644); err != nil {
				t.Fatal(err)
			}
			blob := `{"Connections":[` + strings.Join(conns, ",") + `],"RedirectURL":"","IfRange":"","RangeReprobeEligible":false,` +
				`"RangeValidatorPinned":false,"SequentialSizeUnknown":false,"Range":true,"ResourceSize":` + strconv.Itoa(size) +
				`,"FileSize":` + strconv.Itoa(size) + `}`
			rawMeta := `{"req":{"url":"` + s.url() + `","extra":{"method":"","header":null,"body":""}},` +
				`"res":{"name":"","size":` + strconv.Itoa(size) + `,"range":true,"files":[{"name":"adaptive.bin","path":"","size":` + strconv.Itoa(size) + `}]},` +
				`"opts":{"name":"adaptive.bin","path":"` + dir + `","selectFiles":null,"extra":{"connections":4,"autoTorrent":null}}}`

			var meta fetcher.FetcherMeta
			if err := json.Unmarshal([]byte(rawMeta), &meta); err != nil {
				t.Fatal(err)
			}
			fm := new(FetcherManager)
			v, restore := fm.Restore()
			if err := json.Unmarshal([]byte(blob), v); err != nil {
				t.Fatal(err)
			}
			f := restore(&meta, v).(*Fetcher)
			ctl := controller.NewController()
			ctl.GetConfig = func(v any) {
				raw, _ := json.Marshal(config{Connections: 16, Adaptive: adaptive})
				json.Unmarshal(raw, v)
			}
			f.Setup(ctl)
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if err := waitDone(t, f, 30*time.Second); err != nil {
				t.Fatal(err)
			}
			if (f.adaptive != nil) != adaptive {
				t.Fatalf("adaptive controller used = %v, want %v", f.adaptive != nil, adaptive)
			}
			assertAdaptiveFile(t, path, s.data)
		})
	}
}

func TestAdaptivePatchConnectionsLowersCeiling(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 64<<20, 1<<20, 6<<20, 0, false)
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 15*time.Second, "growth to 5 connections", func() bool { return s.active.Load() >= 5 })

	if err := f.Patch(nil, &base.Options{Extra: &http.OptsExtra{Connections: 2}}); err != nil {
		t.Fatal(err)
	}
	if got := f.meta.Opts.Extra.(*http.OptsExtra).Connections; got != 2 {
		t.Fatalf("connections option = %d after Patch, want 2", got)
	}
	tl := sampleConnections(s)
	waitFor(t, 5*time.Second, "the lower ceiling", func() bool {
		recent := tl.snapshot()
		return len(recent) >= 10 && settled(recent, len(recent)-10, len(recent)) <= 2
	})
	s.totalCap.Store(0)
	if err := waitDone(t, f, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	samples := tl.finish()
	for i, n := range samples[len(samples)/2:] {
		if n > 2 {
			t.Fatalf("sample %d shows %d connections after the ceiling became 2", i, n)
		}
	}
	assertAdaptiveFile(t, path, s.data)
}

func TestAdaptivePatchWithoutOptionsKeepsConnections(t *testing.T) {
	f := buildConfigFetcher(config{Connections: 8}).(*Fetcher)
	f.meta.Opts = &base.Options{Extra: &http.OptsExtra{Connections: 8}}
	if err := f.Patch(nil, &base.Options{Extra: &http.OptsExtra{}}); err != nil {
		t.Fatal(err)
	}
	if got := f.meta.Opts.Extra.(*http.OptsExtra).Connections; got != 8 {
		t.Fatalf("connections = %d after an empty Patch, want 8", got)
	}
}

// An origin that advertises Range but ignores it: the first request falls back
// to a sequential download, and the adaptive loop hands over to slow start.
func TestAdaptiveIntegrationIgnoredRangeHandsOverToSequential(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 4<<20, 8<<20, 0, 0, false)
	s.ignoreRange = true
	f, path := newAdaptiveFetcher(t, config{Connections: 16, Adaptive: true}, s, nil)
	if !f.Meta().Res.Range {
		t.Fatal("setup: the server must advertise Range")
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitDone(t, f, 30*time.Second); err != nil {
		t.Fatalf("an ignored Range must fall back without error: %v", err)
	}
	if f.adaptive == nil {
		t.Fatal("setup: adaptive must have started before the fallback")
	}
	if f.Meta().Res.Range {
		t.Fatal("the fetcher did not fall back to a sequential download")
	}
	if n := len(f.Stats().Snapshot.(*http.Stats).Connections); n != 1 {
		t.Fatalf("got %d connections, want 1", n)
	}
	assertAdaptiveFile(t, path, s.data)
}

// The task switch turns adaptive on although the global default is off.
func TestAdaptiveTaskOnOverridesGlobalOff(t *testing.T) {
	useFastAdaptiveClock(t)
	s := newAdaptiveTestServer(t, 8<<20, 4<<20, 0, 0, false)
	f, path := newAdaptiveFetcher(t, config{Connections: 16}, s, &http.OptsExtra{Adaptive: boolPtr(true)})
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitDone(t, f, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if f.adaptive == nil {
		t.Fatal("the task switch did not turn adaptive on")
	}
	if got := f.slowStart.batchHistory; fmt.Sprint(got) != "[1]" {
		t.Fatalf("slow-start batches = %v, want only the first connection's book entry", got)
	}
	assertAdaptiveFile(t, path, s.data)
}
