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
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/xiaoqidun/setft"
)

const (
	connectTimeout = 15 * time.Second
	readTimeout    = 15 * time.Second
	helpMinSize    = 1 * 1024 * 1024 // Minimum chunk size for helper connections
)

// ============================================================================
// Error Types
// ============================================================================

type RequestError struct {
	Code int
	Msg  string
}

func NewRequestError(code int, msg string) *RequestError {
	return &RequestError{Code: code, Msg: msg}
}

func (re *RequestError) Error() string {
	return fmt.Sprintf("http request fail, code:%d, msg:%s", re.Code, re.Msg)
}

func isFailureExemptHTTPCode(code int) bool {
	if code >= 500 && code <= 599 {
		return true
	}

	switch code {
	case 429, 408, 440, 499:
		return true
	default:
		return false
	}
}

func shouldCountHTTPFailure(err error) bool {
	var re *RequestError
	if !errors.As(err, &re) {
		return false
	}

	return !isFailureExemptHTTPCode(re.Code)
}

func extractRequestError(err error) *RequestError {
	var re *RequestError
	if errors.As(err, &re) {
		return re
	}

	return nil
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

// startBatch marks the start of a new batch with count connections
func (s *slowStartController) startBatch(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batchPending = count
	s.batchReady = 0
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

// isMaxReached returns true if max connections have been launched
func (s *slowStartController) isMaxReached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.totalLaunched >= s.maxConnections
}

// ============================================================================
// Fetcher
// ============================================================================

type Fetcher struct {
	ctl    *controller.Controller
	config *config
	doneCh chan error

	meta *fetcher.FetcherMeta

	// State machine
	state atomic.Int32 // fetcherState

	// Connections
	connMu      sync.Mutex
	connections []*connection
	resolveConn *connection // The special resolve connection

	// Slow start controller
	slowStart *slowStartController

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

	// Target file
	file         *os.File
	fileMu       sync.Mutex
	redirectURL  string
	redirectLock sync.Mutex

	// Lifecycle control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

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
	httpReq, err := f.buildRequest(nil, req)
	if err != nil {
		f.setState(stateError)
		return err
	}

	client := f.buildClient()

	// Send normal HTTP request (no Range header)
	resp, err := client.Do(httpReq)
	if err != nil {
		f.setState(stateError)
		return err
	}

	// Parse response to get resource info
	res := &base.Resource{
		Range: false,
		Files: []*base.FileInfo{},
	}

	if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
		resp.Body.Close()
		f.setState(stateError)
		return NewRequestError(resp.StatusCode, resp.Status)
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

	default:
		return fmt.Errorf("cannot start in current state: %v", state)
	}
}

func (f *Fetcher) doStart() error {
	// Wait for resolve to complete
	<-f.resolvedCh

	state := f.getState()
	if state == stateDone || state == stateError {
		return nil
	}

	// Open or create target file first (needed for prefetch copy)
	name := f.meta.SingleFilepath()
	var err error
	_, err = os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			f.file, err = f.ctl.Touch(name, f.meta.Res.Size)
		} else {
			return err
		}
	} else {
		f.file, err = os.OpenFile(name, os.O_RDWR, os.ModeAppend)
	}
	if err != nil {
		return err
	}

	// For range-supported resources, stop prefetch and copy data
	// For non-range resources, the response will be used directly
	var prefetchedBytes int64
	if f.meta.Res.Range {
		// Stop async prefetch and copy data to target file
		prefetchedBytes = f.stopPrefetchAndCopyData()
		f.resolveDataPos.Store(prefetchedBytes)

		// Also close resolve response if still open
		f.resolveRespLock.Lock()
		if f.resolveResp != nil {
			f.resolveResp.Body.Close()
			f.resolveResp = nil
		}
		f.resolveRespLock.Unlock()
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

	// Start download
	f.setState(stateSlowStart)
	go f.downloadLoop()

	return nil
}

func (f *Fetcher) downloadLoop() {
	defer func() {
		f.fileMu.Lock()
		if f.file != nil {
			f.file.Close()
		}
		f.fileMu.Unlock()

		// Update file last modified time
		if f.config.UseServerCtime && f.meta.Res.Files[0].Ctime != nil {
			setft.SetFileTime(f.meta.SingleFilepath(), time.Now(), *f.meta.Res.Files[0].Ctime, *f.meta.Res.Files[0].Ctime)
		}
	}()

	// Check if this is a resume or fresh start
	isResume := len(f.connections) > 0

	if !isResume {
		// Fresh start: begin with resolve connection
		f.startResolveDownload()
	} else {
		// Resume: restart existing connections
		f.resumeConnections()
		f.waitForCompletion()
		return
	}

	// Slow start loop
	for {
		select {
		case <-f.ctx.Done():
			// Paused or cancelled
			return
		case <-f.slowStart.expansionCh:
			// Batch completed, try to expand
			if f.checkCompletion() {
				return
			}
			f.expandConnections()
		}
	}
}

