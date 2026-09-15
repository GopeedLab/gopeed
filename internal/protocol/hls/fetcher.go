package hls

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/httpclient"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// ReqExtra mirrors the HTTP protocol extra: the create dialog attaches HTTP
// style options to m3u8 URLs (same JSON keys), so the same shape is decoded.
type ReqExtra struct {
	Method string            `json:"method,omitempty"`
	Header map[string]string `json:"header,omitempty"`
	Body   string            `json:"body,omitempty"`
}

func copyReqExtra(extra *ReqExtra) *ReqExtra {
	copied := &ReqExtra{}
	if extra == nil {
		return copied
	}
	copied.Method = extra.Method
	copied.Body = extra.Body
	if extra.Header != nil {
		copied.Header = make(map[string]string, len(extra.Header))
		for key, value := range extra.Header {
			copied.Header[key] = value
		}
	}
	return copied
}

// fetcherState is the resolved download plan persisted for resume. StagingID
// gives the task its own staging folder (two tasks downloading the same URL
// into the same directory never share segment files) and survives restarts
// through the persisted state. StagingDir pins the absolute folder so user
// renames or moves of the task output do not orphan the staged segments.
type fetcherState struct {
	MediaURL   string     `json:"mediaURL"`
	OutputName string     `json:"outputName"`
	Segments   []*Segment `json:"segments"`
	StagingID  string     `json:"stagingID,omitempty"`
	StagingDir string     `json:"stagingDir,omitempty"`
}

// Stats is the protocol task statistics snapshot.
type Stats struct {
	SegmentsTotal  int `json:"segmentsTotal"`
	SegmentsDone   int `json:"segmentsDone"`
	SegmentsFailed int `json:"segmentsFailed"`
}

// tempDirPrefix separates HLS staging folders from regular task files.
const tempDirPrefix = ".gopeed-hls-"

// hlsRun holds every piece of state that belongs to one Start..Pause/complete
// cycle. Runs never share mutable state: a Pause that is immediately followed
// by a Start replaces the whole run, so workers of the previous run cannot
// race with the next one. The run also snapshots the plan, request headers,
// client and merge target so concurrent Patch calls cannot change them under
// a running download.
type hlsRun struct {
	ctx    context.Context
	cancel context.CancelFunc
	// wg covers the queue producer AND the segment workers: Pause must not
	// return while the producer can still hand out segment indexes.
	wg   sync.WaitGroup
	done chan struct{} // closed when the supervisor has fully exited

	segs       []*Segment // plan snapshot, read-only for the duration of the run
	extra      *ReqExtra  // header snapshot (deep copy)
	client     *http.Client
	outputPath string // merge target snapshot
	tempDir    string

	keyCache   map[string][]byte
	keyCacheMu sync.Mutex

	stagingID   string // identity of the staging folder, bound into the journal
	planHash    string // fingerprint of the download plan, validated on resume
	journal     map[int64]int64
	journalPath string
	journalMu   sync.Mutex

	// downloaded counts bytes of the final output that are safely on disk
	// (completed segment files), not raw network traffic: discarded retry
	// bytes must never show up as progress.
	downloaded atomic.Int64
	doneCount  atomic.Int64
	failCount  atomic.Int64
	baseBytes  int64 // stored bytes restored from the journal at startup
	baseDone   int   // completed segments restored from the journal at startup
}

func (r *hlsRun) statsSnapshot() Stats {
	return Stats{
		SegmentsTotal:  len(r.segs),
		SegmentsDone:   r.baseDone + int(r.doneCount.Load()),
		SegmentsFailed: int(r.failCount.Load()),
	}
}

