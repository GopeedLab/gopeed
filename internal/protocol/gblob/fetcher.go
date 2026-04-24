package gblob

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

type config struct{}

type Fetcher struct {
	ctl      *controller.Controller
	registry *Registry
	meta     *fetcher.FetcherMeta
	doneCh   chan error

	mu      sync.Mutex
	cancel  context.CancelFunc
	runDone chan struct{}
	paused  bool
	closed  bool

	downloaded   atomic.Int64
	lastSourceID string
}

type fetcherData struct {
	LastSourceID string `json:"lastSourceId"`
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	f.doneCh = make(chan error, 1)
}

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) error {
	if f.registry == nil {
		return ErrSourceNotFound
	}
	src, err := f.registry.Get(req.URL)
	if err != nil {
		return err
	}
	f.meta.Req = req
	f.meta.Opts = opts
	if f.meta.Opts == nil {
		f.meta.Opts = &base.Options{}
	}
	name := f.meta.Opts.Name
	if name == "" {
		name = src.ID
	}
	snapshot := src.Snapshot()
	size := snapshot.Written
	if snapshot.DeclaredSize > 0 {
		size = snapshot.DeclaredSize
	}
	if src.Type == SourceTypeWritableStream {
		// Writable streams may continue producing bytes after Resolve,
		// so treat the total size as unknown until the writer closes.
		if snapshot.DeclaredSize == 0 {
			size = 0
		}
	}
	f.meta.Res = &base.Resource{
		Size:  size,
		Range: snapshot.Range,
		Files: []*base.FileInfo{
			{
				Name: name,
				Path: "",
				Size: size,
				Req:  req,
			},
		},
	}
	return nil
}

func (f *Fetcher) Start() error {
	if f.registry == nil {
		return ErrSourceNotFound
	}
	currentSourceID, err := ParseURL(f.meta.Req.URL)
	if err != nil {
		return err
	}
	f.mu.Lock()
	if f.runDone != nil {
		f.mu.Unlock()
		return nil
	}
	f.paused = false
	runDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.runDone = runDone
	f.mu.Unlock()
	cleanupStartErr := func(err error) error {
		f.mu.Lock()
		if f.runDone == runDone {
			f.runDone = nil
			f.cancel = nil
		}
		f.mu.Unlock()
		return err
	}

	path := f.meta.SingleFilepath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return cleanupStartErr(err)
	}
	target, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return cleanupStartErr(err)
	}
	offset := int64(0)
	allowResumeAcrossSource := f.meta.Res != nil && f.meta.Res.Range
	if f.lastSourceID != "" && f.lastSourceID != currentSourceID && !allowResumeAcrossSource {
		if err := target.Truncate(0); err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
		if _, err := target.Seek(0, io.SeekStart); err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
	} else {
		stat, err := target.Stat()
		if err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
		offset = stat.Size()
		if _, err := target.Seek(offset, io.SeekStart); err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
	}
	f.lastSourceID = currentSourceID
	f.downloaded.Store(offset)

	sourceOffset := offset
	src, err := f.registry.Get(f.meta.Req.URL)
	if err != nil {
		target.Close()
		return cleanupStartErr(err)
	}
	snapshot := src.Snapshot()
	if snapshot.Written == 0 && !snapshot.Started {
		if f.meta.Res != nil && f.meta.Res.Range && offset > 0 {
			if err := f.registry.Reopen(f.meta.Req.URL, offset); err != nil {
				target.Close()
				return cleanupStartErr(err)
			}
			sourceOffset = 0
		} else if err := f.registry.StartSource(f.meta.Req.URL); err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
		src, err = f.registry.Get(f.meta.Req.URL)
		if err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
		snapshot = src.Snapshot()
	}
	if f.meta.Res != nil && f.meta.Res.Range && (snapshot.State == SourceStateAborted || snapshot.State == SourceStateFailed) {
		if err := f.registry.Reopen(f.meta.Req.URL, offset); err != nil {
			target.Close()
			return cleanupStartErr(err)
		}
		sourceOffset = 0
	}

	go f.copyLoop(ctx, target, offset, sourceOffset)
	return nil
}

