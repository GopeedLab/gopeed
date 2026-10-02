// Package ftp downloads files and directory trees over FTP, FTPS (implicit
// TLS) and FTPES (explicit TLS, AUTH TLS).
//
// A file is split into segments of at least 1 MiB. Each segment is fetched on
// its own login with REST and RETR, and its data connection is closed once
// the segment's last byte has arrived. A server that refuses a login lowers
// the task's connection ceiling to the logins that worked, and a server that
// refuses REST gets one sequential stream per file.
package ftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/rs/zerolog"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	pftp "github.com/GopeedLab/gopeed/pkg/protocol/ftp"
)

const (
	// Many servers cap logins per user, so FTP starts lower than HTTP.
	defaultConnections = 4
	maxConnections     = 256
	minSegmentSize     = 1 << 20
	maxRetries         = 5
	copyBufferSize     = 256 << 10
)

// retryDelay is the first wait before a retry. Later retries wait longer.
var retryDelay = 500 * time.Millisecond

var (
	errSizeChanged     = errors.New("ftp: the remote file changed size since the task was resolved")
	errShortTransfer   = errors.New("ftp: the transfer ended before the segment was complete")
	errRestUnsupported = errors.New("ftp: the server does not support REST")
	errDiskWrite       = errors.New("ftp: cannot write the downloaded data")
)

type config struct {
	Connections int `json:"connections"`
}

// segment is a byte range of one file. End is inclusive, so an empty file
// has End == Begin-1 and nothing to fetch.
type segment struct {
	File       int `json:",omitempty"` // index in Res.Files
	Begin      int64
	End        int64
	Downloaded int64
}

func (s *segment) remain() int64 {
	return s.End - s.Begin + 1 - s.Downloaded
}

// lane is one worker that logged in, shown as one connection in the stats.
type lane struct {
	downloaded int64
	retryTimes int
	failed     bool
	completed  bool
}

// loginError marks a failure to connect or log in, as opposed to a failure
// during a transfer.
type loginError struct{ err error }

func (e *loginError) Error() string { return e.err.Error() }
func (e *loginError) Unwrap() error { return e.err }

type Fetcher struct {
	manager *FetcherManager
	ctl     *controller.Controller
	config  *config
	meta    *fetcher.FetcherMeta
	doneCh  chan error

	warnOnce sync.Once
	// loginOK is set once these credentials have logged in, after which a
	// 530 on another login means the server is full.
	loginOK atomic.Bool

	// mu guards the download state below.
	mu       sync.Mutex
	segments []*segment
	maxConns int
	noRest   bool
	lanes    []*lane
	done     bool

	// runMu serializes Start and Pause.
	runMu   sync.Mutex
	cancel  context.CancelFunc
	runDone chan struct{}
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	f.doneCh = make(chan error, 1)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	if ctl != nil && ctl.GetConfig != nil {
		ctl.GetConfig(&f.config)
	}
	if f.config == nil {
		f.config = &config{Connections: defaultConnections}
	}
}

func (f *Fetcher) logger() *zerolog.Logger {
	if f.ctl == nil {
		return nil
	}
	return f.ctl.Logger
}

func (f *Fetcher) warn(format string, args ...any) {
	if l := f.logger(); l != nil {
		l.Warn().Msgf(format, args...)
	}
}

func (f *Fetcher) initOptions() error {
	opts := f.meta.Opts
	if err := base.ParseOptExtra[pftp.OptsExtra](opts); err != nil {
		return err
	}
	if opts.Extra == nil {
		opts.Extra = &pftp.OptsExtra{}
	}
	return nil
}

func (f *Fetcher) extra() *pftp.OptsExtra {
	if e, ok := f.meta.Opts.Extra.(*pftp.OptsExtra); ok && e != nil {
		return e
	}
	return &pftp.OptsExtra{}
}