type Fetcher struct {
	fetcher.DefaultFetcher
	meta   *fetcher.FetcherMeta
	config *config

	// mu guards the fields below and every lifecycle transition. Progress and
	// Stats are read from the engine's checkpoint goroutine while a run is
	// active; they only take mu to snapshot the active run pointer, then read
	// run-owned atomics lock-free.
	mu             sync.Mutex
	extra          *ReqExtra
	client         *http.Client
	state          *fetcherState
	activeRun      *hlsRun
	closed         bool
	lastStats      Stats
	lastDownloaded int64

	impSession *httpclient.ImpersonationSession
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	// Initializes Ctl and DoneCh, which Wait() depends on.
	f.DefaultFetcher.Setup(ctl)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	f.impSession = httpclient.NewImpersonationSession()
	f.config = &config{}
	f.Ctl.GetConfig(f.config)
	f.config.normalize()
	// Restored fetchers receive their request through Setup (the engine calls
	// it after the Restore builder): parse the extra headers so resumed
	// segment/key requests keep Referer/Cookie and other anti-hotlink headers.
	f.mu.Lock()
	f.reloadExtraLocked()
	f.mu.Unlock()
}

// reloadExtraLocked refreshes f.extra from the typed extra of meta.Req.
func (f *Fetcher) reloadExtraLocked() {
	if f.meta == nil || f.meta.Req == nil {
		return
	}
	if err := base.ParseReqExtra[ReqExtra](f.meta.Req); err != nil {
		return
	}
	if e, ok := f.meta.Req.Extra.(*ReqExtra); ok {
		f.extra = e
	}
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.meta.Req = req
	f.meta.Opts = opts
	f.extra = &ReqExtra{}
	if err = base.ParseReqExtra[ReqExtra](req); err != nil {
		return
	}
	if e, ok := req.Extra.(*ReqExtra); ok {
		f.extra = e
	}
	if f.client == nil {
		f.client = f.buildClient()
	}

	state, name, err := f.resolvePlanLocked(context.Background(), "")
	if err != nil {
		return err
	}
	state.StagingID = randomStagingID()
	f.state = state
	// Pin the staging folder now so the very first persisted state already
	// carries it: a crash right after create still resumes into the same
	// folder.
	f.resolveStagingDirLocked()
	size := planTotalSize(state.Segments)
	f.meta.Res = &base.Resource{
		Range: false,
		Files: []*base.FileInfo{
			{
				Name: name,
				Size: size,
			},
		},
		Size: size,
	}
	return nil
}

// resolvePlanLocked fetches and parses the playlist into a fresh state. It
// must be called with f.mu held while the fetcher is quiescent. When keepName
// is non-empty the output keeps that file name (used when re-resolving an
// existing task so a refreshed URL does not rename the file).
func (f *Fetcher) resolvePlanLocked(ctx context.Context, keepName string) (*fetcherState, string, error) {
	content, finalURL, err := f.fetchText(ctx, f.playlistMethod(), f.meta.Req.URL)
	if err != nil {
		return nil, "", fmt.Errorf("fetch playlist failed: %w", err)
	}
	baseURL, err := url.Parse(finalURL)
	if err != nil {
		return nil, "", err
	}

	mediaURL := finalURL
	if IsMasterPlaylist(content) {
		master, err := ParseMasterPlaylist(content, baseURL)
		if err != nil {
			return nil, "", err
		}
		best := PickBestVariant(master.Variants)
		if err := checkAudioRendition(master, best); err != nil {
			return nil, "", err
		}
		variantContent, variantURL, err := f.fetchText(ctx, http.MethodGet, best.URI)
		if err != nil {
			return nil, "", fmt.Errorf("fetch variant playlist failed: %w", err)
		}
		variantBase, err := url.Parse(variantURL)
		if err != nil {
			return nil, "", err
		}
		content = variantContent
		baseURL = variantBase
		mediaURL = variantURL
	}
	media, err := ParseMedia(content, baseURL)
	if err != nil {
		return nil, "", err
	}
	if err := validateSupported(media); err != nil {
		return nil, "", err
	}

	outputName := keepName
	if outputName == "" {
		outputName = DeriveOutputName(mediaURL, media)
	}
	state := &fetcherState{
		MediaURL:   mediaURL,
		OutputName: outputName,
		Segments:   media.Segments,
	}
	if f.config.PrefetchContentLength {
		f.prefetchSize(media.Segments)
	}
	return state, outputName, nil
}

