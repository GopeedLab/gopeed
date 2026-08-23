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
	End        int64
	Downloaded int64
}

func (c *chunk) remain() int64 {
	return c.End - c.Begin + 1 - c.Downloaded
}

func newChunk(begin int64, end int64) *chunk {
	return &chunk{
		Begin: begin,
		End:   end,
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
	retryTimes int
	lastErr    error

	// Speed tracking for work stealing decisions
	speed             int64 // bytes per second
	lastSpeedCheck    int64 // timestamp in nanoseconds
	lastSpeedDownload int64 // bytes downloaded at last check

	ctx    context.Context
	cancel context.CancelFunc
	run    *downloadRun
}

type targetFile interface {
	WriteAt(p []byte, off int64) (n int, err error)
	Close() error
}

type targetWriteError struct {
	err error
}

// downloadRun owns all lifecycle state for one Start invocation. Keeping these
// values run-local prevents an immediate retry from replacing channels or
// cancellation functions that are still used by the previous run.
type downloadRun struct {
	ctx      context.Context
	cancel   context.CancelFunc
	loopDone chan struct{}

	terminalErrMu sync.Mutex
	terminalErr   error

	completionOnce sync.Once
	resultMu       sync.Mutex
	resultErr      error
	resultReady    bool
}

func newDownloadRun() *downloadRun {
	ctx, cancel := context.WithCancel(context.Background())
	return &downloadRun{
		ctx:      ctx,
		cancel:   cancel,
		loopDone: make(chan struct{}),
	}
}

func (r *downloadRun) fail(err error) {
	if err == nil {
		return
	}
	r.terminalErrMu.Lock()
	if r.terminalErr == nil {
		r.terminalErr = err
	}
	r.terminalErrMu.Unlock()
	r.cancel()
}

func (r *downloadRun) getTerminalErr() error {
	r.terminalErrMu.Lock()
	defer r.terminalErrMu.Unlock()
	return r.terminalErr
}

func (r *downloadRun) setResult(err error) {
	r.completionOnce.Do(func() {
		r.resultMu.Lock()
		r.resultErr = err
		r.resultReady = true
		r.resultMu.Unlock()
	})
}

func (r *downloadRun) result() (error, bool) {
	r.resultMu.Lock()
	defer r.resultMu.Unlock()
	return r.resultErr, r.resultReady
}

func (e *targetWriteError) Error() string {
	return fmt.Sprintf("write http data: %v", e.err)
}

func (e *targetWriteError) Unwrap() error {
	return e.err
}

func isTargetWriteError(err error) bool {
	var targetErr *targetWriteError
	return errors.As(err, &targetErr)
}

func waitForRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
	state  atomic.Int32 // fetcherState
	closed atomic.Bool

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
	prefetchDoneCh   chan struct{} // Closed after asyncPrefetch releases its resources
	prefetchStopOnce sync.Once

	// Target file
	file         targetFile
	fileMu       sync.Mutex
	redirectURL  string
	redirectLock sync.Mutex

	// Lifecycle control
	startMu sync.Mutex
	runMu   sync.Mutex
	run     *downloadRun
	wg      sync.WaitGroup

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

func (f *Fetcher) currentRun() *downloadRun {
	f.runMu.Lock()
	defer f.runMu.Unlock()
	return f.run
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
		t, _ := time.Parse(time.RFC1123, lastModified)
		lastModifiedTime = &t
	}

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
	if res.Range && res.Size > 0 {
		f.prefetchStopCh = make(chan struct{})
		f.prefetchDoneCh = make(chan struct{})
		f.prefetchStopOnce = sync.Once{}
		f.prefetchDone.Store(false)
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
		close(f.prefetchDoneCh)
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

	for {
		select {
		case <-f.prefetchStopCh:
			// Stop signal received (Start was called)
			return
		default:
		}

		n, err := resp.Body.Read(buf)
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

// stopPrefetch closes the response body to unblock any in-flight read, then
// waits until asyncPrefetch has stopped using the temporary file.
func (f *Fetcher) stopPrefetch() {
	if f.prefetchStopCh == nil {
		return
	}
	f.prefetchStopOnce.Do(func() {
		close(f.prefetchStopCh)
	})

	f.resolveRespLock.Lock()
	if f.resolveResp != nil {
		_ = f.resolveResp.Body.Close()
		f.resolveResp = nil
	}
	f.resolveRespLock.Unlock()

	if f.prefetchDoneCh != nil {
		<-f.prefetchDoneCh
	}
}

// stopPrefetchAndCopyData stops async prefetch and copies the available bytes
// to the target file.
func (f *Fetcher) stopPrefetchAndCopyData() (copied int64, err error) {
	f.stopPrefetch()

	prefetched := f.prefetchSize.Load()
	if prefetched == 0 {
		f.cleanupPrefetchFile()
		return 0, nil
	}
	defer f.cleanupPrefetchFile()

	// Copy prefetch data to target file
	if f.prefetchFile != nil && f.file != nil {
		// Seek to beginning of prefetch file
		if _, err = f.prefetchFile.Seek(0, io.SeekStart); err != nil {
			return 0, fmt.Errorf("seek http prefetch data: %w", err)
		}

		// Copy to target file at position 0
		buf := make([]byte, 32*1024)
		for copied < prefetched {
			readBuf := buf
			if remain := prefetched - copied; remain < int64(len(readBuf)) {
				readBuf = readBuf[:remain]
			}
			n, readErr := f.prefetchFile.Read(readBuf)
			if n > 0 {
				if err = f.writeTargetAt(buf[:n], copied); err != nil {
					return copied, err
				}
				copied += int64(n)
			}
			if readErr != nil {
				if readErr == io.EOF && copied == prefetched {
					break
				}
				if readErr == io.EOF {
					readErr = io.ErrUnexpectedEOF
				}
				return copied, fmt.Errorf("read http prefetch data: %w", readErr)
			}
		}
	}

	return copied, nil
}

func (f *Fetcher) writeTargetAt(p []byte, offset int64) error {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()

	if f.file == nil {
		return &targetWriteError{err: errors.New("target file is unavailable")}
	}
	written, err := f.file.WriteAt(p, offset)
	if err != nil {
		return &targetWriteError{err: err}
	}
	if written != len(p) {
		return &targetWriteError{err: io.ErrShortWrite}
	}
	return nil
}

func (f *Fetcher) closeTargetFile() error {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()

	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}

func (f *Fetcher) markConnectionWriteFailed(conn *connection, err error) {
	f.connMu.Lock()
	conn.State = connFailed
	conn.failed = true
	conn.lastErr = err
	f.connMu.Unlock()
	if f.slowStart != nil {
		f.slowStart.onConnectFailed()
	}
	if conn.run != nil {
		conn.run.fail(err)
	}
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
}

func (f *Fetcher) Start() error {
	if f.closed.Load() {
		return errors.New("http fetcher is closed")
	}
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
		// Already downloading. Pause transitions to statePaused before returning,
		// so starting another run here would race the active WaitGroup and file.
		return nil

	case stateError:
		// Retry after error: reset and restart
		return f.doStart()

	default:
		return fmt.Errorf("cannot start in current state: %v", state)
	}
}

func (f *Fetcher) doStart() error {
	f.startMu.Lock()
	defer f.startMu.Unlock()
	if f.closed.Load() {
		return errors.New("http fetcher is closed")
	}

	// Wait for resolve to complete
	<-f.resolvedCh

	state := f.getState()
	if state == stateDone || state == stateSlowStart || state == stateSteady {
		return nil
	}

	// If retrying after error, reset connection states for retry
	if state == stateError {
		// finishDownload sets stateError before downloadLoop publishes the result.
		// A caller may retry based on state without first calling Wait, so ensure
		// the old run has completely exited before replacing run-local state.
		if previousRun := f.currentRun(); previousRun != nil {
			<-previousRun.loopDone
		}
		// Drain any pending error from doneCh before retry
		select {
		case <-f.doneCh:
		default:
		}

		f.connMu.Lock()
		for _, conn := range f.connections {
			// Reset connections that can be retried
			if !conn.Completed && conn.State != connCompleted {
				f.resetConnectionForRestart(conn)
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
		// Stop async prefetch and copy data to target file
		var copyErr error
		prefetchedBytes, copyErr = f.stopPrefetchAndCopyData()

		// Also close resolve response if still open
		f.resolveRespLock.Lock()
		if f.resolveResp != nil {
			f.resolveResp.Body.Close()
			f.resolveResp = nil
		}
		f.resolveRespLock.Unlock()
		if copyErr != nil {
			_ = f.closeTargetFile()
			return copyErr
		}
		f.resolveDataPos.Store(prefetchedBytes)
	}

	// Avoid request extra modified by extension
	if err = base.ParseReqExtra[fhttp.ReqExtra](f.meta.Req); err != nil {
		_ = f.closeTargetFile()
		return err
	}

	// Initialize slow start controller
	maxConns := f.meta.Opts.Extra.(*fhttp.OptsExtra).Connections
	f.slowStart = newSlowStartController(maxConns)

	// Create run-local lifecycle state. Wait is signalled only after this run's
	// loop and all of its connection goroutines have exited.
	run := newDownloadRun()
	f.runMu.Lock()
	f.run = run
	f.runMu.Unlock()

	// Start download
	f.setState(stateSlowStart)
	go f.downloadLoop(run)

	return nil
}

func (f *Fetcher) downloadLoop(run *downloadRun) {
	defer func() {
		// Update file last modified time before closing
		if f.config.UseServerCtime && f.meta.Res.Files[0].Ctime != nil {
			setft.SetFileTime(f.meta.SingleFilepath(), time.Now(), *f.meta.Res.Files[0].Ctime, *f.meta.Res.Files[0].Ctime)
		}

		if err, ok := run.result(); ok {
			f.doneCh <- err
		}
		// Close only after the result has been published. A retry waiting on
		// loopDone can now safely drain the previous result before replacing run.
		close(run.loopDone)
	}()

	// Check if this is a resume or fresh start
	f.connMu.Lock()
	isResume := len(f.connections) > 0
	f.connMu.Unlock()

	if !isResume {
		// Fresh start: begin with resolve connection
		f.startResolveDownload(run)
		if _, completed := run.result(); completed {
			return
		}
		if !f.meta.Res.Range || f.meta.Res.Size == 0 {
			return
		}
	} else {
		// Resume: restart existing connections
		f.resumeConnections(run)
		f.waitForCompletion(run)
		return
	}

	// Slow start loop
	for {
		select {
		case <-run.ctx.Done():
			// A target write error is terminal for the whole run. Pause also
			// cancels the context, but intentionally has no completion result.
			if run.getTerminalErr() != nil {
				f.waitForCompletion(run)
			}
			return
		case <-f.slowStart.expansionCh:
			// Batch completed, try to expand
			if f.checkCompletion() {
				// All work is done, wait for connections to finish
				f.waitForCompletion(run)
				return
			}
			f.expandConnections(run)

			// Check if we've reached steady state (max connections)
			if f.getState() == stateSteady {
				// Wait for all connections to complete
				f.waitForCompletion(run)
				return
			}
		}
	}
}

func (f *Fetcher) startResolveDownload(run *downloadRun) {
	// If no range support or size unknown, just use single connection with resolve response
	if !f.meta.Res.Range || f.meta.Res.Size == 0 {
		// Create a single connection for the entire file
		conn := &connection{
			ID:    0,
			Role:  rolePrimary,
			State: connNotStarted,
			Chunk: newChunk(0, 0), // For non-range, end doesn't matter
		}
		conn.ctx, conn.cancel = context.WithCancel(run.ctx)
		conn.run = run
		f.connMu.Lock()
		f.connections = append(f.connections, conn)
		f.connMu.Unlock()

		f.wg.Add(1)
		// Use the resolve response directly
		go f.runConnectionWithResolveResp(conn)

		// For non-range downloads, wait for completion directly in this goroutine
		// Don't create another goroutine to avoid WaitGroup reuse issues
		f.waitForCompletion(run)
		return
	}

	// Range supported: use slow start to launch connections
	// Start first batch of connections
	f.expandConnections(run)
}

func (f *Fetcher) expandConnections(run *downloadRun) {
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

		// If prefetched all data, mark as done
		if prefetched >= totalSize {
			f.connMu.Unlock()
			f.finishDownload(run, nil)
			return
		}

		// First connection starts from prefetched position
		conn := &connection{
			ID:    0,
			Role:  rolePrimary,
			State: connNotStarted,
			Chunk: newChunk(prefetched, totalSize-1),
		}
		// Mark prefetched bytes as already downloaded
		conn.Chunk.Downloaded = 0    // Start fresh from prefetched position
		conn.Downloaded = prefetched // Track total downloaded including prefetch

		conn.ctx, conn.cancel = context.WithCancel(run.ctx)
		conn.run = run
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
		splitPoint := maxRemainConn.Chunk.End - maxRemainConn.Chunk.remain()/2
		newChunk := newChunk(splitPoint+1, maxRemainConn.Chunk.End)
		maxRemainConn.Chunk.End = splitPoint

		connID := len(f.connections)
		conn := &connection{
			ID:    connID,
			Role:  roleWorker,
			State: connNotStarted,
			Chunk: newChunk,
		}
		conn.ctx, conn.cancel = context.WithCancel(run.ctx)
		conn.run = run

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
	defer f.wg.Done()

	f.connMu.Lock()
	conn.State = connConnecting
	conn.retryTimes = 0
	f.connMu.Unlock()

	// Use fast-fail client for quick retry during download phase
	client := f.buildFastFailClient()
	buf := make([]byte, 8192)

	retries := 0

	for {
		// Rebuild client with updated fast-fail timeout on retries
		if retries > 0 {
			client = f.buildFastFailClient()
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
		if isTargetWriteError(err) {
			f.markConnectionWriteFailed(conn, err)
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
		if !waitForRetry(conn.ctx, retryDelay) {
			return
		}
	}
}

// downloadChunkOnce performs a single HTTP request for the current chunk without retrying.
// If the redirect URL fails with an expiration-related error (401, 403, 410),
// it will automatically retry with the original URL and update the redirect URL on success.
func (f *Fetcher) downloadChunkOnce(conn *connection, client *http.Client, buf []byte) error {
	if conn.ctx.Err() != nil {
		return conn.ctx.Err()
	}

	// Read chunk boundaries under lock to get a consistent snapshot
	// This protects against concurrent modification by helpOtherConnection
	f.connMu.Lock()
	if f.meta.Res.Range && conn.Chunk.remain() <= 0 {
		f.connMu.Unlock()
		return nil
	}
	rangeStart := conn.Chunk.Begin + conn.Chunk.Downloaded
	rangeEnd := conn.Chunk.End
	f.connMu.Unlock()

	httpReq, err := f.buildRequest(conn.ctx, f.meta.Req)
	if err != nil {
		return err
	}

	if f.meta.Res.Range {
		httpReq.Header.Set(base.HttpHeaderRange,
			fmt.Sprintf(base.HttpHeaderRangeFormat, rangeStart, rangeEnd))
	}

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
			fallbackResp, fallbackErr := f.tryFallbackToOriginalURL(conn.ctx, client, rangeStart, rangeEnd)
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
	for {
		if conn.ctx.Err() != nil {
			return conn.ctx.Err()
		}

		n, err := reader.Read(buf)
		if n > 0 {
			finished := false
			var writeOffset int64

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
				if remain < int64(n) {
					n = int(remain)
					finished = true
				}
			}
			writeOffset = conn.Chunk.Begin + conn.Chunk.Downloaded
			f.connMu.Unlock()

			if writeErr := f.writeTargetAt(buf[:n], writeOffset); writeErr != nil {
				return writeErr
			}

			// Lock again to update Downloaded atomically with the read above
			f.connMu.Lock()
			conn.Chunk.Downloaded += int64(n)
			conn.Downloaded += int64(n)
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

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
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
			if writeErr := f.writeTargetAt(buf[:n], writeOffset); writeErr != nil {
				f.markConnectionWriteFailed(conn, writeErr)
				return
			}

			f.connMu.Lock()
			conn.Chunk.Downloaded += int64(n)
			conn.Downloaded += int64(n)
			f.connMu.Unlock()
		}
		if err != nil {
			if err == io.EOF {
				f.connMu.Lock()
				conn.Completed = true
				conn.State = connCompleted
				f.connMu.Unlock()
				return
			}
			// Reading from resolve response failed: treat as transient (do not count as fail)
			f.connMu.Lock()
			conn.State = connFailed
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
					if writeErr := f.writeTargetAt(buf[:n], writeOffset); writeErr != nil {
						return writeErr
					}

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
		if isTargetWriteError(err) {
			f.markConnectionWriteFailed(conn, err)
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
			if !waitForRetry(conn.ctx, retryDelay) {
				return
			}
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
		if !waitForRetry(conn.ctx, retryDelay) {
			return
		}
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
	helper.Chunk.Begin = slowestConn.Chunk.End - slowestConn.Chunk.remain()/2
	helper.Chunk.End = slowestConn.Chunk.End
	helper.Chunk.Downloaded = 0
	slowestConn.Chunk.End = helper.Chunk.Begin - 1
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
		conn.Chunk.End = 0
		conn.Chunk.Downloaded = 0
	}
	conn.Downloaded = 0
	conn.Completed = false
	conn.speed = 0
	conn.lastSpeedCheck = 0
	conn.lastSpeedDownload = 0
}

func (f *Fetcher) resumeConnections(run *downloadRun) {
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
		f.resetConnectionForRestart(conn)
		// Reset the connection state for resume
		conn.ctx, conn.cancel = context.WithCancel(run.ctx)
		conn.run = run
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

func (f *Fetcher) waitForCompletion(run *downloadRun) {
	f.wg.Wait()
	// Target write errors cancel sibling connections but must still complete the
	// run with the original local-storage error. A plain cancellation is Pause.
	if run.getTerminalErr() != nil || run.ctx.Err() == nil {
		f.onDownloadComplete(run)
	}
}

func (f *Fetcher) onDownloadComplete(run *downloadRun) {
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
			needsMoreData = conn.Chunk != nil && conn.Chunk.remain() > 0
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

	// Check for any errors, but ignore 403 (server connection limit) errors if download completed
	finalErr := run.getTerminalErr()
	if finalErr == nil && !downloadComplete && !allChunksComplete {
		for _, conn := range f.connections {
			if conn.State == connFailed && conn.failed {
				// Skip 403 errors (server connection limit) - these are expected when exceeding server's limit
				if re := extractRequestError(conn.lastErr); re != nil && re.Code == 403 {
					continue
				}
				if re := extractRequestError(conn.lastErr); re != nil {
					finalErr = fmt.Errorf("connection %d failed: retries=%d, status=%d", conn.ID, conn.retryTimes, re.Code)
				} else if isTargetWriteError(conn.lastErr) {
					finalErr = conn.lastErr
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

	f.finishDownload(run, finalErr)
}

// finishDownload closes the target and records exactly one terminal result.
// downloadLoop publishes that result only after its defer closes loopDone.
func (f *Fetcher) finishDownload(run *downloadRun, finalErr error) {
	if closeErr := f.closeTargetFile(); closeErr != nil && finalErr == nil {
		finalErr = fmt.Errorf("close http target file: %w", closeErr)
	}

	if finalErr != nil {
		f.setState(stateError)
	} else {
		f.setState(stateDone)
	}

	run.setResult(finalErr)
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
	f.startMu.Lock()
	defer f.startMu.Unlock()
	return f.pauseLocked()
}

func (f *Fetcher) pauseLocked() error {
	run := f.currentRun()
	if run != nil {
		run.cancel()
	}
	if f.resolveCancel != nil {
		f.resolveCancel()
	}

	// Closing the resolve body unblocks prefetch before its temporary file is
	// sought, copied, closed, or removed.
	f.stopPrefetch()

	// Wait for this exact run rather than a channel that a retry can replace.
	if run != nil {
		<-run.loopDone
	}

	// Wait for all connection goroutines to stop
	f.wg.Wait()

	// Clean up prefetch file
	f.cleanupPrefetchFile()

	// Clean up resolve response if still held
	f.resolveRespLock.Lock()
	if f.resolveResp != nil {
		f.resolveResp.Body.Close()
		f.resolveResp = nil
	}
	f.resolveRespLock.Unlock()

	_ = f.closeTargetFile()

	f.setState(statePaused)
	return nil
}

func (f *Fetcher) Close() error {
	f.startMu.Lock()
	defer f.startMu.Unlock()
	f.closed.Store(true)
	err := f.pauseLocked()
	if f.impersonationSession != nil {
		f.impersonationSession.Clear()
	}
	return err
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Stats() any {
	f.connMu.Lock()
	defer f.connMu.Unlock()

	statsConnections := make([]*fhttp.StatsConnection, 0)
	for _, connection := range f.connections {
		statsConnections = append(statsConnections, &fhttp.StatsConnection{
			Downloaded: connection.Downloaded,
			Completed:  connection.Completed,
			Failed:     connection.failed,
			RetryTimes: connection.retryTimes,
		})
	}
	return &fhttp.Stats{
		Connections: statsConnections,
	}
}

func (f *Fetcher) Progress() fetcher.Progress {
	p := make(fetcher.Progress, 0)

	total := int64(0)
	f.connMu.Lock()
	if f.resolveConn != nil {
		total += f.resolveConn.Downloaded
	}
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