// connections is the number of parallel logins to use. The caller holds mu.
func (f *Fetcher) connections() int {
	if f.noRest {
		return 1
	}
	n := f.extra().Connections
	if n <= 0 && f.config != nil {
		n = f.config.Connections
	}
	if n <= 0 {
		n = defaultConnections
	}
	n = min(n, maxConnections)
	if f.maxConns > 0 {
		n = min(n, f.maxConns)
	}
	return n
}

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) error {
	if opts == nil {
		opts = &base.Options{}
	}
	f.meta.Req = req
	f.meta.Opts = opts
	if err := f.initOptions(); err != nil {
		return err
	}
	t, err := parseTarget(req.URL, f.extra().TLS)
	if err != nil {
		return err
	}
	s, err := f.dial(context.Background(), t)
	if err != nil {
		return err
	}
	// quit, not abort: a server that caps logins must see this session end
	// before the download's own logins arrive.
	defer s.quit()
	f.loginOK.Store(true)
	res, err := f.resolveTarget(s, t)
	if err != nil {
		return err
	}
	f.meta.Res = res
	f.mu.Lock()
	f.segments = nil
	f.done = false
	f.mu.Unlock()
	return nil
}

// selected returns the indexes of the files to download, in SelectFiles
// order. The caller holds mu or owns the fetcher.
func (f *Fetcher) selected() []int {
	if f.meta == nil || f.meta.Res == nil {
		return nil
	}
	if f.meta.Opts != nil && len(f.meta.Opts.SelectFiles) > 0 {
		return f.meta.Opts.SelectFiles
	}
	all := make([]int, len(f.meta.Res.Files))
	for i := range all {
		all[i] = i
	}
	return all
}

// localPath is where a file lands: under the task path, with the file's
// directories below the URL kept.
func (f *Fetcher) localPath(index int) string {
	if f.meta.Res.Name == "" {
		return f.meta.SingleFilepath()
	}
	file := f.meta.Res.Files[index]
	return path.Join(f.meta.FolderPath(), file.Path, file.Name)
}

// remotePath is the server path of a file. A single file uses the URL's own
// path, which a local rename never touches.
func (f *Fetcher) remotePath(t *target, index int) string {
	if f.meta.Res.Name == "" {
		return t.path
	}
	file := f.meta.Res.Files[index]
	return path.Join(t.path, file.Path, file.Name)
}

// plan splits every selected file into segments of at least minSegmentSize,
// one per connection at most. The caller holds mu.
func (f *Fetcher) plan() {
	conns := f.connections()
	f.segments = nil
	for _, index := range f.selected() {
		size := f.meta.Res.Files[index].Size
		n := int64(1)
		if !f.noRest {
			n = max(1, min(int64(conns), size/minSegmentSize))
		}
		for i := int64(0); i < n; i++ {
			f.segments = append(f.segments, &segment{
				File:  index,
				Begin: i * size / n,
				End:   (i+1)*size/n - 1,
			})
		}
	}
}

// planSequential gives every unfinished file one segment from byte 0,
// because without REST a transfer cannot start in the middle. The caller
// holds mu.
func (f *Fetcher) planSequential() {
	complete := map[int]bool{}
	for _, s := range f.segments {
		if _, seen := complete[s.File]; !seen {
			complete[s.File] = true
		}
		if s.remain() > 0 {
			complete[s.File] = false
		}
	}
	var out []*segment
	for _, index := range f.selected() {
		size := f.meta.Res.Files[index].Size
		if complete[index] {
			out = append(out, &segment{File: index, Begin: 0, End: size - 1, Downloaded: size})
			continue
		}
		out = append(out, &segment{File: index, Begin: 0, End: size - 1})
	}
	f.segments = out
	// Bytes counted by the old segments are fetched again.
	f.lanes = nil
}