func (f *Fetcher) Patch(req *base.Request, opts *base.Options) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.activeRun != nil {
		return errors.New("task is running; pause it before changing its settings")
	}
	if req != nil {
		cur := f.meta.Req
		if cur == nil {
			f.meta.Req = req
			cur = req
		}
		if req.URL != "" && req.URL != cur.URL {
			cur.URL = req.URL
			// The playlist URL changed (e.g. refreshed signed link): drop the
			// plan so the engine re-resolves on next start (it does when Res
			// is nil). Only explicitly supplied fields are applied, so a
			// header-only or rename-only Patch keeps the resolved plan and
			// the output path.
			f.dropPlanLocked()
		}
		if err := mergeReqExtra(cur, req); err != nil {
			return err
		}
		if req.Proxy != nil && cur.Proxy != req.Proxy {
			cur.Proxy = req.Proxy
			// Transport settings changed: rebuild the client before the
			// next run.
			f.client = nil
		}
		if len(req.Labels) > 0 {
			if cur.Labels == nil {
				cur.Labels = make(map[string]string, len(req.Labels))
			}
			for key, value := range req.Labels {
				cur.Labels[key] = value
			}
		}
		// SkipVerifyCert is a plain bool, so a partial update cannot tell an
		// absent field from an explicit false: it is intentionally not merged
		// (the HTTP protocol Patch ignores it as well).
	}
	if opts != nil {
		if f.meta.Opts == nil {
			f.meta.Opts = opts
		} else {
			if opts.Name != "" {
				f.meta.Opts.Name = opts.Name
			}
			if opts.Path != "" {
				f.meta.Opts.Path = opts.Path
			}
		}
	}
	f.reloadExtraLocked()
	return nil
}

// mergeReqExtra merges the typed extra of incoming into cur following the
// HTTP protocol semantics: Method/Body are overridden when non-empty and
// headers are merged per key.
func mergeReqExtra(cur *base.Request, incoming *base.Request) error {
	if incoming.Extra == nil {
		return nil
	}
	if err := base.ParseReqExtra[ReqExtra](incoming); err != nil {
		return err
	}
	inc, ok := incoming.Extra.(*ReqExtra)
	if !ok {
		return nil
	}
	if err := base.ParseReqExtra[ReqExtra](cur); err != nil {
		return err
	}
	target, ok := cur.Extra.(*ReqExtra)
	if !ok {
		target = &ReqExtra{}
		cur.Extra = target
	}
	if inc.Method != "" {
		target.Method = inc.Method
	}
	if inc.Body != "" {
		target.Body = inc.Body
	}
	if len(inc.Header) > 0 {
		if target.Header == nil {
			target.Header = make(map[string]string, len(inc.Header))
		}
		for key, value := range inc.Header {
			target.Header[key] = value
		}
	}
	return nil
}

// dropPlanLocked discards the resolved plan after a URL change. The staging
// folder of the dropped plan is removed too: this only runs while the
// fetcher is quiescent, so no run can still be writing into it.
func (f *Fetcher) dropPlanLocked() {
	if f.state != nil && f.state.StagingDir != "" {
		os.RemoveAll(f.state.StagingDir)
	}
	f.state = nil
	f.meta.Res = nil
}

// existingOutputNameLocked returns the output file name the task already
// uses, so a re-resolve does not rename an in-progress download.
func (f *Fetcher) existingOutputNameLocked() string {
	if f.meta.Opts != nil && f.meta.Opts.Name != "" {
		return f.meta.Opts.Name
	}
	if res := f.meta.Res; res != nil && len(res.Files) > 0 && res.Files[0] != nil {
		return res.Files[0].Name
	}
	return ""
}

