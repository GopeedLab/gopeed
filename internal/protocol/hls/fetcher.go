package hls

import (
	"bytes"
	"context"
	"encoding/json"
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

// fetcherState is the resolved download plan persisted for resume.
type fetcherState struct {
	MediaURL   string     `json:"mediaURL"`
	OutputName string     `json:"outputName"`
	Segments   []*Segment `json:"segments"`
}

// Stats is the protocol task statistics snapshot.
type Stats struct {
	SegmentsTotal  int `json:"segmentsTotal"`
	SegmentsDone   int `json:"segmentsDone"`
	SegmentsFailed int `json:"segmentsFailed"`
}

// tempDirPrefix separates HLS staging folders from regular task files.
const tempDirPrefix = ".gopeed-hls-"

type Fetcher struct {
	fetcher.DefaultFetcher
	meta   *fetcher.FetcherMeta
	config *config

	extra      *ReqExtra
	client     *http.Client
	impSession *httpclient.ImpersonationSession

	state *fetcherState

	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mergeMu sync.Mutex

	downloaded atomic.Int64 // network bytes received, seeded from the journal
	doneCount  atomic.Int64
	failCount  atomic.Int64

	keyCache   map[string][]byte
	keyCacheMu sync.Mutex

	journal     map[int64]int64 // plan index -> stored byte size
	journalPath string
	journalMu   sync.Mutex
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	// Initializes Ctl and DoneCh, which Wait() depends on.
	f.DefaultFetcher.Setup(ctl)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	f.impSession = httpclient.NewImpersonationSession()
	f.keyCache = make(map[string][]byte)
	f.config = &config{}
	f.Ctl.GetConfig(f.config)
	f.config.normalize()
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) (err error) {
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

	ctx := context.Background()
	content, finalURL, err := f.fetchText(ctx, f.playlistMethod(), req.URL)
	if err != nil {
		return fmt.Errorf("fetch playlist failed: %w", err)
	}
	baseURL, err := url.Parse(finalURL)
	if err != nil {
		return err
	}

	var media *Media
	mediaURL := finalURL
	if IsMasterPlaylist(content) {
		variants, err := ParseMaster(content, baseURL)
		if err != nil {
			return err
		}
		best := PickBestVariant(variants)
		variantContent, variantURL, err := f.fetchText(ctx, http.MethodGet, best.URI)
		if err != nil {
			return fmt.Errorf("fetch variant playlist failed: %w", err)
		}
		variantBase, err := url.Parse(variantURL)
		if err != nil {
			return err
		}
		content = variantContent
		baseURL = variantBase
		mediaURL = variantURL
	}
	media, err = ParseMedia(content, baseURL)
	if err != nil {
		return err
	}
	if media.Live {
		return errors.New("live streams are not supported yet")
	}

	outputName := DeriveOutputName(mediaURL, media)
	f.state = &fetcherState{
		MediaURL:   mediaURL,
		OutputName: outputName,
		Segments:   media.Segments,
	}
	if f.config.PrefetchContentLength {
		f.prefetchSize(media.Segments)
	}
	size := totalPlanSize(media.Segments)

	f.meta.Res = &base.Resource{
		Range: false,
		Files: []*base.FileInfo{
			{
				Name: outputName,
				Size: size,
			},
		},
	}
	f.meta.Res.Size = size
	return nil
}

func (f *Fetcher) Patch(req *base.Request, opts *base.Options) (err error) {
	urlChanged := f.meta.Req == nil || f.meta.Req.URL != req.URL
	f.meta.Req = req
	f.meta.Opts = opts
	f.extra = &ReqExtra{}
	if err = base.ParseReqExtra[ReqExtra](req); err != nil {
		return
	}
	if e, ok := req.Extra.(*ReqExtra); ok {
		f.extra = e
	}
	if urlChanged {
		// The playlist URL changed (e.g. refreshed signed link): drop the plan
		// so the engine re-resolves on next start (it does when Res is nil).
		f.state = nil
		f.meta.Res = nil
	}
	return
}

func (f *Fetcher) Start() error {
	if f.state == nil {
		return errors.New("hls fetcher is not resolved")
	}
	ctx, cancel := context.WithCancel(context.Background())
	f.ctx, f.cancel = ctx, cancel
	// Workers must be registered synchronously so a Pause() right after
	// Start() is guaranteed to wait on all of them.
	if err := f.startWorkers(); err != nil {
		return err
	}
	// The supervisor owns this run's context: a Pause/Start cycle creates a
	// fresh context and the previous supervisor must never act on it.
	go f.supervise(ctx)
	return nil
}

// Pause stops all segment downloads synchronously. A merge already in flight
// is local I/O and is awaited rather than aborted.
func (f *Fetcher) Pause() error {
	if f.cancel != nil {
		f.cancel()
	}
	f.wg.Wait()
	f.mergeMu.Lock()
	f.mergeMu.Unlock()
	return nil
}

func (f *Fetcher) Close() error {
	if f.cancel != nil {
		f.cancel()
	}
	if f.impSession != nil {
		f.impSession.Clear()
	}
	if f.state != nil && f.meta != nil {
		os.RemoveAll(f.tempDir())
	}
	return nil
}

// Progress reports network bytes received for the single output file.
func (f *Fetcher) Progress() fetcher.Progress {
	return fetcher.Progress{f.downloaded.Load()}
}

func (f *Fetcher) Stats() *fetcher.Stats {
	stats := &Stats{}
	if f.state != nil {
		stats.SegmentsTotal = len(f.state.Segments)
	}
	stats.SegmentsDone = int(f.doneCount.Load())
	stats.SegmentsFailed = int(f.failCount.Load())
	return &fetcher.Stats{Snapshot: stats}
}

// supervise waits for the segment workers, then merges. A canceled context
// exits silently (the engine is pausing), everything else is reported through
// DoneCh.
func (f *Fetcher) supervise(ctx context.Context) {
	f.wg.Wait()
	if ctx.Err() != nil {
		return
	}
	if failed := f.failCount.Load(); failed > 0 {
		f.DoneCh <- fmt.Errorf("%d segment(s) failed to download", failed)
		return
	}
	f.mergeMu.Lock()
	err := f.merge()
	f.mergeMu.Unlock()
	if err == nil {
		os.RemoveAll(f.tempDir())
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	f.DoneCh <- err
}

// startWorkers prepares the staging folder and launches the segment download
// workers. It returns immediately when every segment is already complete.
func (f *Fetcher) startWorkers() error {
	tempDir := f.tempDir()
	if err := os.MkdirAll(tempDir, 0777); err != nil {
		return err
	}
	f.journalPath = filepath.Join(tempDir, "journal.json")
	if err := f.loadJournal(); err != nil {
		return err
	}

	var pending []int64
	var baseBytes int64
	for i := range f.state.Segments {
		if size, ok := f.journal[int64(i)]; ok {
			if info, err := os.Stat(f.segmentPath(tempDir, int64(i))); err == nil && info.Size() == size {
				baseBytes += size
				f.doneCount.Add(1)
				continue
			}
		}
		pending = append(pending, int64(i))
	}
	f.downloaded.Store(baseBytes)
	if len(pending) == 0 {
		return nil
	}

	queue := make(chan int64)
	go func() {
		defer close(queue)
		for _, idx := range pending {
			select {
			case <-f.ctx.Done():
				return
			case queue <- idx:
			}
		}
	}()

	workers := f.config.SegmentConnections
	if workers > len(pending) {
		workers = len(pending)
	}
	for i := 0; i < workers; i++ {
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			for {
				select {
				case <-f.ctx.Done():
					return
				case idx, ok := <-queue:
					if !ok {
						return
					}
					if err := f.downloadSegment(idx); err != nil {
						if errors.Is(err, context.Canceled) {
							return
						}
						f.failCount.Add(1)
					}
				}
			}
		}()
	}
	return nil
}

func (f *Fetcher) downloadSegment(idx int64) error {
	seg := f.state.Segments[idx]
	data, err := f.fetchSegmentData(seg)
	if err != nil {
		return err
	}
	if seg.Key != nil {
		key, err := f.fetchKey(seg.Key)
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
	if err := f.storeSegment(idx, data); err != nil {
		return err
	}
	f.doneCount.Add(1)
	return nil
}

// fetchSegmentData downloads one segment with retries, returning its (still
// encrypted) content. Network bytes are counted for progress as they arrive.
func (f *Fetcher) fetchSegmentData(seg *Segment) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= f.config.MaxRetries; attempt++ {
		if err := f.ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			select {
			case <-f.ctx.Done():
				return nil, f.ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		data, err := f.tryGetSegment(seg)
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

func (f *Fetcher) tryGetSegment(seg *Segment) ([]byte, error) {
	resp, err := f.do(f.ctx, http.MethodGet, seg.URI, seg.Byterange)
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
			f.downloaded.Add(int64(n))
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

func (f *Fetcher) fetchKey(key *Key) ([]byte, error) {
	f.keyCacheMu.Lock()
	if cached, ok := f.keyCache[key.URI]; ok {
		f.keyCacheMu.Unlock()
		return cached, nil
	}
	f.keyCacheMu.Unlock()

	var data []byte
	var err error
	for attempt := 0; attempt <= f.config.MaxRetries; attempt++ {
		if err = f.ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			select {
			case <-f.ctx.Done():
				return nil, f.ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		resp, reqErr := f.do(f.ctx, http.MethodGet, key.URI, nil)
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
	f.keyCacheMu.Lock()
	f.keyCache[key.URI] = data
	f.keyCacheMu.Unlock()
	return data, nil
}

// merge concatenates the downloaded segments in plan order into the final
// output file. Segments are stored decrypted, so this is pure I/O.
func (f *Fetcher) merge() error {
	target := f.meta.SingleFilepath()
	if err := os.MkdirAll(filepath.Dir(target), 0777); err != nil {
		return err
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	var total int64
	for i := range f.state.Segments {
		data, err := os.ReadFile(f.segmentPath(f.tempDir(), int64(i)))
		if err != nil {
			return fmt.Errorf("read segment %d failed: %w", i, err)
		}
		if _, err := out.Write(data); err != nil {
			return err
		}
		total += int64(len(data))
	}
	f.meta.Res.Size = total
	if len(f.meta.Res.Files) > 0 && f.meta.Res.Files[0] != nil {
		f.meta.Res.Files[0].Size = total
	}
	return nil
}

func (f *Fetcher) storeSegment(idx int64, data []byte) error {
	tempDir := f.tempDir()
	partPath := f.segmentPath(tempDir, idx) + ".part"
	if err := os.WriteFile(partPath, data, 0666); err != nil {
		return err
	}
	if err := os.Rename(partPath, f.segmentPath(tempDir, idx)); err != nil {
		return err
	}
	f.journalMu.Lock()
	defer f.journalMu.Unlock()
	f.journal[idx] = int64(len(data))
	return f.saveJournal()
}

func (f *Fetcher) segmentPath(tempDir string, idx int64) string {
	return filepath.Join(tempDir, "seg-"+strconv.FormatInt(idx, 10))
}

// tempDir derives a stable staging folder from the playlist URL so resumes
// find their segments across app restarts.
func (f *Fetcher) tempDir() string {
	return filepath.Join(filepath.Dir(f.meta.SingleFilepath()), tempDirPrefix+sha1Hex(f.state.MediaURL)[:12])
}

func (f *Fetcher) loadJournal() error {
	f.journal = make(map[int64]int64)
	data, err := os.ReadFile(f.journalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	stored := make(map[string]int64)
	if err := json.Unmarshal(data, &stored); err != nil {
		// A broken journal must not kill a restartable download.
		return nil
	}
	for key, size := range stored {
		idx, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
		f.journal[idx] = size
	}
	return nil
}

func (f *Fetcher) saveJournal() error {
	stored := make(map[string]int64, len(f.journal))
	for idx, size := range f.journal {
		stored[strconv.FormatInt(idx, 10)] = size
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return os.WriteFile(f.journalPath, data, 0666)
}

// prefetchSize probes segment Content-Length via HEAD so the task can show a
// percentage. Any failure resets all sizes to zero (indeterminate progress).
func (f *Fetcher) prefetchSize(segments []*Segment) {
	var need []*Segment
	for _, seg := range segments {
		if seg.Byterange == nil {
			need = append(need, seg)
		}
	}
	if len(need) == 0 || len(need) > maxPrefetchSizeSegments {
		return
	}

	workerCount := f.config.SegmentConnections
	if workerCount > 8 {
		workerCount = 8
	}
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
	req, err := f.buildRequest(ctx, http.MethodHead, u, nil, nil)
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
	resp, err := f.do(ctx, method, u, nil)
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

func (f *Fetcher) do(ctx context.Context, method, u string, byterange *ByteRange) (*http.Response, error) {
	if method == "" {
		method = http.MethodGet
	}
	if f.client == nil {
		f.client = f.buildClient()
	}
	req, err := f.buildRequest(ctx, method, u, nil, byterange)
	if err != nil {
		return nil, err
	}
	return f.client.Do(req)
}

func (f *Fetcher) playlistMethod() string {
	if f.extra != nil && f.extra.Method != "" {
		return f.extra.Method
	}
	return http.MethodGet
}

func totalPlanSize(segments []*Segment) int64 {
	var total int64
	for _, seg := range segments {
		if seg.Size > 0 {
			total += seg.Size
		}
	}
	return total
}