// openFiles opens every selected file for writing. A fresh plan creates and
// sizes them; a resumed one keeps their content, and a file that has gone
// missing starts over.
func (f *Fetcher) openFiles(fresh bool) (map[int]*os.File, error) {
	files := map[int]*os.File{}
	closeAll := func() {
		for _, file := range files {
			_ = file.Close()
		}
	}
	for _, index := range f.selected() {
		name := f.localPath(index)
		var (
			file *os.File
			err  error
		)
		_, statErr := os.Stat(name)
		switch {
		case fresh || os.IsNotExist(statErr):
			file, err = f.ctl.Touch(name, f.meta.Res.Files[index].Size)
			if err == nil && !fresh {
				f.mu.Lock()
				for _, s := range f.segments {
					if s.File == index {
						s.Downloaded = 0
					}
				}
				f.mu.Unlock()
			}
		case statErr != nil:
			err = statErr
		default:
			file, err = os.OpenFile(name, os.O_RDWR, 0)
		}
		if err != nil {
			closeAll()
			return nil, err
		}
		files[index] = file
	}
	return files, nil
}

func (f *Fetcher) Start() error {
	f.runMu.Lock()
	defer f.runMu.Unlock()
	if f.runDone != nil {
		select {
		case <-f.runDone:
		default:
			return nil // already running
		}
	}
	if err := f.initOptions(); err != nil {
		return err
	}
	t, err := parseTarget(f.meta.Req.URL, f.extra().TLS)
	if err != nil {
		return err
	}

	// A paused or failed run may have left a result nobody read.
	select {
	case <-f.doneCh:
	default:
	}

	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		f.doneCh <- nil
		return nil
	}
	fresh := len(f.segments) == 0
	if fresh {
		f.plan()
	} else if f.noRest {
		f.planSequential()
	}
	f.mu.Unlock()

	files, err := f.openFiles(fresh)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.runDone = make(chan struct{})
	go f.run(ctx, t, files, f.runDone)
	return nil
}

func (f *Fetcher) run(ctx context.Context, t *target, files map[int]*os.File, done chan struct{}) {
	defer close(done)
	err := f.download(ctx, t, files)
	for _, file := range files {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	f.mu.Lock()
	if err == nil {
		f.done = true
		for _, l := range f.lanes {
			l.completed = true
		}
	}
	f.mu.Unlock()
	if err != nil && ctx.Err() != nil {
		return // paused
	}
	select {
	case f.doneCh <- err:
	default:
	}
}

func (f *Fetcher) download(ctx context.Context, t *target, files map[int]*os.File) error {
	for {
		err := f.runWorkers(ctx, t, files)
		if !errors.Is(err, errRestUnsupported) {
			return err
		}
		f.mu.Lock()
		f.noRest = true
		f.planSequential()
		f.mu.Unlock()
		f.warn("ftp: the server refused REST, so each file is downloaded in one stream")
	}
}

// pool hands out the segments of one run to its workers. Every decision to
// leave the run is taken under mu together with the queue it depends on, so
// a segment put back can never be left without a worker.
type pool struct {
	mu      sync.Mutex
	queue   []*segment
	running int // workers still in the run
}

// take returns the next segment. A worker that finds the queue empty leaves
// the run.
func (p *pool) take() *segment {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		p.running--
		return nil
	}
	s := p.queue[0]
	p.queue = p.queue[1:]
	return s
}

func (p *pool) putBack(s *segment) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append([]*segment{s}, p.queue...)
}

// refused puts back the segment of a worker whose login the server turned
// away. The worker leaves when another worker is still in the run to take
// the segment, and left is the number of workers that remain; the last
// worker stays and retries.
func (p *pool) refused(s *segment) (leave bool, left int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append([]*segment{s}, p.queue...)
	if p.running > 1 {
		p.running--
		return true, p.running
	}
	return false, p.running
}

// leave takes a worker out of the run when it stops for any other reason.
func (p *pool) leave() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running--
}