func (f *Fetcher) startResolveDownload() {
	// If no range support or size unknown, just use single connection with resolve response
	if !f.meta.Res.Range || f.meta.Res.Size == 0 {
		// Create a single connection for the entire file
		conn := &connection{
			ID:    0,
			Role:  rolePrimary,
			State: connNotStarted,
			Chunk: newChunk(0, 0), // For non-range, end doesn't matter
		}
		conn.ctx, conn.cancel = context.WithCancel(f.ctx)
		f.connections = append(f.connections, conn)

		f.wg.Add(1)
		// Use the resolve response directly
		go f.runConnectionWithResolveResp(conn)

		// For non-range downloads, wait for completion in the main downloadLoop
		// by triggering immediate completion check
		go func() {
			f.wg.Wait()
			// Signal expansion channel to trigger completion check in downloadLoop
			select {
			case f.slowStart.expansionCh <- struct{}{}:
			default:
			}
		}()
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
		go f.waitForCompletion()
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
			f.setState(stateDone)
			f.doneCh <- nil
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
			if remain > maxRemain && remain > helpMinSize*2 {
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
		conn.ctx, conn.cancel = context.WithCancel(f.ctx)

		newConns = append(newConns, conn)
		f.connections = append(f.connections, conn)
	}

	f.connMu.Unlock()

	if len(newConns) == 0 {
		// No new connections could be created, stop expansion
		f.setState(stateSteady)
		go f.waitForCompletion()
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

	conn.State = connConnecting
	client := f.buildClient()
	buf := make([]byte, 8192)

	retries := 0
	conn.retryTimes = 0

	for {
		err := f.downloadChunkOnce(conn, client, buf)
		if err == nil {
			if !f.meta.Res.Range || !f.helpOtherConnection(conn) {
				conn.Completed = true
				conn.State = connCompleted
				return
			}

			// Reset counters after a successful help switch
			retries = 0
			conn.retryTimes = 0
			continue
		}

		if errors.Is(err, context.Canceled) {
			return
		}

		if re := extractRequestError(err); re != nil {
			conn.lastErr = re
		} else {
			conn.lastErr = err
		}

		if shouldCountHTTPFailure(err) {
			if re := extractRequestError(err); re != nil && re.Code == 403 {
				conn.State = connFailed
				conn.failed = true
				if f.slowStart != nil {
					f.slowStart.onConnectFailed()
				}
				return
			}
			conn.retryTimes++
			conn.failed = true
			if f.slowStart != nil {
				f.slowStart.onConnectFailed()
			}
			if conn.retryTimes >= 3 {
				conn.State = connFailed
				return
			}
		}

		conn.State = connFailed
		retryDelay := time.Second * time.Duration(retries+1)
		if retryDelay > 5*time.Second {
			retryDelay = 5 * time.Second
		}
		retries++
		time.Sleep(retryDelay)
	}
}

// downloadChunkOnce performs a single HTTP request for the current chunk without retrying.
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

	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
		return NewRequestError(resp.StatusCode, resp.Status)
	}

	conn.State = connDownloading
	conn.failed = false

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

			f.fileMu.Lock()
			if f.file != nil {
				_, writeErr := f.file.WriteAt(buf[:n], writeOffset)
				if writeErr != nil {
					f.fileMu.Unlock()
					return writeErr
				}
			}
			f.fileMu.Unlock()

			// Lock again to update Downloaded atomically with the read above
			f.connMu.Lock()
			conn.Chunk.Downloaded += int64(n)
			conn.Downloaded += int64(n)
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

	conn.State = connConnecting
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

	conn.State = connDownloading
	conn.failed = false

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
			f.fileMu.Lock()
			if f.file != nil {
				_, writeErr := f.file.WriteAt(buf[:n], conn.Chunk.Downloaded)
				if writeErr != nil {
					f.fileMu.Unlock()
					conn.State = connFailed
					conn.failed = true
					if f.slowStart != nil {
						f.slowStart.onConnectFailed()
					}
					return
				}
			}
			f.fileMu.Unlock()

			conn.Chunk.Downloaded += int64(n)
			conn.Downloaded += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				conn.Completed = true
				conn.State = connCompleted
				return
			}
			// Reading from resolve response failed: treat as transient (do not count as fail)
			conn.State = connFailed
			return
		}
	}
}