func (f *Fetcher) Start() error {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return errors.New("hls fetcher is closed")
	}
	if f.activeRun != nil {
		f.mu.Unlock()
		return errors.New("hls fetcher already running")
	}
	if f.state == nil {
		if f.meta.Req == nil || f.meta.Res == nil {
			f.mu.Unlock()
			return errors.New("hls fetcher is not resolved")
		}
		// Crash window: the engine persists the task (with its resource)
		// before the first HLS state checkpoint lands, so a restored task
		// can have a resource but no plan. Rebuild the plan instead of
		// failing, keeping the existing output name.
		if f.client == nil {
			f.client = f.buildClient()
		}
		state, _, err := f.resolvePlanLocked(context.Background(), f.existingOutputNameLocked())
		if err != nil {
			f.mu.Unlock()
			return fmt.Errorf("re-resolve failed: %w", err)
		}
		f.state = state
		size := planTotalSize(state.Segments)
		if res := f.meta.Res; res != nil {
			res.Size = size
			if len(res.Files) > 0 && res.Files[0] != nil {
				res.Files[0].Size = size
			}
		}
	}
	// The client is built here, before any worker exists, so segment
	// goroutines never race on lazy initialization.
	if f.client == nil {
		f.client = f.buildClient()
	}

	ctx, cancel := context.WithCancel(context.Background())
	run := &hlsRun{
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		segs:       f.state.Segments,
		extra:      copyReqExtra(f.extra),
		client:     f.client,
		outputPath: f.meta.SingleFilepath(),
		stagingID:  f.state.StagingID,
		planHash:   planFingerprint(f.state),
		tempDir:    f.resolveStagingDirLocked(),
		keyCache:   make(map[string][]byte),
		journal:    make(map[int64]int64),
	}
	f.activeRun = run
	f.mu.Unlock()

	if err := f.startRun(run); err != nil {
		// No supervisor was started for this run, so close its done channel
		// here to release anyone waiting on it.
		run.cancel()
		close(run.done)
		f.clearActiveRun(run)
		return err
	}
	go f.supervise(run)
	return nil
}

// Pause stops all segment downloads synchronously. It waits for the queue
// producer, every worker AND the supervisor (including a merge in flight)
// before returning, so a Store right after Pause sees a consistent state.
func (f *Fetcher) Pause() error {
	f.mu.Lock()
	run := f.activeRun
	f.mu.Unlock()
	if run == nil {
		return nil
	}
	run.cancel()
	<-run.done
	f.clearActiveRun(run)
	return nil
}

// Close pauses any active run and only removes the staging folder once every
// run goroutine has stopped writing to it.
func (f *Fetcher) Close() error {
	f.mu.Lock()
	f.closed = true
	run := f.activeRun
	tempDir := ""
	if f.state != nil {
		tempDir = f.resolveStagingDirLocked()
	}
	f.mu.Unlock()

	if run != nil {
		run.cancel()
		<-run.done
		f.clearActiveRun(run)
	}
	if f.impSession != nil {
		f.impSession.Clear()
	}
	if tempDir != "" {
		os.RemoveAll(tempDir)
	}
	return nil
}

// clearActiveRun detaches a finished run and keeps its counters as the last
// known values for idle Stats/Progress reporting.
func (f *Fetcher) clearActiveRun(run *hlsRun) {
	f.mu.Lock()
	if f.activeRun == run {
		f.activeRun = nil
		f.lastStats = run.statsSnapshot()
		f.lastDownloaded = run.downloaded.Load()
	}
	f.mu.Unlock()
}

// Progress reports final-output bytes stored on disk for the single file.
func (f *Fetcher) Progress() fetcher.Progress {
	f.mu.Lock()
	run := f.activeRun
	if run != nil {
		f.mu.Unlock()
		return fetcher.Progress{run.downloaded.Load()}
	}
	last := f.lastDownloaded
	f.mu.Unlock()
	return fetcher.Progress{last}
}