func (f *Fetcher) runWorkers(ctx context.Context, t *target, files map[int]*os.File) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	f.mu.Lock()
	p := &pool{}
	for _, s := range f.segments {
		if s.remain() > 0 {
			p.queue = append(p.queue, s)
		}
	}
	workers := min(f.connections(), len(p.queue))
	f.mu.Unlock()
	if workers == 0 {
		return nil
	}
	p.running = workers

	var (
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr error
	)
	for id := 0; id < workers; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := f.worker(runCtx, t, files, p, id); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				cancel()
			}
		}(id)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return err
	}
	if firstErr != nil {
		return firstErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.segments {
		if s.remain() > 0 {
			return errors.New("ftp: the download stopped with data missing")
		}
	}
	return nil
}

// worker fetches segments from the pool until it is empty. A worker whose
// login the server refuses (421, or 530 once these credentials have worked)
// hands its segment back and leaves while other workers remain, lowering the
// task's ceiling to the workers left. This holds whether or not the worker
// had logged in before: a slot can also go to another client mid-download.
func (f *Fetcher) worker(ctx context.Context, t *target, files map[int]*os.File, p *pool, id int) error {
	var ln *lane
	f.mu.Lock()
	if id < len(f.lanes) {
		ln = f.lanes[id]
	}
	f.mu.Unlock()
	onLogin := func() {
		f.mu.Lock()
		if ln == nil {
			ln = &lane{}
			f.lanes = append(f.lanes, ln)
		}
		f.mu.Unlock()
	}

	failures := 0
	pendingRetries := 0
	for {
		seg := p.take()
		if seg == nil {
			return nil
		}
		err := f.fetchSegment(ctx, t, files, seg, &ln, onLogin)
		if err == nil {
			failures = 0
			continue
		}
		if ctx.Err() != nil {
			p.putBack(seg)
			p.leave()
			return nil
		}

		var le *loginError
		if errors.As(err, &le) && isLoginRefusal(le.err, f.loginOK.Load()) {
			if leave, left := p.refused(seg); leave {
				f.lowerCeiling(left)
				return nil
			}
			// The last worker: wait for a slot.
		} else {
			p.putBack(seg)
			if errors.Is(err, errRestUnsupported) {
				p.leave()
				return err
			}
			if isPermanent(err) || errors.Is(err, errDiskWrite) {
				f.markFailed(ln)
				p.leave()
				return err
			}
		}

		failures++
		f.mu.Lock()
		if ln != nil {
			ln.retryTimes += pendingRetries + 1
			pendingRetries = 0
		} else {
			pendingRetries++
		}
		f.mu.Unlock()
		if failures > maxRetries {
			f.markFailed(ln)
			p.leave()
			return fmt.Errorf("ftp: giving up after %d retries: %w", maxRetries, err)
		}
		select {
		case <-ctx.Done():
			p.leave()
			return nil
		case <-time.After(retryDelay * time.Duration(failures)):
		}
	}
}

func (f *Fetcher) lowerCeiling(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.maxConns == 0 || n < f.maxConns {
		f.maxConns = n
	}
}

func (f *Fetcher) markFailed(ln *lane) {
	if ln == nil {
		return
	}
	f.mu.Lock()
	ln.failed = true
	f.mu.Unlock()
}

// isRestRefused tells a REST the server does not implement. RETR itself
// fails with 550 and the like, never with these codes.
func isRestRefused(err error) bool {
	switch replyCode(err) {
	case ftp.StatusBadCommand, ftp.StatusBadArguments, ftp.StatusNotImplemented, ftp.StatusNotImplementedParameter:
		return true
	}
	return false
}