func (f *Fetcher) copyLoop(ctx context.Context, target *os.File, targetOffset, sourceOffset int64) {
	var finalErr error

	defer target.Close()
	defer func() {
		f.mu.Lock()
		runDone := f.runDone
		f.runDone = nil
		f.cancel = nil
		paused := f.paused
		closed := f.closed
		f.mu.Unlock()
		close(runDone)
		if closed {
			select {
			case f.doneCh <- nil:
			default:
			}
			return
		}
		if paused {
			return
		}
		select {
		case f.doneCh <- finalErr:
		default:
		}
	}()

	src, err := f.registry.Get(f.meta.Req.URL)
	if err != nil {
		finalErr = err
		return
	}
	sourceFile, err := os.Open(src.Path)
	if err != nil {
		finalErr = err
		return
	}
	defer sourceFile.Close()

	buf := make([]byte, 32*1024)
	for {
		if err := f.registry.WaitForReadable(ctx, f.meta.Req.URL, sourceOffset); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			finalErr = err
			return
		}

		snapshot, serr := f.registry.Get(f.meta.Req.URL)
		if serr != nil {
			finalErr = serr
			return
		}
		current := snapshot.Snapshot()
		for sourceOffset < current.Written {
			remain := current.Written - sourceOffset
			readSize := int64(len(buf))
			if remain < readSize {
				readSize = remain
			}
			n, err := sourceFile.ReadAt(buf[:readSize], sourceOffset)
			if err != nil && !errors.Is(err, io.EOF) {
				finalErr = err
				return
			}
			if n == 0 {
				break
			}
			if _, err := target.Write(buf[:n]); err != nil {
				finalErr = err
				return
			}
			sourceOffset += int64(n)
			targetOffset += int64(n)
			f.downloaded.Store(targetOffset)
		}

		if current.State == SourceStateClosed && sourceOffset >= current.Written {
			if f.meta.Res != nil {
				if f.meta.Res.Size > 0 && targetOffset < f.meta.Res.Size {
					finalErr = io.ErrUnexpectedEOF
					return
				}
				if f.meta.Res.Size == 0 {
					f.meta.Res.Size = targetOffset
				}
				f.meta.Res.Files[0].Size = f.meta.Res.Size
			}
			finalErr = nil
			return
		}
		if current.State == SourceStateAborted || current.State == SourceStateFailed {
			err := current.Err
			if err == nil {
				err = ErrSourceAborted
			}
			finalErr = err
			return
		}
	}
}

func (f *Fetcher) Patch(req *base.Request, opts *base.Options) error {
	return nil
}

func (f *Fetcher) Pause() error {
	f.mu.Lock()
	f.paused = true
	cancel := f.cancel
	runDone := f.runDone
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if runDone != nil {
		<-runDone
	}
	return nil
}

func (f *Fetcher) Close() error {
	f.mu.Lock()
	f.closed = true
	cancel := f.cancel
	runDone := f.runDone
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if runDone != nil {
		<-runDone
	}
	select {
	case f.doneCh <- nil:
	default:
	}
	return nil
}

func (f *Fetcher) Stats() any {
	return map[string]any{}
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Progress() fetcher.Progress {
	return fetcher.Progress{f.downloaded.Load()}
}

func (f *Fetcher) Wait() error {
	return <-f.doneCh
}

type FetcherManager struct {
	registry *Registry
}

func (fm *FetcherManager) SetRegistry(registry *Registry) {
	fm.registry = registry
}

func (fm *FetcherManager) Name() string {
	return Scheme
}

func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: strings.ToUpper(Scheme),
		},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{
		registry: fm.registry,
	}
}

func (fm *FetcherManager) ParseName(u string) string {
	id, err := ParseURL(u)
	if err != nil {
		return ""
	}
	return id
}

func (fm *FetcherManager) AutoRename() bool {
	return true
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (data any, err error) {
	_f := f.(*Fetcher)
	return &fetcherData{
		LastSourceID: _f.lastSourceID,
	}, nil
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherData{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		fd := v.(*fetcherData)
		return &Fetcher{
			registry:     fm.registry,
			meta:         meta,
			doneCh:       make(chan error, 1),
			lastSourceID: fd.LastSourceID,
		}
	}
}

func (fm *FetcherManager) Close() error {
	return nil
}