// runConnectionFallback is used when resolve response is not available
func (f *Fetcher) runConnectionFallback(conn *connection) {
	client := f.buildClient()
	buf := make([]byte, 8192)

	retries := 0
	countedRetries := 0

	for {
		if conn.ctx.Err() != nil {
			return
		}

		conn.State = connConnecting

		err := func() error {
			httpReq, err := f.buildRequest(conn.ctx, f.meta.Req)
			if err != nil {
				return err
			}

			resp, err := client.Do(httpReq)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
				return NewRequestError(resp.StatusCode, resp.Status)
			}

			conn.State = connDownloading
			conn.failed = false

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
					f.fileMu.Lock()
					if f.file != nil {
						_, writeErr := f.file.WriteAt(buf[:n], conn.Chunk.Downloaded)
						if writeErr != nil {
							f.fileMu.Unlock()
							return writeErr
						}
					}
					f.fileMu.Unlock()

					conn.Chunk.Downloaded += int64(n)
					conn.Downloaded += int64(n)
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
			conn.Completed = true
			conn.State = connCompleted
			return
		}

		if errors.Is(err, context.Canceled) {
			return
		}

		if re := extractRequestError(err); re != nil {
			conn.lastErr = re
		} else {
			conn.lastErr = err
		}

		if shouldCountHTTPFailure(err) {
			// Immediate fail for server connection limit (403)
			if re := extractRequestError(err); re != nil && re.Code == 403 {
				conn.State = connFailed
				conn.failed = true
				if f.slowStart != nil {
					f.slowStart.onConnectFailed()
				}
				return
			}
			conn.retryTimes++
			countedRetries++
			if countedRetries >= 3 {
				conn.State = connFailed
				conn.failed = true
				if f.slowStart != nil {
					f.slowStart.onConnectFailed()
				}
				return
			}
			// Retry again for counted failures below the cap
			conn.State = connFailed
			retryDelay := time.Second * time.Duration(retries+1)
			if retryDelay > 5*time.Second {
				retryDelay = 5 * time.Second
			}
			retries++
			time.Sleep(retryDelay)
			continue
		}

		// Retry indefinitely for non-counted errors
		conn.State = connFailed
		retryDelay := time.Second * time.Duration(retries+1)
		if retryDelay > 5*time.Second {
			retryDelay = 5 * time.Second
		}
		retries++
		time.Sleep(retryDelay)
	}
}

func (f *Fetcher) helpOtherConnection(helper *connection) bool {
	f.connMu.Lock()
	defer f.connMu.Unlock()

	// Find the slowest connection
	var maxRemainConnection *connection
	var maxRemain int64
	for _, r := range f.connections {
		if r == helper || r.Completed || r.State == connFailed {
			continue
		}

		remain := r.Chunk.remain()
		if remain > maxRemain && remain > helpMinSize {
			maxRemainConnection = r
			maxRemain = remain
		}
	}

	if maxRemainConnection == nil {
		return false
	}

	// Re-calculate the chunk range
	helper.Chunk.Begin = maxRemainConnection.Chunk.End - maxRemainConnection.Chunk.remain()/2
	helper.Chunk.End = maxRemainConnection.Chunk.End
	helper.Chunk.Downloaded = 0
	maxRemainConnection.Chunk.End = helper.Chunk.Begin - 1
	return true
}

func (f *Fetcher) resumeConnections() {
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
		// Reset the connection state for resume
		conn.ctx, conn.cancel = context.WithCancel(f.ctx)
		conn.State = connNotStarted
		conn.failed = false // Clear failed flag for resumed connection
		f.wg.Add(1)
		go f.runConnection(conn)
	}
}

func (f *Fetcher) waitForCompletion() {
	f.wg.Wait()
	// Only trigger completion if not cancelled/paused
	if f.ctx != nil && f.ctx.Err() == nil {
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
		if conn.Chunk != nil && conn.Chunk.remain() > 0 && !conn.Completed && conn.State != connCompleted {
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
	var finalErr error
	if !downloadComplete && !allChunksComplete {
		for _, conn := range f.connections {
			if conn.State == connFailed && conn.failed {
				// Skip 403 errors (server connection limit) - these are expected when exceeding server's limit
				if re := extractRequestError(conn.lastErr); re != nil && re.Code == 403 {
					continue
				}
				if re := extractRequestError(conn.lastErr); re != nil {
					finalErr = fmt.Errorf("connection %d failed: retries=%d, http code=%d, msg=%s", conn.ID, conn.retryTimes, re.Code, re.Msg)
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
		f.setState(stateSteady)
		go f.waitForCompletion()
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
		f.setState(stateSteady)
		go f.waitForCompletion()
		return true
	}

	return false
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

	// Wait for all goroutines to stop
	f.wg.Wait()

	// Wait for prefetch to finish
	for f.prefetchStopCh != nil && !f.prefetchDone.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	// Clean up prefetch file
	f.cleanupPrefetchFile()

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
	return f.Pause()
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
