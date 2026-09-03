package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/httpclient"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/xiaoqidun/setft"
)

const (
	connectTimeout     = 15 * time.Second
	readTimeout        = 15 * time.Second
	minFastFailTimeout = int64(3 * time.Second) // Minimum timeout for fast-fail retry

	// Work stealing parameters
	// When a connection finishes its chunk, it can "steal" work from slow connections.
	stealThresholdSeconds = 3          // Only steal if victim needs > 3 seconds to finish
	stealMinChunkSize     = 512 * 1024 // Min steal size: 512KB (avoid tiny chunks)
)

var (
	errRangeRequestIgnored  = errors.New("server ignored HTTP range request")
	errInvalidRangeResponse = errors.New("invalid HTTP range response")
	errIncompleteResponse   = errors.New("incomplete HTTP response")
)

func isRangeIntegrityError(err error) bool {
	return errors.Is(err, errRangeRequestIgnored) || errors.Is(err, errInvalidRangeResponse)
}

func isTerminalRangeError(err error) bool {
	return isRangeIntegrityError(err)
}

// ============================================================================
// State Machine
// ============================================================================

type fetcherState int32

const (
	stateIdle      fetcherState = iota // Initial state
	stateResolving                     // Resolving resource info
	stateResolved                      // Resolved, waiting for Start or downloading
	stateSlowStart                     // Slow-start phase: exponential connection growth
	stateSteady                        // Steady state: max connections reached
	statePaused                        // Paused
	stateDone                          // Completed
	stateError                         // Error occurred
)

// ============================================================================
// Connection
// ============================================================================

type connectionState int32

const (
	connNotStarted  connectionState = iota // Not yet started
	connConnecting                         // Sending HTTP request
	connDownloading                        // HTTP response OK, downloading
	connCompleted                          // Completed
	connFailed                             // Failed
)

type connectionRole int

const (
	roleResolve connectionRole = iota // Resolve connection: initial probe + temp download
	rolePrimary                       // Primary connection: first successful takeover from Resolve
	roleWorker                        // Worker connection: subsequent connections
)

type chunk struct {
	Begin      int64
	End        *int64 // nil until an open-ended Range response reveals the total size
	Downloaded int64
}

func (c *chunk) remain() int64 {
	return *c.End - c.Begin + 1 - c.Downloaded
}

func (c *chunk) openEnded() bool {
	return c.End == nil
}

func (c *chunk) setEnd(end int64) {
	c.End = &end
}

func newChunk(begin int64, end int64) *chunk {
	return &chunk{
		Begin: begin,
		End:   &end,
	}
}

func newOpenEndedChunk(begin int64) *chunk {
	return &chunk{
		Begin: begin,
	}
}

type connection struct {
	ID         int
	Role       connectionRole
	State      connectionState
	Chunk      *chunk
	Downloaded int64
	Completed  bool

	failed     bool
	exited     bool
	retryTimes int
	lastErr    error

	// Speed tracking for work stealing decisions
	speed             int64 // bytes per second
	lastSpeedCheck    int64 // timestamp in nanoseconds
	lastSpeedDownload int64 // bytes downloaded at last check

	ctx    context.Context
	cancel context.CancelFunc
}

// ============================================================================
// Slow Start Controller
// ============================================================================

type slowStartController struct {
	mu             sync.Mutex
	maxConnections int
	totalLaunched  int
	batchPending   int           // Connections in current batch waiting for HTTP response
	batchReady     int           // Connections in current batch that succeeded
	nextBatchSize  int           // Next batch size: 1, 2, 4, 8...
	expansionCh    chan struct{} // Signal to trigger next expansion
	paused         bool          // Pause expansion (e.g., on 429)
}

func newSlowStartController(maxConnections int) *slowStartController {
	return &slowStartController{
		maxConnections: maxConnections,
		nextBatchSize:  1,
		expansionCh:    make(chan struct{}, 1),
	}
}

// onConnectSuccess is called when a connection successfully gets HTTP response
// Returns true if this completes the current batch
func (s *slowStartController) onConnectSuccess() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.batchReady++
	if s.batchReady >= s.batchPending {
		// Batch complete, signal expansion
		select {
		case s.expansionCh <- struct{}{}:
		default:
		}
		return true
	}
	return false
}

// onConnectFailed is called when a connection fails
func (s *slowStartController) onConnectFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reduce pending count
	if s.batchPending > 0 {
		s.batchPending--
	}
	// If all pending resolved (success or fail), trigger expansion
	// This handles both successful completion and all-failures case
	if s.batchPending == 0 {
		select {
		case s.expansionCh <- struct{}{}:
		default:
		}
	}
}

// getNextBatchSize returns how many connections to start in next batch
// Returns 0 if max reached
func (s *slowStartController) getNextBatchSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.paused {
		return 0
	}

	remaining := s.maxConnections - s.totalLaunched
	if remaining <= 0 {
		return 0
	}

	batchSize := s.nextBatchSize
	if batchSize > remaining {
		batchSize = remaining
	}

	return batchSize
}

// commitBatch confirms that a batch of connections is being launched
func (s *slowStartController) commitBatch(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalLaunched += count
	s.nextBatchSize = s.nextBatchSize * 2 // Exponential growth: 1, 2, 4, 8...
	s.batchPending = count
	s.batchReady = 0
}

func (s *slowStartController) signalExpansion() {
	select {
	case s.expansionCh <- struct{}{}:
	default:
	}
}

// ============================================================================
// Fetcher
// ============================================================================

type Fetcher struct {
	ctl    *controller.Controller
	config *config
	doneCh chan error

	impersonationSession *httpclient.ImpersonationSession

	meta *fetcher.FetcherMeta

	// State machine
	state atomic.Int32 // fetcherState

	// Connections
	connMu      sync.Mutex
	connections []*connection
	resolveConn *connection // The special resolve connection

	// Slow start controller
	slowStart *slowStartController

	// Max connection time for adaptive timeout (stored as int64 nanoseconds for atomic ops)
	maxConnTime atomic.Int64

	// First primary connection success signal
	primaryReadyOnce sync.Once
	primaryReadyCh   chan struct{}

	// Start pending mechanism
	startPending   atomic.Bool
	resolvedCh     chan struct{} // Signal when resolve completes
	resolvedOnce   sync.Once
	resolveDataPos atomic.Int64 // How many bytes downloaded during resolve

	// Resolve response - kept open for one-time URLs
	resolveResp     *http.Response
	resolveRespLock sync.Mutex

	// Async prefetch during resolve phase
	prefetchFile     *os.File      // Temporary file for prefetch data
	prefetchFilePath string        // Path to temporary file
	prefetchSize     atomic.Int64  // Bytes prefetched so far
	prefetchDone     atomic.Bool   // Prefetch completed or stopped
	prefetchErr      error         // Error during prefetch (if any)
	prefetchStopCh   chan struct{} // Signal to stop prefetch
	resolveFallback  atomic.Bool   // Resolve prefetch is still available until the first Range request takes over

	// Target file
	file                  *os.File
	fileMu                sync.Mutex
	redirectURL           string
	redirectLock          sync.Mutex
	ifRange               string
	rangeReprobeEligible  bool
	rangeValidatorPinned  bool
	sequentialSizeUnknown bool

	// Lifecycle control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// downloadLoop lifecycle tracking
	downloadLoopDone chan struct{} // Closed when downloadLoop exits

	// Resolve connection control
	resolveCtx    context.Context
	resolveCancel context.CancelFunc
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	f.doneCh = make(chan error, 1)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	f.ctl.GetConfig(&f.config)
	if f.impersonationSession == nil {
		f.impersonationSession = httpclient.NewImpersonationSession()
	}
	f.resolvedCh = make(chan struct{})
	f.primaryReadyCh = make(chan struct{})

	// Check if this is a restore scenario (has existing connections or meta)
	if f.meta.Res != nil {
		// Already resolved, close the channel immediately
		close(f.resolvedCh)
		f.state.Store(int32(stateResolved))
	} else {
		f.state.Store(int32(stateIdle))
	}
}

func (f *Fetcher) getState() fetcherState {
	return fetcherState(f.state.Load())
}

func (f *Fetcher) setState(s fetcherState) {
	f.state.Store(int32(s))
}

