package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/GopeedLab/gopeed/internal/tempfiles"
)

// HTTPSpool downloads ahead independently of the FFmpeg semaphore. Range
// sources can fetch a missing block on demand, including blocks already evicted.
type HTTPSpool struct {
	demand     chan int64
	demandDone chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	source     Input
	disk       *DiskInput
	network    sync.Mutex
	readMu     sync.Mutex
	pos        int64
	done       chan struct{}
}

func OpenSpoolingHTTP(ctx context.Context, client *http.Client, source HTTPSource, root string, owners ...*tempfiles.Scope) (Input, error) {
	ctx, cancel := context.WithCancel(ctx)
	in, err := OpenHTTP(ctx, client, source)
	if err != nil {
		cancel()
		return nil, err
	}
	disk, err := NewDiskInput(ctx, root, owners...)
	if err != nil {
		cancel()
		in.Close()
		return nil, err
	}
	if ranged := in.(*httpInput); ranged.ranged {
		if _, err = disk.spool.WriteAt(ranged.block, 0); err != nil {
			cancel()
			in.Close()
			disk.Close()
			return nil, err
		}
		ranged.block = nil
	}
	h := &HTTPSpool{ctx: ctx, cancel: cancel, source: in, disk: disk, done: make(chan struct{}), demand: make(chan int64, 1), demandDone: make(chan struct{})}
	go func() {
		defer close(h.demandDone)
		for {
			select {
			case offset := <-h.demand:
				// A prefetch may have filled this position while demand was queued.
				var probe [1]byte
				if n, err, _ := h.disk.spool.readAt(probe[:], offset); n > 0 || err != nil {
					continue
				}
				if err := h.fetch(offset / SpoolBlockSize * SpoolBlockSize); err != nil {
					h.disk.End(err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		defer close(h.done)
		var err error
		if !in.Seekable() {
			_, err = io.CopyBuffer(disk, in, make([]byte, 64*1024))
		} else {
			for offset := int64(0); offset < in.Size(); offset += 3 * SpoolBlockSize {
				if err = h.fetch(offset); err != nil {
					break
				}
			}
		}
		disk.End(err)
	}()
	return h, nil
}
func (h *HTTPSpool) fetch(offset int64) error {
	h.network.Lock()
	defer h.network.Unlock()
	if err := h.ctx.Err(); err != nil {
		return err
	}
	source := h.source.(*httpInput)
	end := offset + min(3*SpoolBlockSize, h.source.Size()-offset)
	buffer := make([]byte, 64*1024)
	for offset < end {
		n, err, _ := h.disk.spool.readAt(buffer[:min(int64(len(buffer)), end-offset)], offset)
		if err != nil {
			return err
		}
		if n > 0 {
			offset += int64(n)
			continue
		}
		resp, err := source.request(offset, end-1)
		if err != nil {
			return err
		}
		err = func() error {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusPartialContent {
				return fmt.Errorf("ffmpeg: Range request returned %d; source changed or stopped supporting Range", resp.StatusCode)
			}
			start, last, size, err := contentRange(resp.Header.Get("Content-Range"))
			if err != nil || start != offset || last >= end || size != source.size {
				return errors.New("ffmpeg: inconsistent Content-Range")
			}
			if source.etag != "" && resp.Header.Get("ETag") != source.etag {
				return errors.New("ffmpeg: HTTP input ETag changed")
			}
			if source.etag == "" && source.modified != "" && resp.Header.Get("Last-Modified") != source.modified {
				return errors.New("ffmpeg: HTTP input Last-Modified changed")
			}
			if err := identityEncoding(resp); err != nil {
				return err
			}
			remaining := last - start + 1
			if resp.ContentLength >= 0 && resp.ContentLength != remaining {
				return errors.New("ffmpeg: Range response length mismatch")
			}
			for remaining > 0 {
				n, err := resp.Body.Read(buffer[:min(int64(len(buffer)), remaining)])
				if n > 0 {
					if _, e := h.disk.spool.WriteAt(buffer[:n], offset); e != nil {
						return e
					}
					offset += int64(n)
					remaining -= int64(n)
				}
				if err != nil {
					if err == io.EOF && remaining == 0 {
						return nil
					}
					if err == io.EOF {
						return io.ErrUnexpectedEOF
					}
					return err
				}
				if n == 0 {
					return io.ErrNoProgress
				}
			}
			n, err := resp.Body.Read(buffer[:1])
			if n != 0 || err != io.EOF {
				if err != nil && err != io.EOF {
					return err
				}
				return errors.New("ffmpeg: oversized Range response")
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
func (h *HTTPSpool) Read(p []byte) (int, error) {
	if !h.Seekable() {
		return h.disk.Read(p)
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	if h.pos >= h.Size() {
		if err := h.disk.spool.discard((h.Size() - 1) / SpoolBlockSize); err != nil {
			return 0, err
		}
		return 0, io.EOF
	}
	for {
		if err := h.ctx.Err(); err != nil {
			return 0, err
		}
		n, err, wake := h.disk.spool.readAt(p, h.pos)
		if n > 0 {
			block := h.pos / SpoolBlockSize
			h.pos += int64(n)
			if h.pos/SpoolBlockSize > block {
				if e := h.disk.spool.discard(block); err == nil {
					err = e
				}
			}
			return n, err
		}
		if err != nil {
			return 0, err
		}
		select {
		case h.demand <- h.pos:
		default:
		}
		// Read as soon as a partial response supplies these bytes, rather than
		// waiting for its entire multi-block Range request to finish.
		select {
		case <-wake:
		case <-h.ctx.Done():
			return 0, h.ctx.Err()
		}
	}
}
func (h *HTTPSpool) Seek(offset int64, whence int) (int64, error) {
	if !h.Seekable() {
		return 0, ErrSeek
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	base := int64(0)
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = h.pos
	case io.SeekEnd:
		base = h.Size()
	default:
		return 0, errors.New("invalid seek origin")
	}
	next := base + offset
	if next < 0 || (offset > 0 && next < base) {
		return 0, errors.New("invalid seek offset")
	}
	h.pos = next
	return next, nil
}
func (h *HTTPSpool) Size() int64              { return h.source.Size() }
func (h *HTTPSpool) Seekable() bool           { return h.source.Seekable() }
func (h *HTTPSpool) Progress() (int64, int64) { return h.disk.Progress() }
func (h *HTTPSpool) Close() error {
	h.cancel()
	h.source.Close()
	<-h.done
	<-h.demandDone
	return h.disk.Close()
}

func (h *HTTPSpool) Failures() <-chan error { return h.disk.Failures() }