func (f *Fetcher) Stats() *fetcher.Stats {
	f.mu.Lock()
	run := f.activeRun
	state := f.state
	if run != nil {
		f.mu.Unlock()
		snap := run.statsSnapshot()
		return &fetcher.Stats{Snapshot: &snap}
	}
	snap := f.lastStats
	f.mu.Unlock()
	if snap.SegmentsTotal == 0 && state != nil {
		snap.SegmentsTotal = len(state.Segments)
	}
	return &fetcher.Stats{Snapshot: &snap}
}

// supervise waits for the segment workers, then merges. A canceled context
// exits silently (the engine is pausing), everything else is reported through
// DoneCh exactly once per run.
func (f *Fetcher) supervise(run *hlsRun) {
	run.wg.Wait()
	defer close(run.done)

	if run.ctx.Err() != nil {
		f.clearActiveRun(run)
		return
	}
	var err error
	if failed := run.failCount.Load(); failed > 0 {
		err = fmt.Errorf("%d segment(s) failed to download", failed)
	} else {
		err = f.merge(run)
		if err == nil {
			os.RemoveAll(run.tempDir)
		}
		if errors.Is(err, context.Canceled) {
			f.clearActiveRun(run)
			return
		}
	}
	f.clearActiveRun(run)
	select {
	case f.DoneCh <- err:
	default:
	}
}

// startRun prepares the staging folder and launches the segment download
// workers. It returns immediately when every segment is already complete.
func (f *Fetcher) startRun(run *hlsRun) error {
	if err := os.MkdirAll(run.tempDir, 0777); err != nil {
		return err
	}
	run.journalPath = filepath.Join(run.tempDir, "journal.json")
	if err := loadJournal(run); err != nil {
		return err
	}

	var pending []int64
	for i := range run.segs {
		if size, ok := run.journal[int64(i)]; ok {
			if info, err := os.Stat(segmentPath(run.tempDir, int64(i))); err == nil && info.Size() == size {
				run.baseBytes += size
				run.baseDone++
				continue
			}
		}
		pending = append(pending, int64(i))
	}
	run.downloaded.Store(run.baseBytes)
	if len(pending) == 0 {
		return nil
	}

	queue := make(chan int64)
	run.wg.Add(1)
	go func() {
		defer run.wg.Done()
		defer close(queue)
		for _, idx := range pending {
			select {
			case <-run.ctx.Done():
				return
			case queue <- idx:
			}
		}
	}()

	workers := min(f.config.SegmentConnections, len(pending))
	for i := 0; i < workers; i++ {
		run.wg.Add(1)
		go func() {
			defer run.wg.Done()
			for {
				select {
				case <-run.ctx.Done():
					return
				case idx, ok := <-queue:
					if !ok {
						return
					}
					if err := f.downloadSegment(run, idx); err != nil {
						if errors.Is(err, context.Canceled) {
							return
						}
						run.failCount.Add(1)
					}
				}
			}
		}()
	}
	return nil
}

func (f *Fetcher) downloadSegment(run *hlsRun, idx int64) error {
	seg := run.segs[idx]
	data, err := f.fetchSegmentData(run, seg)
	if err != nil {
		return err
	}
	if seg.Key != nil {
		key, err := f.fetchKey(run, seg.Key)
		if err != nil {
			return err
		}
		iv := seg.Key.IV
		if iv == nil {
			iv = sequenceIV(seg.Sequence)
		}
		data, err = decryptAES128(data, key, iv)
		if err != nil {
			return err
		}
	}
	if err := storeSegment(run, idx, data); err != nil {
		return err
	}
	run.doneCount.Add(1)
	run.downloaded.Add(int64(len(data)))
	return nil
}