// updateMaxConnTime updates maxConnTime if the new duration is larger
func (f *Fetcher) updateMaxConnTime(d time.Duration) {
	newVal := int64(d)
	if newVal > f.maxConnTime.Load() {
		f.maxConnTime.Store(newVal)
	}
}

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) error {
	if err := base.ParseReqExtra[fhttp.ReqExtra](req); err != nil {
		return err
	}
	f.meta.Req = req
	f.meta.Opts = opts
	if f.meta.Opts == nil {
		f.meta.Opts = &base.Options{}
	}

	// Parse options
	if err := base.ParseOptExtra[fhttp.OptsExtra](opts); err != nil {
		return err
	}
	if opts.Extra == nil {
		opts.Extra = &fhttp.OptsExtra{}
	}
	extra := opts.Extra.(*fhttp.OptsExtra)
	if extra.Connections <= 0 {
		extra.Connections = f.config.Connections
		if extra.Connections <= 0 {
			extra.Connections = 1
		}
	}

	f.setState(stateResolving)

	// Build HTTP request WITHOUT Range header (normal request)
	// This allows the response to be reused for downloading (important for one-time URLs)
	httpReq, err := f.buildRequest(context.TODO(), req)
	if err != nil {
		f.setState(stateError)
		return err
	}

	client := f.buildClient()

	// Send normal HTTP request (no Range header)
	// Track connection time for adaptive timeout in download phase
	connStartTime := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		f.setState(stateError)
		return err
	}
	// Record connection time as baseline for fast-fail timeout
	f.updateMaxConnTime(time.Since(connStartTime))

	// Parse response to get resource info
	res := &base.Resource{
		Range: false,
		Files: []*base.FileInfo{},
	}

	if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
		resp.Body.Close()
		f.setState(stateError)
		return NewRequestError(resp.StatusCode)
	}

	// Check if server supports range requests
	acceptRanges := resp.Header.Get(base.HttpHeaderAcceptRanges)
	contentRange := resp.Header.Get(base.HttpHeaderContentRange)
	if acceptRanges == base.HttpHeaderBytes || strings.HasPrefix(contentRange, base.HttpHeaderBytes) {
		res.Range = true
	}

	// Get content length from Content-Length header
	contentLength := resp.Header.Get(base.HttpHeaderContentLength)
	if contentLength != "" {
		parse, err := strconv.ParseInt(contentLength, 10, 64)
		if err == nil {
			res.Size = parse
		}
	}

	// Parse last modified time
	var lastModifiedTime *time.Time
	lastModified := resp.Header.Get(base.HttpHeaderLastModified)
	if lastModified != "" {
		if t, err := http.ParseTime(lastModified); err == nil {
			lastModifiedTime = &t
		}
	}
	f.ifRange = extractIfRangeValidator(resp.Header)

	file := &base.FileInfo{
		Size:  res.Size,
		Ctime: lastModifiedTime,
	}

	// Parse filename
	contentDisposition := resp.Header.Get(base.HttpHeaderContentDisposition)
	if contentDisposition != "" {
		file.Name = parseFilename(contentDisposition)
	}
	if file.Name == "" {
		file.Name = path.Base(httpReq.URL.Path)
		if file.Name != "" {
			// Use PathUnescape instead of QueryUnescape to correctly handle %2B (should decode to +, not space)
			file.Name, _ = url.PathUnescape(file.Name)
		}
	}
	if file.Name == "" || file.Name == "/" || file.Name == "." {
		file.Name = httpReq.URL.Hostname()
	}

	res.Files = append(res.Files, file)
	f.meta.Res = res

	// Save redirect URL for later connections
	f.redirectURL = resp.Request.URL.String()

	// IMPORTANT: Keep the response body open for downloading in Start phase
	// This is crucial for one-time URLs that can only be accessed once
	f.resolveRespLock.Lock()
	f.resolveResp = resp
	f.resolveRespLock.Unlock()

	f.setState(stateResolved)

	// Signal that resolve is complete
	f.resolvedOnce.Do(func() {
		close(f.resolvedCh)
	})

	// Start async prefetch in background (only for range-supported resources)
	// For non-range resources, the response will be used directly in Start
	if res.Range {
		f.prefetchStopCh = make(chan struct{})
		f.resolveFallback.Store(true)
		go f.asyncPrefetch()
	}

	// If start was called before resolve completed, auto-start
	if f.startPending.Load() {
		go f.doStart()
	}

	return nil
}

// asyncPrefetch downloads data in background during resolve phase
// This data can be reused when Start is called to save time
func (f *Fetcher) asyncPrefetch() {
	defer func() {
		f.prefetchDone.Store(true)
	}()

	// Get the resolve response
	f.resolveRespLock.Lock()
	resp := f.resolveResp
	f.resolveRespLock.Unlock()

	if resp == nil {
		return
	}

	// Create temporary file for prefetch data
	tmpFile, err := os.CreateTemp("", "gopeed-prefetch-*")
	if err != nil {
		f.prefetchErr = err
		return
	}
	f.prefetchFile = tmpFile
	f.prefetchFilePath = tmpFile.Name()

	defer func() {
		// Close response body when prefetch stops
		f.resolveRespLock.Lock()
		if f.resolveResp != nil {
			f.resolveResp.Body.Close()
			f.resolveResp = nil
		}
		f.resolveRespLock.Unlock()
	}()

	buf := make([]byte, 32*1024) // 32KB buffer
	reader := NewTimeoutReader(resp.Body, readTimeout)

	for {
		select {
		case <-f.prefetchStopCh:
			// Stop signal received (Start was called)
			return
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 {
			_, writeErr := tmpFile.Write(buf[:n])
			if writeErr != nil {
				f.prefetchErr = writeErr
				return
			}
			f.prefetchSize.Add(int64(n))
		}
		if err != nil {
			if err == io.EOF {
				// Prefetch completed
				return
			}
			f.prefetchErr = err
			return
		}
	}
}

// stopPrefetchAndGetData stops the async prefetch and returns prefetched bytes
// It also copies prefetched data to the target file
func (f *Fetcher) stopPrefetchAndCopyData() int64 {
	// Signal prefetch to stop (safely)
	if f.prefetchStopCh != nil {
		select {
		case <-f.prefetchStopCh:
			// Already closed
		default:
			close(f.prefetchStopCh)
		}
	}

	// Wait for prefetch to finish (with timeout)
	for i := 0; i < 1000 && !f.prefetchDone.Load(); i++ {
		time.Sleep(10 * time.Millisecond)
	}

	prefetched := f.prefetchSize.Load()
	if prefetched == 0 {
		f.cleanupPrefetchFile()
		return 0
	}

	// Copy prefetch data to target file
	if f.prefetchFile != nil && f.file != nil {
		// Seek to beginning of prefetch file
		f.prefetchFile.Seek(0, io.SeekStart)

		// Copy to target file at position 0
		buf := make([]byte, 32*1024)
		var copied int64
		for copied < prefetched {
			n, err := f.prefetchFile.Read(buf)
			if n > 0 {
				f.file.WriteAt(buf[:n], copied)
				copied += int64(n)
			}
			if err != nil {
				break
			}
		}
	}

	f.cleanupPrefetchFile()
	return prefetched
}

// cleanupPrefetchFile closes and removes the prefetch temporary file
func (f *Fetcher) cleanupPrefetchFile() {
	if f.prefetchFile != nil {
		f.prefetchFile.Close()
		f.prefetchFile = nil
	}
	if f.prefetchFilePath != "" {
		os.Remove(f.prefetchFilePath)
		f.prefetchFilePath = ""
	}
	// The byte count is only reusable while the backing prefetch file exists.
	// Pause may discard that file before the first Start, so retaining the
	// count would make the next Start skip bytes that were never copied.
	f.prefetchSize.Store(0)
	f.resolveFallback.Store(false)
}

func (f *Fetcher) Start() error {
	state := f.getState()

	switch state {
	case stateResolved, statePaused:
		// Normal case: resolved or resuming from pause
		return f.doStart()

	case stateResolving:
		// Early start: mark pending and return immediately
		f.startPending.Store(true)
		return nil

	case stateSlowStart, stateSteady:
		// Already downloading, this is a resume from pause
		return f.doStart()

	case stateError:
		// Retry after error: reset and restart
		return f.doStart()

	default:
		return fmt.Errorf("cannot start in current state: %v", state)
	}
}

func (f *Fetcher) doStart() error {
	// Wait for resolve to complete
	<-f.resolvedCh

	state := f.getState()
	if state == stateDone {
		return nil
	}

	// Each Start/Wait cycle must observe only its own completion result.
	// A paused or failed cycle may have completed concurrently with Pause and
	// left a result buffered before its download loop fully stopped.
	if state == statePaused || state == stateError {
		select {
		case <-f.doneCh:
		default:
		}
	}

	// If retrying after error, reset connection states for retry
	if state == stateError {
		f.connMu.Lock()
		for _, conn := range f.connections {
			// Reset connections that can be retried
			if !conn.Completed && conn.State != connCompleted {
				if !f.hasSequentialPrefixLocked(conn) {
					f.resetConnectionForRestart(conn)
				}
				conn.State = connNotStarted
				conn.failed = false
				conn.retryTimes = 0
				conn.lastErr = nil
			}
		}
		f.connMu.Unlock()
	}

	// Open or create target file first (needed for prefetch copy)
	name := f.meta.SingleFilepath()
	var err error
	var file *os.File
	_, err = os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = f.ctl.Touch(name, f.meta.Res.Size)
		} else {
			return err
		}
	} else {
		file, err = os.OpenFile(name, os.O_RDWR, os.ModeAppend)
	}
	if err != nil {
		return err
	}
	f.fileMu.Lock()
	f.file = file
	f.fileMu.Unlock()

	// For range-supported resources, stop prefetch and copy data
	// For non-range resources, the response will be used directly
	var prefetchedBytes int64
	if f.meta.Res.Range {
		resolveComplete := f.meta.Res.Size > 0 && f.prefetchDone.Load() && f.prefetchErr == nil && f.prefetchSize.Load() >= f.meta.Res.Size
		if f.resolveFallback.Load() && !resolveComplete {
			// Keep Resolve downloading until the first Range response has been
			// validated. The Range request starts at a stable snapshot; a valid
			// takeover discards any overlap downloaded by Resolve in the meantime.
			prefetchedBytes = f.prefetchSize.Load()
		} else {
			// Resolve already completed, or no live Resolve fallback remains.
			prefetchedBytes = f.stopPrefetchAndCopyData()
		}
		f.resolveDataPos.Store(prefetchedBytes)

		if !f.resolveFallback.Load() {
			// Also close resolve response if still open.
			f.resolveRespLock.Lock()
			if f.resolveResp != nil {
				f.resolveResp.Body.Close()
				f.resolveResp = nil
			}
			f.resolveRespLock.Unlock()
		}
	}

	// Avoid request extra modified by extension
	if err = base.ParseReqExtra[fhttp.ReqExtra](f.meta.Req); err != nil {
		return err
	}

	// Initialize slow start controller
	maxConns := f.meta.Opts.Extra.(*fhttp.OptsExtra).Connections
	f.slowStart = newSlowStartController(maxConns)

	// Create main context
	f.ctx, f.cancel = context.WithCancel(context.Background())

	// Create downloadLoop lifecycle channel
	downloadLoopDone := make(chan struct{})
	f.downloadLoopDone = downloadLoopDone

	// Start download
	f.setState(stateSlowStart)
	go f.downloadLoop(downloadLoopDone)

	return nil
}