// fetchSegment logs in, retrieves the segment from its first missing byte
// and writes it in place. It stops reading at the segment's end and closes
// the data connection; the server then reports the transfer aborted, which
// is expected.
func (f *Fetcher) fetchSegment(ctx context.Context, t *target, files map[int]*os.File, seg *segment, ln **lane, onLogin func()) error {
	s, err := f.dial(ctx, t)
	if err != nil {
		return &loginError{err}
	}
	defer s.quit()
	f.loginOK.Store(true)
	onLogin()

	remote := f.remotePath(t, seg.File)
	f.mu.Lock()
	noRest := f.noRest
	wantSize := f.meta.Res.Files[seg.File].Size
	if noRest {
		seg.Downloaded = 0
	}
	pos := seg.Begin + seg.Downloaded
	end := seg.End
	f.mu.Unlock()

	// A file that changed on the server would mix old and new bytes.
	s.deadline()
	if size, err := s.FileSize(remote); err == nil && size != wantSize {
		return errSizeChanged
	} else if err != nil && replyCode(err) == 0 {
		return err
	}

	s.deadline()
	var resp *ftp.Response
	if noRest {
		resp, err = s.Retr(remote)
	} else {
		resp, err = s.RetrFrom(remote, uint64(pos))
	}
	if err != nil {
		if !noRest && pos > 0 && isRestRefused(err) {
			return errRestUnsupported
		}
		return err
	}
	defer resp.Close()
	s.clearDeadline()

	file := files[seg.File]
	buf := make([]byte, copyBufferSize)
	for pos <= end {
		n := int(min(int64(len(buf)), end-pos+1))
		_ = resp.SetDeadline(time.Now().Add(readTimeout))
		nr, rerr := resp.Read(buf[:n])
		if nr > 0 {
			if _, werr := file.WriteAt(buf[:nr], pos); werr != nil {
				return fmt.Errorf("%w: %w", errDiskWrite, werr)
			}
			pos += int64(nr)
			f.mu.Lock()
			seg.Downloaded += int64(nr)
			if *ln != nil {
				(*ln).downloaded += int64(nr)
			}
			f.mu.Unlock()
		}
		if rerr != nil {
			if pos > end {
				break
			}
			if errors.Is(rerr, io.EOF) {
				return errShortTransfer
			}
			return rerr
		}
	}
	return nil
}

func (f *Fetcher) Patch(req *base.Request, opts *base.Options) error {
	if req == nil {
		return nil
	}
	if req.URL != "" {
		f.meta.Req.URL = req.URL
	}
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
	return nil
}

func (f *Fetcher) Pause() error {
	f.runMu.Lock()
	defer f.runMu.Unlock()
	if f.cancel != nil {
		f.cancel()
	}
	if f.runDone != nil {
		<-f.runDone
	}
	return nil
}

func (f *Fetcher) Close() error {
	return f.Pause()
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

// Stats reports one connection per worker that logged in, in the HTTP
// protocol's shape.
func (f *Fetcher) Stats() *fetcher.Stats {
	f.mu.Lock()
	defer f.mu.Unlock()
	var total int64
	if n := int64(len(f.lanes)); n > 0 && f.meta != nil && f.meta.Res != nil && f.meta.Res.Size > 0 {
		total = (f.meta.Res.Size + n - 1) / n
	}
	conns := make([]*pftp.StatsConnection, 0, len(f.lanes))
	for _, l := range f.lanes {
		conns = append(conns, &pftp.StatsConnection{
			Downloaded: l.downloaded,
			Total:      total,
			Completed:  l.completed || (f.done && !l.failed),
			Failed:     l.failed,
			RetryTimes: l.retryTimes,
		})
	}
	return &fetcher.Stats{Snapshot: &pftp.Stats{Connections: conns}}
}

// Progress returns the bytes downloaded of each selected file, in
// SelectFiles order.
func (f *Fetcher) Progress() fetcher.Progress {
	f.mu.Lock()
	defer f.mu.Unlock()
	perFile := map[int]int64{}
	for _, s := range f.segments {
		perFile[s.File] += s.Downloaded
	}
	sel := f.selected()
	p := make(fetcher.Progress, len(sel))
	for i, index := range sel {
		p[i] = perFile[index]
	}
	return p
}

func (f *Fetcher) Wait() error {
	return <-f.doneCh
}