// fetchSegmentData downloads one segment with retries, returning its (still
// encrypted) content.
func (f *Fetcher) fetchSegmentData(run *hlsRun, seg *Segment) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= f.config.MaxRetries; attempt++ {
		if err := run.ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			select {
			case <-run.ctx.Done():
				return nil, run.ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		data, err := f.tryGetSegment(run, seg)
		if err == nil {
			return data, nil
		}
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func (f *Fetcher) tryGetSegment(run *hlsRun, seg *Segment) ([]byte, error) {
	resp, err := f.do(run.ctx, run.client, run.extra, http.MethodGet, seg.URI, seg.Byterange)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("segment %s: unexpected status %d", seg.URI, resp.StatusCode)
	}
	reader := &stallReader{rc: resp.Body, timeout: f.timeout()}
	var buf bytes.Buffer
	chunk := make([]byte, 64<<10)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
	}
	if seg.Byterange != nil && int64(buf.Len()) != seg.Byterange.Length {
		return nil, fmt.Errorf("segment %s: size mismatch, got %d want %d", seg.URI, buf.Len(), seg.Byterange.Length)
	}
	return buf.Bytes(), nil
}

func (f *Fetcher) fetchKey(run *hlsRun, key *Key) ([]byte, error) {
	run.keyCacheMu.Lock()
	if cached, ok := run.keyCache[key.URI]; ok {
		run.keyCacheMu.Unlock()
		return cached, nil
	}
	run.keyCacheMu.Unlock()

	var data []byte
	var err error
	for attempt := 0; attempt <= f.config.MaxRetries; attempt++ {
		if err = run.ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			select {
			case <-run.ctx.Done():
				return nil, run.ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		resp, reqErr := f.do(run.ctx, run.client, run.extra, http.MethodGet, key.URI, nil)
		if reqErr != nil {
			err = reqErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			err = fmt.Errorf("key %s: unexpected status %d", key.URI, resp.StatusCode)
			continue
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	if len(data) != 16 {
		return nil, fmt.Errorf("key %s: invalid key size %d", key.URI, len(data))
	}
	run.keyCacheMu.Lock()
	run.keyCache[key.URI] = data
	run.keyCacheMu.Unlock()
	return data, nil
}

// merge concatenates the downloaded segments in plan order into the final
// output file. Segments are stored decrypted, so this is pure I/O. The merge
// writes to a unique temporary file next to the target, checks write and
// close errors, and only then replaces the target: a failed merge leaves the
// staged segments intact so it can be retried, and the unique part name keeps
// concurrent tasks that share an output path from clobbering each other. The
// final size is published through Progress: the engine backfills Res.Size
// from Progress().TotalDownloaded() on completion.
func (f *Fetcher) merge(run *hlsRun) error {
	target := run.outputPath
	if err := os.MkdirAll(filepath.Dir(target), 0777); err != nil {
		return err
	}
	partPath := target + "." + tempDirPrefix + run.stagingID + ".part"
	out, err := os.Create(partPath)
	if err != nil {
		return err
	}
	var total int64
	for i := range run.segs {
		data, err := os.ReadFile(segmentPath(run.tempDir, int64(i)))
		if err != nil {
			out.Close()
			os.Remove(partPath)
			return fmt.Errorf("read segment %d failed: %w", i, err)
		}
		if _, err := out.Write(data); err != nil {
			out.Close()
			os.Remove(partPath)
			return err
		}
		total += int64(len(data))
	}
	if err := out.Close(); err != nil {
		os.Remove(partPath)
		return err
	}
	// Remove-then-rename keeps this idempotent: if a previous merge already
	// renamed its output before a crash lost the bookkeeping, re-merging
	// must still succeed.
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		os.Remove(partPath)
		return err
	}
	if err := os.Rename(partPath, target); err != nil {
		os.Remove(partPath)
		return err
	}
	run.downloaded.Store(total)
	return nil
}

func storeSegment(run *hlsRun, idx int64, data []byte) error {
	partPath := segmentPath(run.tempDir, idx) + ".part"
	if err := os.WriteFile(partPath, data, 0666); err != nil {
		return err
	}
	if err := os.Rename(partPath, segmentPath(run.tempDir, idx)); err != nil {
		return err
	}
	run.journalMu.Lock()
	defer run.journalMu.Unlock()
	run.journal[idx] = int64(len(data))
	return saveJournal(run)
}

func segmentPath(tempDir string, idx int64) string {
	return filepath.Join(tempDir, "seg-"+strconv.FormatInt(idx, 10))
}

// resolveStagingDirLocked returns the staging folder of the current plan,
// deriving and pinning it on first use so later output renames or moves do
// not orphan the staged segments. It lives next to the task output, which is
// exactly opts.Path for a single-file task. Must be called with f.mu held.
func (f *Fetcher) resolveStagingDirLocked() string {
	if f.state.StagingDir == "" {
		if f.state.StagingID == "" {
			// Legacy state (or a hand-crafted plan): migrate to a fresh task
			// owned identity. The old URL-hash folder is deliberately not
			// reused: its contents cannot be proven to belong to this plan,
			// and sharing it with another same-URL task is exactly the bug
			// being fixed. It is left behind as orphaned temp data.
			f.state.StagingID = randomStagingID()
		}
		dir := ""
		if f.meta.Opts != nil {
			dir = f.meta.Opts.Path
		}
		f.state.StagingDir = filepath.Join(dir, tempDirPrefix+f.state.StagingID)
	}
	return f.state.StagingDir
}

// prefetchSize probes segment Content-Length via HEAD. Any failure resets all
// sizes to zero: a partial size estimate would corrupt progress reporting.
func (f *Fetcher) prefetchSize(segments []*Segment) {
	var need []*Segment
	for _, seg := range segments {
		if seg.Size <= 0 {
			need = append(need, seg)
		}
	}
	if len(need) == 0 || len(need) > maxPrefetchSizeSegments {
		return
	}

	workerCount := min(f.config.SegmentConnections, 8)
	var failed atomic.Bool
	queue := make(chan *Segment)
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for seg := range queue {
				ctx, cancel := context.WithTimeout(context.Background(), f.timeout())
				seg.Size = f.headContentLength(ctx, seg.URI)
				cancel()
				if seg.Size <= 0 {
					failed.Store(true)
				}
			}
		}()
	}
	for _, seg := range need {
		queue <- seg
	}
	close(queue)
	wg.Wait()
	if failed.Load() {
		for _, seg := range segments {
			seg.Size = 0
		}
	}
}