func (f *Fetcher) downloadLoop(done chan struct{}) {
	ctx := f.ctx

	defer func() {
		// Update file last modified time before closing
		if f.config.UseServerCtime && f.meta.Res.Files[0].Ctime != nil {
			setft.SetFileTime(f.meta.SingleFilepath(), time.Now(), *f.meta.Res.Files[0].Ctime, *f.meta.Res.Files[0].Ctime)
		}

		// Signal that downloadLoop has exited
		if done != nil {
			close(done)
		}
	}()

	// Check if this is a resume or fresh start
	isResume := len(f.connections) > 0

	if !isResume {
		// Capture the mode before launching the first connection. The server may
		// ignore that connection's Range request and switch the fetcher to a
		// sequential download while the request is in flight.
		startedWithRanges := f.meta.Res.Range
		// Fresh start: begin with resolve connection
		f.startResolveDownload()
		// Non-range downloads wait for their only connection in
		// startResolveDownload. Falling through would consume the connection's
		// expansion signal and publish the same completion a second time.
		if !startedWithRanges || f.getState() == stateDone {
			return
		}
	} else {
		// A retained sequential prefix starts as the first slow-start batch. If
		// its guarded request restores Range support, the normal expansion loop
		// can split the remaining suffix across the configured connections.
		canRecoverRange := f.hasSequentialRecoveryCandidate()
		if canRecoverRange {
			f.slowStart.commitBatch(1)
		}
		f.resumeConnections()
		if !canRecoverRange {
			f.waitForCompletion(ctx)
			return
		}
	}

	// Slow start loop
	for {
		select {
		case <-ctx.Done():
			// Paused or cancelled
			return
		case <-f.slowStart.expansionCh:
			f.connMu.Lock()
			rangeMode := f.meta.Res.Range
			sequentialExited := len(f.connections) == 1 && f.connections[0].exited
			f.connMu.Unlock()
			if !rangeMode {
				// The primary response is still being consumed sequentially. A
				// successful re-probe or final connection exit will signal again.
				if sequentialExited {
					f.waitForCompletion(ctx)
					return
				}
				continue
			}
			// Batch completed, try to expand
			if f.checkCompletion() {
				// All work is done, wait for connections to finish
				f.waitForCompletion(ctx)
				return
			}
			f.expandConnections()

			// Check if we've reached steady state (max connections)
			if f.getState() == stateSteady {
				// Wait for all connections to complete
				f.waitForCompletion(ctx)
				return
			}
		}
	}
}

func (f *Fetcher) startResolveDownload() {
	// Without Range support, consume the reusable Resolve response sequentially.
	// A Range-capable resource with an unknown size instead starts one open-ended
	// worker, which discovers the total from Content-Range before slow-start expands.
	if !f.meta.Res.Range {
		// Create a single connection for the entire file
		conn := &connection{
			ID:    0,
			Role:  rolePrimary,
			State: connNotStarted,
			Chunk: newChunk(0, 0), // For non-range, end doesn't matter
		}
		conn.ctx, conn.cancel = context.WithCancel(f.ctx)
		f.connMu.Lock()
		f.connections = append(f.connections, conn)
		f.connMu.Unlock()

		f.wg.Add(1)
		// Use the resolve response directly
		go f.runConnectionWithResolveResp(conn)

		// For non-range downloads, wait for completion directly in this goroutine
		// Don't create another goroutine to avoid WaitGroup reuse issues
		f.waitForCompletion(f.ctx)
		return
	}

	// Range supported: use slow start to launch connections
	// Start first batch of connections
	f.expandConnections()
}

func (f *Fetcher) expandConnections() {
	batchSize := f.slowStart.getNextBatchSize()
	if batchSize <= 0 {
		// Max reached, transition to steady state
		f.setState(stateSteady)
		// Don't start a new goroutine - let the downloadLoop handle completion
		// This avoids multiple goroutines calling wg.Wait() simultaneously
		return
	}

	totalSize := f.meta.Res.Size

	f.connMu.Lock()

	// For first batch (no existing connections), allocate the remaining file to first connection
	if len(f.connections) == 0 {
		// Check if we have prefetched data
		prefetched := f.resolveDataPos.Load()

		// If prefetched all data, mark as done. A zero size means the extent is
		// still unknown, so the first worker must discover it with bytes=start-.
		if totalSize > 0 && prefetched >= totalSize {
			f.connMu.Unlock()

			// Close the file before signaling completion
			f.fileMu.Lock()
			if f.file != nil {
				f.file.Close()
				f.file = nil
			}
			f.fileMu.Unlock()

			f.setState(stateDone)
			f.doneCh <- nil
			return
		}

		// First connection starts from prefetched position
		connChunk := newOpenEndedChunk(prefetched)
		if totalSize > 0 {
			connChunk = newChunk(prefetched, totalSize-1)
		}
		conn := &connection{
			ID:    0,
			Role:  rolePrimary,
			State: connNotStarted,
			Chunk: connChunk,
		}
		// Mark prefetched bytes as already downloaded
		conn.Chunk.Downloaded = 0    // Start fresh from prefetched position
		conn.Downloaded = prefetched // Track total downloaded including prefetch

		conn.ctx, conn.cancel = context.WithCancel(f.ctx)
		f.connections = append(f.connections, conn)
		f.connMu.Unlock()

		f.slowStart.commitBatch(1)
		f.wg.Add(1)
		go f.runConnection(conn)
		return
	}

	// For subsequent batches, use "help other connection" strategy
	// Find connections with enough remaining work to split
	// During slow start, use fixed minimum size since speed is not yet stable
	minSplitSize := int64(stealMinChunkSize)

	newConns := make([]*connection, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		// Find the connection with most remaining work
		var maxRemainConn *connection
		var maxRemain int64

		for _, conn := range f.connections {
			if conn.Completed || conn.State == connFailed {
				continue
			}
			if conn.Chunk.openEnded() {
				continue
			}
			remain := conn.Chunk.remain()
			// Only split if remaining work is at least 2x the minimum split size
			if remain > maxRemain && remain > minSplitSize*2 {
				maxRemainConn = conn
				maxRemain = remain
			}
		}

		if maxRemainConn == nil {
			// No connection has enough work to split
			break
		}

		// Split the work: new connection takes the latter half
		chunkEnd := *maxRemainConn.Chunk.End
		splitPoint := chunkEnd - maxRemainConn.Chunk.remain()/2
		newChunk := newChunk(splitPoint+1, chunkEnd)
		maxRemainConn.Chunk.setEnd(splitPoint)

		connID := len(f.connections)
		conn := &connection{
			ID:    connID,
			Role:  roleWorker,
			State: connNotStarted,
			Chunk: newChunk,
		}
		conn.ctx, conn.cancel = context.WithCancel(f.ctx)

		newConns = append(newConns, conn)
		f.connections = append(f.connections, conn)
	}

	f.connMu.Unlock()

	if len(newConns) == 0 {
		// No new connections could be created, stop expansion
		f.setState(stateSteady)
		return
	}

	// Commit batch to slow start controller
	f.slowStart.commitBatch(len(newConns))

	// Launch connections
	for _, conn := range newConns {
		f.wg.Add(1)
		go f.runConnection(conn)
	}
}

func (f *Fetcher) runConnection(conn *connection) {
	defer func() {
		// A range-advertising origin can fall back to one sequential response.
		// Wake the download loop after that connection exits because its earlier
		// response-ready signal was intentionally not used to expand.
		f.connMu.Lock()
		conn.exited = true
		sequentialRecovery := !f.meta.Res.Range && (f.rangeReprobeEligible || conn.Completed) && len(f.connections) == 1
		f.connMu.Unlock()
		f.wg.Done()
		if sequentialRecovery && f.slowStart != nil {
			f.slowStart.signalExpansion()
		}
	}()

	f.connMu.Lock()
	conn.exited = false
	conn.State = connConnecting
	f.connMu.Unlock()

	// Use fast-fail client for quick retry during download phase
	client := f.buildFastFailClient()
	buf := make([]byte, 8192)

	retries := 0
	f.connMu.Lock()
	conn.retryTimes = 0
	f.connMu.Unlock()

	for {
		// Rebuild client with updated fast-fail timeout on retries
		if retries > 0 {
			client = f.buildFastFailClient()

			// Preserve an eligible sequential prefix long enough to probe whether
			// the origin has recovered byte-range support. downloadChunkOnce resets
			// it only when no safe validator exists or If-Range returns a full 200.
			f.connMu.Lock()
			if !f.hasSequentialPrefixLocked(conn) {
				f.resetConnectionForRestart(conn)
			}
			f.connMu.Unlock()
		}

		err := f.downloadChunkOnce(conn, client, buf)
		if err == nil {
			if !f.meta.Res.Range || !f.helpOtherConnection(conn) {
				f.connMu.Lock()
				conn.Completed = true
				conn.State = connCompleted
				f.connMu.Unlock()
				return
			}

			// Reset counters after a successful help switch
			retries = 0
			f.connMu.Lock()
			conn.retryTimes = 0
			f.connMu.Unlock()
			continue
		}

		if errors.Is(err, context.Canceled) {
			return
		}
		requestErr := extractRequestError(err)
		shouldUseResolveFallback := requestErr == nil || !isFailureExemptHTTPCode(requestErr.Code)
		if shouldUseResolveFallback && f.completeFromResolveFallback(conn) {
			return
		}
		if isTerminalRangeError(err) {
			f.connMu.Lock()
			conn.lastErr = err
			conn.State = connFailed
			conn.failed = true
			f.connMu.Unlock()
			if f.slowStart != nil {
				f.slowStart.onConnectFailed()
			}
			return
		}

		f.connMu.Lock()
		if re := extractRequestError(err); re != nil {
			conn.lastErr = re
		} else {
			conn.lastErr = err
		}
		f.connMu.Unlock()

		if shouldCountHTTPFailure(err) {
			if re := extractRequestError(err); re != nil && re.Code == 403 {
				f.connMu.Lock()
				conn.State = connFailed
				conn.failed = true
				f.connMu.Unlock()
				if f.slowStart != nil {
					f.slowStart.onConnectFailed()
				}
				return
			}
			f.connMu.Lock()
			conn.retryTimes++
			retryTimes := conn.retryTimes
			conn.failed = true
			f.connMu.Unlock()
			if f.slowStart != nil {
				f.slowStart.onConnectFailed()
			}
			if retryTimes >= 3 {
				f.connMu.Lock()
				conn.State = connFailed
				f.connMu.Unlock()
				return
			}
		}

		f.connMu.Lock()
		conn.State = connFailed
		f.connMu.Unlock()
		retryDelay := time.Second * time.Duration(retries+1)
		if retryDelay > 5*time.Second {
			retryDelay = 5 * time.Second
		}
		retries++
		time.Sleep(retryDelay)
	}
}

// downloadChunkOnce performs a single HTTP request for the current chunk without retrying.
// If the redirect URL fails with an expiration-related error (401, 403, 410),
// it will automatically retry with the original URL and update the redirect URL on success.
func (f *Fetcher) downloadChunkOnce(conn *connection, client *http.Client, buf []byte) error {
	if conn.ctx.Err() != nil {
		return conn.ctx.Err()
	}

	// Read chunk boundaries under lock to get a consistent snapshot. A prefix
	// without a safe validator cannot be resumed, so restart it before building
	// the request; the returned full response will overwrite from byte zero.
	f.connMu.Lock()
	sequentialSizeUnknown := f.sequentialSizeUnknown
	hasSequentialPrefix := f.hasSequentialPrefixLocked(conn)
	resumeProbe := f.canProbeSequentialResumeLocked(conn)
	intentionalRestart := sequentialSizeUnknown
	if sequentialSizeUnknown || (hasSequentialPrefix && !resumeProbe) {
		f.resetConnectionForRestart(conn)
		f.resolveDataPos.Store(0)
		f.rangeValidatorPinned = false
		intentionalRestart = true
	}
	openEndedRange := f.meta.Res.Range && conn.Chunk.openEnded()
	if f.meta.Res.Range && !openEndedRange && conn.Chunk.remain() <= 0 {
		f.connMu.Unlock()
		return nil
	}
	rangeStart := conn.Chunk.Begin + conn.Chunk.Downloaded
	var rangeEnd *int64
	if !openEndedRange {
		end := *conn.Chunk.End
		rangeEnd = &end
	}
	rangeMode := f.meta.Res.Range
	ifRange := ""
	if resumeProbe {
		end := f.meta.Res.Size - 1
		rangeEnd = &end
		ifRange = f.ifRange
	} else if rangeMode && (f.rangeValidatorPinned || f.resolveFallback.Load()) {
		ifRange = f.ifRange
	}
	f.connMu.Unlock()

	httpReq, err := f.buildRequest(conn.ctx, f.meta.Req)
	if err != nil {
		return err
	}

	if rangeMode || resumeProbe {
		httpReq.Header.Set(base.HttpHeaderRange, formatRangeHeader(rangeStart, rangeEnd))
		if ifRange != "" {
			httpReq.Header.Set(base.HttpHeaderIfRange, ifRange)
		}
	}
	rangeRequested := httpReq.Header.Get(base.HttpHeaderRange) != ""

	// Record connection start time for adaptive timeout tracking
	connStartTime := time.Now()

	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}

	if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
		resp.Body.Close()
		originalErr := NewRequestError(resp.StatusCode)

		// Check if this might be a redirect URL expiration error
		// If so, try falling back to the original URL
		if f.hasRedirectURL() && isRedirectExpiredError(originalErr) {
			fallbackResp, fallbackErr := f.tryFallbackToOriginalURL(conn.ctx, client, rangeStart, rangeEnd, rangeRequested, ifRange)
			if fallbackErr == nil && fallbackResp != nil {
				// Fallback succeeded, use this response instead
				resp = fallbackResp
				// Update the redirect URL from the response
				if resp.Request != nil && resp.Request.URL != nil {
					f.updateRedirectURL(resp.Request.URL.String())
				}
			} else {
				// Fallback also failed, return the original error
				if fallbackResp != nil {
					fallbackResp.Body.Close()
				}
				return originalErr
			}
		} else {
			return originalErr
		}
	}

	expectedResponseLength := int64(-1)
	var responseBytesToDiscard int64
	if rangeRequested && resp.StatusCode == base.HttpCodeOK {
		if conn.ID == 0 && f.resolveFallback.CompareAndSwap(true, false) {
			// This response is itself a complete representation, so the older
			// Resolve stream is no longer needed. Stop and copy its prefix before
			// the sequential restart truncates/overwrites the target from zero.
			f.stopPrefetchAndCopyData()
		}
		if resumeProbe {
			// If-Range returning 200 provides the complete current representation.
			// Discard the old prefix counters and consume this response from byte 0.
			f.restartSequentialDownload(conn, extractIfRangeValidator(resp.Header))
			intentionalRestart = true
		} else {
			if err := f.fallbackToSequentialDownload(conn, extractIfRangeValidator(resp.Header)); err != nil {
				resp.Body.Close()
				return fmt.Errorf("%w: expected 206 Partial Content, got 200 OK", err)
			}
			intentionalRestart = true
		}
	}
	if rangeRequested && resp.StatusCode == base.HttpCodePartialContent {
		var responseTotal int64
		expectedResponseLength, responseTotal, err = validateRangeResponse(resp, rangeStart, rangeEnd, f.meta.Res.Size)
		if err != nil {
			resp.Body.Close()
			return err
		}
		if openEndedRange {
			if err := f.resolveOpenEndedRange(conn, responseTotal); err != nil {
				resp.Body.Close()
				return err
			}
		}
		responseBytesToDiscard, err = f.commitResolveTakeover(conn, rangeStart)
		if err != nil {
			resp.Body.Close()
			return err
		}
		if resumeProbe {
			// The origin recovered Range support. Convert the preserved sequential
			// prefix into a normal ranged chunk before writing the suffix.
			f.connMu.Lock()
			f.meta.Res.Range = true
			f.rangeReprobeEligible = false
			f.rangeValidatorPinned = true
			f.sequentialSizeUnknown = false
			conn.Chunk.Begin = 0
			conn.Chunk.setEnd(f.meta.Res.Size - 1)
			f.connMu.Unlock()
		}
	}
	if !rangeRequested && resp.StatusCode == base.HttpCodeOK {
		f.connMu.Lock()
		if f.rangeReprobeEligible {
			// This full response might itself become the retained prefix if its
			// body is interrupted, so bind any later If-Range probe to it.
			f.ifRange = extractIfRangeValidator(resp.Header)
		}
		f.connMu.Unlock()
	}
	if !f.meta.Res.Range && resp.StatusCode == base.HttpCodeOK {
		if intentionalRestart {
			if resp.ContentLength >= 0 {
				if err := f.resizeSequentialResource(resp.ContentLength); err != nil {
					resp.Body.Close()
					return err
				}
				expectedResponseLength = resp.ContentLength
			} else if err := f.beginUnknownSizeSequentialResource(); err != nil {
				resp.Body.Close()
				return err
			}
		} else if f.meta.Res.Size > 0 {
			expectedResponseLength = f.meta.Res.Size
			if resp.ContentLength >= 0 && resp.ContentLength != expectedResponseLength {
				resp.Body.Close()
				return fmt.Errorf("%w: Content-Length %d does not match resource size %d", errInvalidRangeResponse, resp.ContentLength, expectedResponseLength)
			}
		}
	}
	defer resp.Body.Close()

	// Record successful connection time for adaptive timeout
	f.updateMaxConnTime(time.Since(connStartTime))

	f.connMu.Lock()
	conn.State = connDownloading
	conn.failed = false
	f.connMu.Unlock()

	if conn.Role == rolePrimary || conn.ID == 0 {
		f.primaryReadyOnce.Do(func() {
			close(f.primaryReadyCh)
		})
	}
	if f.slowStart != nil {
		f.slowStart.onConnectSuccess()
	}

	reader := NewTimeoutReader(resp.Body, readTimeout)
	var responseBytesRead int64
	for {
		if conn.ctx.Err() != nil {
			return conn.ctx.Err()
		}

		n, err := reader.Read(buf)
		if n > 0 {
			responseBytesRead += int64(n)
			if expectedResponseLength >= 0 && responseBytesRead > expectedResponseLength {
				return fmt.Errorf("%w: response body exceeds expected length %d", errInvalidRangeResponse, expectedResponseLength)
			}

			data := buf[:n]
			if responseBytesToDiscard > 0 {
				discard := min(int64(len(data)), responseBytesToDiscard)
				data = data[discard:]
				responseBytesToDiscard -= discard
			}

			finished := false
			var writeOffset int64

			if len(data) > 0 {
				// Lock to safely read chunk state and calculate write parameters
				// This protects against concurrent chunk splitting by helpOtherConnection
				f.connMu.Lock()
				if f.meta.Res.Range {
					// Check current chunk boundaries - this respects any concurrent chunk splitting
					remain := conn.Chunk.remain()
					if remain <= 0 {
						// Chunk has been fully downloaded (possibly split and reduced)
						f.connMu.Unlock()
						return nil
					}
					if remain < int64(len(data)) {
						data = data[:remain]
						finished = true
					}
				}
				writeOffset = conn.Chunk.Begin + conn.Chunk.Downloaded
				f.connMu.Unlock()

				f.fileMu.Lock()
				if f.file != nil {
					_, writeErr := f.file.WriteAt(data, writeOffset)
					if writeErr != nil {
						f.fileMu.Unlock()
						return writeErr
					}
				}
				f.fileMu.Unlock()

				// Lock again to update Downloaded atomically with the read above
				f.connMu.Lock()
				conn.Chunk.Downloaded += int64(len(data))
				conn.Downloaded += int64(len(data))
				// Update connection speed periodically
				now := time.Now().UnixNano()
				if conn.lastSpeedCheck == 0 {
					conn.lastSpeedCheck = now
					conn.lastSpeedDownload = conn.Downloaded
				} else if now-conn.lastSpeedCheck >= int64(500*time.Millisecond) {
					elapsed := float64(now-conn.lastSpeedCheck) / float64(time.Second)
					if elapsed > 0 {
						conn.speed = int64(float64(conn.Downloaded-conn.lastSpeedDownload) / elapsed)
					}
					conn.lastSpeedCheck = now
					conn.lastSpeedDownload = conn.Downloaded
				}
				f.connMu.Unlock()

				if finished {
					return nil
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				if intentionalRestart && resp.ContentLength < 0 {
					if err := f.resizeSequentialResource(responseBytesRead); err != nil {
						return err
					}
				}
				if expectedResponseLength >= 0 && responseBytesRead != expectedResponseLength {
					if rangeRequested && responseBytesRead < expectedResponseLength {
						return fmt.Errorf("%w: response body length %d, expected %d", errIncompleteResponse, responseBytesRead, expectedResponseLength)
					}
					return fmt.Errorf("%w: response body length %d, expected %d", errInvalidRangeResponse, responseBytesRead, expectedResponseLength)
				}
				return nil
			}
			return err
		}
	}
}