func (f *Fetcher) headContentLength(ctx context.Context, u string) int64 {
	req, err := buildRequest(ctx, f.extra, http.MethodHead, u, nil, nil)
	if err != nil {
		return 0
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0
	}
	return resp.ContentLength
}

func (f *Fetcher) fetchText(ctx context.Context, method, u string) (string, string, error) {
	resp, err := f.do(ctx, f.client, f.extra, method, u, nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, u)
	}
	data, err := io.ReadAll(io.LimitReader(&stallReader{rc: resp.Body, timeout: f.timeout()}, maxPlaylistSize))
	if err != nil {
		return "", "", err
	}
	return string(data), resp.Request.URL.String(), nil
}

func (f *Fetcher) do(ctx context.Context, client *http.Client, extra *ReqExtra, method, u string, byterange *ByteRange) (*http.Response, error) {
	if method == "" {
		method = http.MethodGet
	}
	req, err := buildRequest(ctx, extra, method, u, nil, byterange)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (f *Fetcher) playlistMethod() string {
	if f.extra != nil && f.extra.Method != "" {
		return f.extra.Method
	}
	return http.MethodGet
}

// planTotalSize sums the known segment sizes. It returns 0 unless every
// segment is unencrypted with a known size: ciphertext lengths overstate the
// final output (CBC padding is removed on decrypt), and partial estimates
// would make progress stall below 100%. For indeterminate plans the engine
// backfills Res.Size from Progress().TotalDownloaded() when the task
// completes (downloader.go watch).
func planTotalSize(segments []*Segment) int64 {
	var total int64
	for _, seg := range segments {
		if seg.Key != nil {
			return 0
		}
		if seg.Size <= 0 {
			return 0
		}
		total += seg.Size
	}
	return total
}