// commitResolveTakeover stops the still-live Resolve download only after the
// first Range response has been validated. Resolve may have advanced beyond
// requestedStart while the speculative request was in flight, so the overlap
// is returned for the caller to discard from the Range response body.
func (f *Fetcher) commitResolveTakeover(conn *connection, requestedStart int64) (int64, error) {
	if conn.ID != 0 || !f.resolveFallback.CompareAndSwap(true, false) {
		return 0, nil
	}

	prefetched := f.stopPrefetchAndCopyData()
	if prefetched < requestedStart || prefetched > f.meta.Res.Size {
		return 0, fmt.Errorf("resolve prefetch advanced to invalid offset %d for range start %d", prefetched, requestedStart)
	}
	f.resolveDataPos.Store(prefetched)

	overlap := prefetched - requestedStart
	f.connMu.Lock()
	if f.ifRange != "" {
		f.rangeValidatorPinned = true
	}
	conn.Chunk.Downloaded = overlap
	conn.Downloaded = prefetched
	f.connMu.Unlock()
	return overlap, nil
}

// completeFromResolveFallback keeps the original full response authoritative
// when the speculative first Range request cannot take over. It waits for that
// response to finish, copies it once, and converts the task to one completed
// sequential connection instead of accepting an incomplete Range result.
func (f *Fetcher) completeFromResolveFallback(conn *connection) bool {
	if conn.ID != 0 || !f.resolveFallback.CompareAndSwap(true, false) {
		return false
	}

	for !f.prefetchDone.Load() {
		if conn.ctx.Err() != nil {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
	knownSize := f.meta.Res.Size > 0
	if f.prefetchErr != nil || (knownSize && f.prefetchSize.Load() != f.meta.Res.Size) {
		f.cleanupPrefetchFile()
		return false
	}

	prefetched := f.stopPrefetchAndCopyData()
	if knownSize && prefetched != f.meta.Res.Size {
		return false
	}
	f.resolveDataPos.Store(prefetched)

	f.connMu.Lock()
	if !knownSize {
		f.meta.Res.Size = prefetched
		if len(f.meta.Res.Files) > 0 && f.meta.Res.Files[0] != nil {
			f.meta.Res.Files[0].Size = prefetched
		}
	}
	f.meta.Res.Range = false
	f.rangeReprobeEligible = false
	f.rangeValidatorPinned = false
	conn.Chunk = newChunk(0, prefetched-1)
	conn.Chunk.Downloaded = prefetched
	conn.Downloaded = prefetched
	conn.Completed = true
	conn.State = connCompleted
	conn.failed = false
	conn.lastErr = nil
	f.connMu.Unlock()
	return true
}

func validateRangeResponse(resp *http.Response, requestedStart int64, requestedEnd *int64, totalSize int64) (int64, int64, error) {
	contentRange := resp.Header.Get(base.HttpHeaderContentRange)
	fields := strings.Fields(contentRange)
	if len(fields) != 2 || !strings.EqualFold(fields[0], base.HttpHeaderBytes) {
		return 0, 0, fmt.Errorf("%w: malformed Content-Range %q", errInvalidRangeResponse, contentRange)
	}

	rangeAndTotal := strings.Split(fields[1], "/")
	if len(rangeAndTotal) != 2 {
		return 0, 0, fmt.Errorf("%w: malformed Content-Range %q", errInvalidRangeResponse, contentRange)
	}
	bounds := strings.Split(rangeAndTotal[0], "-")
	if len(bounds) != 2 {
		return 0, 0, fmt.Errorf("%w: malformed Content-Range %q", errInvalidRangeResponse, contentRange)
	}

	start, startErr := strconv.ParseInt(bounds[0], 10, 64)
	end, endErr := strconv.ParseInt(bounds[1], 10, 64)
	total, totalErr := strconv.ParseInt(rangeAndTotal[1], 10, 64)
	if startErr != nil || endErr != nil || totalErr != nil || start < 0 || end < start || total <= end {
		return 0, 0, fmt.Errorf("%w: malformed Content-Range %q", errInvalidRangeResponse, contentRange)
	}
	if start != requestedStart || (requestedEnd != nil && end != *requestedEnd) || (requestedEnd == nil && end != total-1) {
		return 0, 0, fmt.Errorf("%w: Content-Range %d-%d does not match requested range", errInvalidRangeResponse, start, end)
	}
	if totalSize > 0 && total != totalSize {
		return 0, 0, fmt.Errorf("%w: Content-Range total %d does not match resource size %d", errInvalidRangeResponse, total, totalSize)
	}

	expectedLength := end - start + 1
	if resp.ContentLength >= 0 && resp.ContentLength != expectedLength {
		return 0, 0, fmt.Errorf("%w: Content-Length %d does not match range length %d", errInvalidRangeResponse, resp.ContentLength, expectedLength)
	}
	return expectedLength, total, nil
}

func (f *Fetcher) resolveOpenEndedRange(conn *connection, totalSize int64) error {
	if totalSize <= 0 {
		return fmt.Errorf("%w: invalid discovered resource size %d", errInvalidRangeResponse, totalSize)
	}

	// Publish the discovered extent before the response-ready signal can expand
	// the slow-start batch. Store observes the same metadata/progress snapshot.
	f.connMu.Lock()
	f.fileMu.Lock()
	if f.file == nil {
		f.fileMu.Unlock()
		f.connMu.Unlock()
		return errors.New("target file is not open")
	}
	err := f.file.Truncate(totalSize)
	f.fileMu.Unlock()
	if err != nil {
		f.connMu.Unlock()
		return err
	}
	f.meta.Res.Size = totalSize
	if len(f.meta.Res.Files) > 0 && f.meta.Res.Files[0] != nil {
		f.meta.Res.Files[0].Size = totalSize
	}
	conn.Chunk.setEnd(totalSize - 1)
	f.connMu.Unlock()
	return nil
}

// fallbackToSequentialDownload handles origins that advertise byte ranges but
// ignore the first Range request and return the full representation with 200.
// Slow start launches only the primary connection initially, so it is safe to
// reset the prefetch offset and consume that response from byte zero. If the
// server changes behavior after parallel connections have started, fail rather
// than risk assembling a corrupt file.
func (f *Fetcher) fallbackToSequentialDownload(conn *connection, ifRange string) error {
	f.connMu.Lock()
	defer f.connMu.Unlock()

	if !f.meta.Res.Range {
		return nil
	}
	if conn.ID != 0 || len(f.connections) != 1 {
		return errRangeRequestIgnored
	}

	f.meta.Res.Range = false
	f.rangeReprobeEligible = true
	f.rangeValidatorPinned = false
	f.ifRange = ifRange
	f.resetConnectionForRestart(conn)
	f.resolveDataPos.Store(0)
	return nil
}

// restartSequentialDownload consumes an If-Range 200 response as the complete
// current representation. The old prefix is no longer trusted, but this body
// can safely overwrite it from byte zero.
func (f *Fetcher) restartSequentialDownload(conn *connection, ifRange string) {
	f.connMu.Lock()
	defer f.connMu.Unlock()

	f.meta.Res.Range = false
	f.rangeReprobeEligible = true
	f.rangeValidatorPinned = false
	f.ifRange = ifRange
	f.resetConnectionForRestart(conn)
	f.resolveDataPos.Store(0)
}

// resizeSequentialResource makes a full restart response authoritative for
// both progress accounting and the target file. Truncation removes stale tail
// bytes when the new representation is shorter and preallocates when longer.
func (f *Fetcher) resizeSequentialResource(size int64) error {
	if size < 0 {
		return fmt.Errorf("invalid sequential response size %d", size)
	}

	// Keep the file mutation and its recovery metadata in one connMu critical
	// section so Store cannot persist one side of the transition without the
	// other. Other download paths never hold fileMu while acquiring connMu.
	f.connMu.Lock()
	f.fileMu.Lock()
	if f.file == nil {
		f.fileMu.Unlock()
		f.connMu.Unlock()
		return errors.New("target file is not open")
	}
	err := f.file.Truncate(size)
	f.fileMu.Unlock()
	if err != nil {
		f.connMu.Unlock()
		return err
	}

	f.meta.Res.Size = size
	if len(f.meta.Res.Files) > 0 {
		f.meta.Res.Files[0].Size = size
	}
	f.sequentialSizeUnknown = false
	f.connMu.Unlock()
	return nil
}

// beginUnknownSizeSequentialResource discards all stale bytes and records that
// a restarted chunked response has no authoritative total size yet. If this
// response is interrupted, the persisted flag forces the next attempt to start
// from byte zero instead of validating a Range response against an old size.
func (f *Fetcher) beginUnknownSizeSequentialResource() error {
	f.connMu.Lock()
	f.fileMu.Lock()
	if f.file == nil {
		f.fileMu.Unlock()
		f.connMu.Unlock()
		return errors.New("target file is not open")
	}
	err := f.file.Truncate(0)
	f.fileMu.Unlock()
	if err != nil {
		f.connMu.Unlock()
		return err
	}

	f.meta.Res.Size = 0
	if len(f.meta.Res.Files) > 0 {
		f.meta.Res.Files[0].Size = 0
	}
	f.sequentialSizeUnknown = true
	f.ifRange = ""
	f.rangeValidatorPinned = false
	f.connMu.Unlock()
	return nil
}

// runConnectionWithResolveResp uses the response body from Resolve phase
// This is crucial for one-time URLs that can only be accessed once
func (f *Fetcher) runConnectionWithResolveResp(conn *connection) {
	defer f.wg.Done()

	f.connMu.Lock()
	conn.State = connConnecting
	f.connMu.Unlock()

	buf := make([]byte, 8192)

	// Get the resolve response
	f.resolveRespLock.Lock()
	resp := f.resolveResp
	f.resolveResp = nil // Take ownership
	f.resolveRespLock.Unlock()

	if resp == nil {
		// No resolve response available, fall back to normal connection
		f.runConnectionFallback(conn)
		return
	}

	defer resp.Body.Close()

	f.connMu.Lock()
	conn.State = connDownloading
	conn.failed = false
	f.connMu.Unlock()

	// Signal primary ready
	f.primaryReadyOnce.Do(func() {
		close(f.primaryReadyCh)
	})
	if f.slowStart != nil {
		f.slowStart.onConnectSuccess()
	}

	// Download data from resolve response
	reader := NewTimeoutReader(resp.Body, readTimeout)
	for {
		if conn.ctx.Err() != nil {
			return
		}

		n, err := reader.Read(buf)
		if n > 0 {
			f.connMu.Lock()
			writeOffset := conn.Chunk.Downloaded
			f.connMu.Unlock()

			f.fileMu.Lock()
			if f.file != nil {
				_, writeErr := f.file.WriteAt(buf[:n], writeOffset)
				if writeErr != nil {
					f.fileMu.Unlock()
					f.connMu.Lock()
					conn.State = connFailed
					conn.failed = true
					f.connMu.Unlock()
					if f.slowStart != nil {
						f.slowStart.onConnectFailed()
					}
					return
				}
			}
			f.fileMu.Unlock()

			f.connMu.Lock()
			conn.Chunk.Downloaded += int64(n)
			conn.Downloaded += int64(n)
			f.connMu.Unlock()
		}
		if err != nil {
			if err == io.EOF {
				f.connMu.Lock()
				if f.meta.Res.Size == 0 {
					f.meta.Res.Size = conn.Downloaded
					if len(f.meta.Res.Files) > 0 && f.meta.Res.Files[0] != nil {
						f.meta.Res.Files[0].Size = conn.Downloaded
					}
				}
				conn.Completed = true
				conn.State = connCompleted
				f.connMu.Unlock()
				return
			}
			// The Resolve response cannot be resumed on this path. Preserve the read
			// error so an unknown-length stream is not mistaken for a clean EOF.
			f.connMu.Lock()
			conn.State = connFailed
			conn.failed = true
			conn.lastErr = err
			f.connMu.Unlock()
			return
		}
	}
}

// runConnectionFallback is used when resolve response is not available
func (f *Fetcher) runConnectionFallback(conn *connection) {
	// Use fast-fail client for quick retry during download phase
	client := f.buildFastFailClient()
	buf := make([]byte, 8192)

	retries := 0
	countedRetries := 0

	for {
		if conn.ctx.Err() != nil {
			return
		}

		// Rebuild client with updated fast-fail timeout on retries
		if retries > 0 {
			client = f.buildFastFailClient()
		}

		f.connMu.Lock()
		conn.State = connConnecting
		f.connMu.Unlock()

		err := func() error {
			httpReq, err := f.buildRequest(conn.ctx, f.meta.Req)
			if err != nil {
				return err
			}

			// Record connection start time for adaptive timeout tracking
			connStartTime := time.Now()

			resp, err := client.Do(httpReq)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
				return NewRequestError(resp.StatusCode)
			}

			// Record successful connection time for adaptive timeout
			f.updateMaxConnTime(time.Since(connStartTime))

			f.connMu.Lock()
			conn.State = connDownloading
			conn.failed = false
			f.connMu.Unlock()

			f.primaryReadyOnce.Do(func() {
				close(f.primaryReadyCh)
			})
			if f.slowStart != nil {
				f.slowStart.onConnectSuccess()
			}

			reader := NewTimeoutReader(resp.Body, readTimeout)
			for {
				if conn.ctx.Err() != nil {
					return conn.ctx.Err()
				}

				n, err := reader.Read(buf)
				if n > 0 {
					f.connMu.Lock()
					writeOffset := conn.Chunk.Downloaded
					f.connMu.Unlock()

					f.fileMu.Lock()
					if f.file != nil {
						_, writeErr := f.file.WriteAt(buf[:n], writeOffset)
						if writeErr != nil {
							f.fileMu.Unlock()
							return writeErr
						}
					}
					f.fileMu.Unlock()

					f.connMu.Lock()
					conn.Chunk.Downloaded += int64(n)
					conn.Downloaded += int64(n)
					f.connMu.Unlock()
				}
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
			}
		}()

		if err == nil {
			f.connMu.Lock()
			conn.Completed = true
			conn.State = connCompleted
			f.connMu.Unlock()
			return
		}

		if errors.Is(err, context.Canceled) {
			return
		}

		f.connMu.Lock()
		if re := extractRequestError(err); re != nil {
			conn.lastErr = re
		} else {
			conn.lastErr = err
		}
		f.connMu.Unlock()

		if shouldCountHTTPFailure(err) {
			// Immediate fail for server connection limit (403)
			if re := extractRequestError(err); re != nil && re.Code == 403 {
				f.connMu.Lock()
				conn.State = connFailed
				conn.failed = true
				f.connMu.Unlock()
				if f.slowStart != nil {
					f.slowStart.onConnectFailed()
				}
				return
			}
			f.connMu.Lock()
			conn.retryTimes++
			f.connMu.Unlock()
			countedRetries++
			if countedRetries >= 3 {
				f.connMu.Lock()
				conn.State = connFailed
				conn.failed = true
				f.connMu.Unlock()
				if f.slowStart != nil {
					f.slowStart.onConnectFailed()
				}
				return
			}
			// Retry again for counted failures below the cap
			f.connMu.Lock()
			conn.State = connFailed
			f.connMu.Unlock()
			retryDelay := time.Second * time.Duration(retries+1)
			if retryDelay > 5*time.Second {
				retryDelay = 5 * time.Second
			}
			retries++
			time.Sleep(retryDelay)
			continue
		}

		// Retry indefinitely for non-counted errors
		f.connMu.Lock()
		conn.State = connFailed
		f.connMu.Unlock()
		retryDelay := time.Second * time.Duration(retries+1)
		if retryDelay > 5*time.Second {
			retryDelay = 5 * time.Second
		}
		retries++
		time.Sleep(retryDelay)
	}
}

// helpOtherConnection implements work stealing: when a connection finishes its chunk,
// it looks for connections that need more than stealThresholdSeconds to finish and steals half of its work.
func (f *Fetcher) helpOtherConnection(helper *connection) bool {
	f.connMu.Lock()
	defer f.connMu.Unlock()

	// Find the connection with longest remaining time
	var slowestConn *connection
	var maxRemainSeconds int64
	for _, r := range f.connections {
		if r == helper || r.Completed || r.State == connFailed {
			continue
		}
		if r.Chunk.openEnded() {
			continue
		}

		remain := r.Chunk.remain()
		if remain < stealMinChunkSize {
			continue
		}

		// Calculate remaining time in seconds for this connection
		var remainSeconds int64
		if r.speed > 0 {
			remainSeconds = remain / r.speed
		} else {
			// Speed unknown, assume it needs help if chunk is large enough
			remainSeconds = stealThresholdSeconds + 1
		}

		// Only consider if it needs more than threshold seconds to finish
		if remainSeconds > stealThresholdSeconds && remainSeconds > maxRemainSeconds {
			slowestConn = r
			maxRemainSeconds = remainSeconds
		}
	}

	if slowestConn == nil {
		return false
	}

	// Re-calculate the chunk range: steal half of the remaining work
	chunkEnd := *slowestConn.Chunk.End
	helper.Chunk.Begin = chunkEnd - slowestConn.Chunk.remain()/2
	helper.Chunk.setEnd(chunkEnd)
	helper.Chunk.Downloaded = 0
	slowestConn.Chunk.setEnd(helper.Chunk.Begin - 1)
	return true
}

func (f *Fetcher) resetConnectionForRestart(conn *connection) {
	if f.meta.Res.Range {
		return
	}

	// Without range support a new request always starts from byte 0,
	// so pause/retry must restart instead of continuing from the old offset.
	if conn.Chunk == nil {
		conn.Chunk = newChunk(0, 0)
	} else {
		conn.Chunk.Begin = 0
		conn.Chunk.setEnd(0)
		conn.Chunk.Downloaded = 0
	}
	conn.Downloaded = 0
	conn.Completed = false
	conn.speed = 0
	conn.lastSpeedCheck = 0
	conn.lastSpeedDownload = 0
}

// hasSequentialPrefixLocked reports whether a single sequential connection has
// a useful contiguous prefix. The caller must hold connMu.
func (f *Fetcher) hasSequentialPrefixLocked(conn *connection) bool {
	if f.meta == nil || f.meta.Res == nil || f.meta.Res.Range || !f.rangeReprobeEligible || f.meta.Res.Size <= 0 {
		return false
	}
	if conn == nil || conn.ID != 0 || len(f.connections) != 1 || conn.Chunk == nil {
		return false
	}
	downloaded := conn.Chunk.Downloaded
	return downloaded > 0 && downloaded < f.meta.Res.Size
}

// canProbeSequentialResumeLocked reports whether the prefix can be resumed
// safely with If-Range. The caller must hold connMu.
func (f *Fetcher) canProbeSequentialResumeLocked(conn *connection) bool {
	return !f.sequentialSizeUnknown && f.ifRange != "" && f.hasSequentialPrefixLocked(conn)
}

func (f *Fetcher) hasSequentialRecoveryCandidate() bool {
	f.connMu.Lock()
	defer f.connMu.Unlock()
	if len(f.connections) != 1 {
		return false
	}
	return f.hasSequentialPrefixLocked(f.connections[0])
}

func (f *Fetcher) resumeConnections() {
	// Collect connections to resume while holding the lock
	var toResume []*connection

	f.connMu.Lock()
	for _, conn := range f.connections {
		// Only skip connections that have truly completed successfully
		if conn.Completed || conn.State == connCompleted {
			continue
		}
		// For failed connections, skip if:
		// 1. They have exhausted retries (retryTimes >= 3), OR
		// 2. They failed with a permanent error like 403
		if conn.State == connFailed && conn.failed {
			// Check if it's a permanent error (like 403)
			if re := extractRequestError(conn.lastErr); re != nil && re.Code == 403 {
				continue
			}
			// Check if retries exhausted
			if conn.retryTimes >= 3 {
				continue
			}
		}
		if !f.hasSequentialPrefixLocked(conn) {
			f.resetConnectionForRestart(conn)
		}
		// Reset the connection state for resume
		conn.ctx, conn.cancel = context.WithCancel(f.ctx)
		conn.State = connNotStarted
		conn.failed = false // Clear failed flag for resumed connection
		toResume = append(toResume, conn)
	}
	f.connMu.Unlock()

	// Start connections outside the lock
	for _, conn := range toResume {
		f.wg.Add(1)
		go f.runConnection(conn)
	}
}

func (f *Fetcher) waitForCompletion(ctx context.Context) {
	f.wg.Wait()
	// Only trigger completion for the Start cycle that launched this waiter.
	if ctx != nil && ctx.Err() == nil {
		f.onDownloadComplete()
	}
}

func (f *Fetcher) onDownloadComplete() {
	f.connMu.Lock()

	// First, check if download actually completed successfully
	// Calculate total downloaded from all connections
	totalDownloaded := int64(0)
	if f.resolveConn != nil {
		totalDownloaded += f.resolveConn.Downloaded
	}
	for _, conn := range f.connections {
		totalDownloaded += conn.Downloaded
	}

	// Check if all chunks are complete (no remaining bytes)
	allChunksComplete := true
	for _, conn := range f.connections {
		needsMoreData := false
		if f.meta.Res.Range {
			needsMoreData = conn.Chunk != nil && (conn.Chunk.openEnded() || conn.Chunk.remain() > 0)
		} else if f.meta.Res.Size > 0 {
			needsMoreData = conn.Downloaded < f.meta.Res.Size
		} else {
			needsMoreData = !conn.Completed && conn.State != connCompleted
		}

		if needsMoreData && !conn.Completed && conn.State != connCompleted {
			// This connection has remaining work and isn't done
			// Check if it failed with 403 (server limit) - these can be ignored if other connections completed the work
			if conn.State == connFailed && conn.failed {
				if re := extractRequestError(conn.lastErr); re != nil && re.Code == 403 {
					// 403 is server connection limit, check if other connections will complete this chunk
					continue
				}
			}
			allChunksComplete = false
			break
		}
	}

	// If total downloaded matches file size, consider it a success regardless of connection failures
	downloadComplete := f.meta.Res.Size > 0 && totalDownloaded >= f.meta.Res.Size

	// Integrity errors must win over byte-count completion. A malformed response
	// may be detected only after the expected bytes have already been written.
	var finalErr error
	for _, conn := range f.connections {
		if conn.State == connFailed && conn.failed && isTerminalRangeError(conn.lastErr) {
			finalErr = fmt.Errorf("connection %d failed: retries=%d, err=%w", conn.ID, conn.retryTimes, conn.lastErr)
			break
		}
	}

	// Check for other errors, but ignore 403 (server connection limit) errors if download completed.
	if finalErr == nil && !downloadComplete && !allChunksComplete {
		for _, conn := range f.connections {
			if conn.State == connFailed && conn.failed {
				// Skip 403 errors (server connection limit) - these are expected when exceeding server's limit
				if re := extractRequestError(conn.lastErr); re != nil && re.Code == 403 {
					continue
				}
				if re := extractRequestError(conn.lastErr); re != nil {
					finalErr = fmt.Errorf("connection %d failed: retries=%d, status=%d", conn.ID, conn.retryTimes, re.Code)
				} else if conn.lastErr != nil {
					finalErr = fmt.Errorf("connection %d failed: retries=%d, err=%v", conn.ID, conn.retryTimes, conn.lastErr)
				} else {
					finalErr = fmt.Errorf("connection %d failed: retries=%d", conn.ID, conn.retryTimes)
				}
				break
			}
		}
	}
	f.connMu.Unlock()

	// Close the file before signaling completion
	// This ensures the file handle is released before Wait() returns
	f.fileMu.Lock()
	if f.file != nil {
		f.file.Close()
		f.file = nil
	}
	f.fileMu.Unlock()

	if finalErr != nil {
		f.setState(stateError)
	} else {
		f.setState(stateDone)
	}

	select {
	case f.doneCh <- finalErr:
	default:
	}
}

func (f *Fetcher) checkCompletion() bool {
	// Check if all data has been downloaded
	f.connMu.Lock()
	defer f.connMu.Unlock()

	totalDownloaded := int64(0)
	if f.resolveConn != nil {
		totalDownloaded += f.resolveConn.Downloaded
	}
	for _, conn := range f.connections {
		totalDownloaded += conn.Downloaded
	}

	if f.meta.Res.Size > 0 && totalDownloaded >= f.meta.Res.Size {
		// Don't start a new goroutine - let the caller handle completion
		return true
	}

	// Check if all connections completed
	allCompleted := true
	if f.resolveConn != nil && !f.resolveConn.Completed && f.resolveConn.State != connCompleted {
		allCompleted = false
	}
	for _, conn := range f.connections {
		if !conn.Completed && conn.State != connCompleted && conn.State != connFailed {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		// Don't start a new goroutine - let the caller handle completion
		return true
	}

	return false
}

// Patch modifies the HTTP request information.
func (f *Fetcher) Patch(req *base.Request, opts *base.Options) error {
	// Patch request info
	if req != nil {
		if req.URL != "" {
			f.meta.Req.URL = req.URL
			// Clear redirect URL when URL is changed, so new requests use the new URL
			f.updateRedirectURL("")
		}
		if req.Extra != nil {
			if err := base.ParseReqExtra[fhttp.ReqExtra](req); err != nil {
				return err
			}
			patchExtra := req.Extra.(*fhttp.ReqExtra)
			// Merge Extra fields instead of replacing entirely
			if f.meta.Req.Extra == nil {
				f.meta.Req.Extra = &fhttp.ReqExtra{}
			}
			existingExtra := f.meta.Req.Extra.(*fhttp.ReqExtra)
			// Update Method only if non-empty
			if patchExtra.Method != "" {
				existingExtra.Method = patchExtra.Method
			}
			// Update Body only if non-empty
			if patchExtra.Body != "" {
				existingExtra.Body = patchExtra.Body
			}
			// Merge Headers: existing keys are overwritten, new keys are added
			if patchExtra.Header != nil {
				if existingExtra.Header == nil {
					existingExtra.Header = make(map[string]string)
				}
				for k, v := range patchExtra.Header {
					existingExtra.Header[k] = v
				}
			}
		}
		// Merge Labels: existing keys are overwritten, new keys are added
		if req.Labels != nil {
			if f.meta.Req.Labels == nil {
				f.meta.Req.Labels = make(map[string]string)
			}
			for k, v := range req.Labels {
				f.meta.Req.Labels[k] = v
			}
		}
		if req.Proxy != nil {
			f.meta.Req.Proxy = req.Proxy
		}
	}

	return nil
}

func (f *Fetcher) Pause() error {
	if f.cancel != nil {
		f.cancel()
	}
	if f.resolveCancel != nil {
		f.resolveCancel()
	}

	// Stop prefetch if running
	if f.prefetchStopCh != nil {
		select {
		case <-f.prefetchStopCh:
			// Already closed
		default:
			close(f.prefetchStopCh)
		}
	}

	// Wait for downloadLoop to exit first (it will call wg.Wait internally)
	if f.downloadLoopDone != nil {
		<-f.downloadLoopDone
	}

	// Wait for all connection goroutines to stop
	f.wg.Wait()

	// Wait for prefetch to finish
	for f.prefetchStopCh != nil && !f.prefetchDone.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	// If Pause wins before the first Range response is validated, materialize
	// the Resolve prefix before discarding its temporary file. The first
	// connection was allocated after the earlier prefetch snapshot, so advance
	// its range start to the final copied offset for a safe persisted resume.
	if f.meta != nil && f.meta.Res != nil && f.meta.Res.Range && f.prefetchFile != nil {
		f.resolveFallback.Store(false)
		prefetched := f.stopPrefetchAndCopyData()
		if prefetched > 0 {
			f.resolveDataPos.Store(prefetched)
			f.connMu.Lock()
			if len(f.connections) == 1 {
				conn := f.connections[0]
				if conn != nil && conn.ID == 0 && conn.Chunk != nil && !conn.Completed {
					conn.Chunk.Begin = prefetched
					if f.meta.Res.Size > 0 {
						conn.Chunk.setEnd(f.meta.Res.Size - 1)
					} else {
						conn.Chunk.End = nil
					}
					conn.Chunk.Downloaded = 0
					conn.Downloaded = prefetched
				}
			}
			f.connMu.Unlock()
		}
	} else {
		f.cleanupPrefetchFile()
	}

	// Clean up resolve response if still held
	f.resolveRespLock.Lock()
	if f.resolveResp != nil {
		f.resolveResp.Body.Close()
		f.resolveResp = nil
	}
	f.resolveRespLock.Unlock()

	f.fileMu.Lock()
	if f.file != nil {
		f.file.Close()
		f.file = nil
	}
	f.fileMu.Unlock()

	f.setState(statePaused)
	return nil
}

func (f *Fetcher) Close() error {
	err := f.Pause()
	if f.impersonationSession != nil {
		f.impersonationSession.Clear()
	}
	return err
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Stats() *fetcher.Stats {
	f.connMu.Lock()
	defer f.connMu.Unlock()

	terminal := f.getState() == stateDone
	prefetched := int64(0)
	connectionCount := len(f.connections)
	if connectionCount == 0 {
		prefetched = f.resolveDataPos.Load()
		if terminal && f.meta != nil && f.meta.Res != nil && f.meta.Res.Size > prefetched {
			// A small range response can finish entirely during resolve prefetch,
			// before a formal connection is created. The fetcher owns this fact, so
			// expose one aggregate connection here instead of making Downloader
			// infer protocol state from task-level progress.
			prefetched = f.meta.Res.Size
		}
		if prefetched > 0 {
			connectionCount = 1
		}
	}
	connectionTotal := f.averageConnectionTotal(connectionCount)
	statsConnections := make([]*fhttp.StatsConnection, 0)
	for _, connection := range f.connections {
		completed := connection.Completed || connection.State == connCompleted
		// A successful fetcher has no active downloads left. Work stealing and
		// completion-by-total-size can leave a helper connection's last internal
		// state non-terminal even though the whole HTTP transfer is finished.
		if terminal && !connection.failed {
			completed = true
		}
		statsConnections = append(statsConnections, &fhttp.StatsConnection{
			Downloaded: connection.Downloaded,
			Total:      connectionTotal,
			Completed:  completed,
			Failed:     connection.failed,
			RetryTimes: connection.retryTimes,
		})
	}
	if len(statsConnections) == 0 && prefetched > 0 {
		statsConnections = append(statsConnections, &fhttp.StatsConnection{
			Downloaded: prefetched,
			Total:      connectionTotal,
			Completed:  terminal,
		})
	}
	return &fetcher.Stats{
		Snapshot: &fhttp.Stats{
			Connections: statsConnections,
		},
	}
}

// averageConnectionTotal returns the nominal equal share used to compare how
// much work each currently visible connection has contributed. HTTP slow start
// can add connections over time, so all lanes are recalculated against the
// current connection count whenever stats are requested. A remainder is
// rounded up to avoid reporting 100% before a connection reaches its share.
func (f *Fetcher) averageConnectionTotal(connectionCount int) int64 {
	if f.meta == nil || f.meta.Res == nil || f.meta.Res.Size <= 0 || connectionCount <= 0 {
		return 0
	}
	total := f.meta.Res.Size / int64(connectionCount)
	if f.meta.Res.Size%int64(connectionCount) != 0 {
		total++
	}
	return total
}

func (f *Fetcher) Progress() fetcher.Progress {
	p := make(fetcher.Progress, 0)

	total := int64(0)
	if f.resolveConn != nil {
		total += f.resolveConn.Downloaded
	}

	f.connMu.Lock()
	for _, conn := range f.connections {
		total += conn.Downloaded
	}
	f.connMu.Unlock()

	p = append(p, total)
	return p
}

func (f *Fetcher) Wait() error {
	return <-f.doneCh
}
