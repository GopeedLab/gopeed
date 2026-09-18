package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/GopeedLab/gopeed/internal/tempfiles"
)

const SpoolBlockSize int64 = 8 * 1024 * 1024

type byteSpan struct{ start, end int64 }

// blockSpool tracks valid byte intervals separately from file length: a sparse
// file's unwritten holes are never valid media. No file handles survive a call.
type blockSpool struct {
	owner                *tempfiles.Scope
	failures             chan error
	mu                   sync.Mutex
	dir                  string
	blocks               map[int64][]byteSpan
	seen                 []byteSpan
	wake                 chan struct{}
	ended, closed        bool
	err                  error
	downloaded, received int64
}

func newBlockSpool(root string, owners ...*tempfiles.Scope) (*blockSpool, error) {
	if root != "" {
		if err := os.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
	}
	dir, err := os.MkdirTemp(root, "gopeed-media-")
	if err != nil {
		return nil, err
	}
	var owner *tempfiles.Scope
	if len(owners) > 0 {
		owner = owners[0]
	}
	if err := owner.Track(dir); err != nil {
		return nil, err
	}
	return &blockSpool{owner: owner, failures: make(chan error, 1), dir: dir, blocks: make(map[int64][]byteSpan), wake: make(chan struct{})}, nil
}
func (s *blockSpool) notify()                 { close(s.wake); s.wake = make(chan struct{}) }
func (s *blockSpool) path(index int64) string { return filepath.Join(s.dir, fmt.Sprintf("%d", index)) }
func addSpan(spans []byteSpan, next byteSpan) ([]byteSpan, int64) {
	before := int64(0)
	for _, span := range spans {
		before += span.end - span.start
	}
	out := make([]byteSpan, 0, len(spans)+1)
	for _, span := range spans {
		if span.end < next.start {
			out = append(out, span)
		} else if next.end < span.start {
			out = append(out, next)
			next = span
		} else {
			next.start = min(next.start, span.start)
			next.end = max(next.end, span.end)
		}
	}
	out = append(out, next)
	after := int64(0)
	for _, span := range out {
		after += span.end - span.start
	}
	return out, after - before
}
func (s *blockSpool) WriteAt(p []byte, offset int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if offset < 0 {
		return 0, errors.New("negative media offset")
	}
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	if s.err != nil {
		return 0, s.err
	}
	written := 0
	for len(p) > 0 {
		index, local := offset/SpoolBlockSize, offset%SpoolBlockSize
		count := int(min(int64(len(p)), SpoolBlockSize-local))
		f, err := os.OpenFile(s.path(index), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			s.fail(err)
			s.notify()
			return written, err
		}
		n, err := f.WriteAt(p[:count], local)
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if n > 0 {
			s.blocks[index], _ = addSpan(s.blocks[index], byteSpan{local, local + int64(n)})
			var added int64
			s.seen, added = addSpan(s.seen, byteSpan{offset, offset + int64(n)})
			s.downloaded += added
			s.received += int64(n)
			written += n
			offset += int64(n)
			p = p[n:]
		}
		if err == nil && n != count {
			err = io.ErrShortWrite
		}
		if err != nil {
			s.fail(err)
			s.notify()
			return written, err
		}
	}
	s.notify()
	return written, nil
}
func (s *blockSpool) readAt(p []byte, offset int64) (int, error, <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.ErrClosedPipe, nil
	}
	if s.err != nil {
		return 0, s.err, nil
	}
	index, local := offset/SpoolBlockSize, offset%SpoolBlockSize
	for _, span := range s.blocks[index] {
		if span.start <= local && local < span.end {
			f, err := os.Open(s.path(index))
			if err != nil {
				return 0, err, nil
			}
			n, err := f.ReadAt(p[:min(int64(len(p)), span.end-local)], local)
			f.Close()
			return n, err, nil
		}
	}
	return 0, nil, s.wake
}
func (s *blockSpool) discard(index int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path(index)); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(s.blocks, index)
	return nil
}
func (s *blockSpool) end(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ended = true
	if s.err == nil {
		s.fail(err)
	}
	s.notify()
}
func (s *blockSpool) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	clear(s.blocks)
	s.seen = nil
	s.notify()
	return s.owner.Remove(s.dir)
}
func (s *blockSpool) progress() (int64, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.downloaded, s.received
}

// DiskInput consumes a sequential producer without retaining its payload in RAM.
// Writes acknowledge durable-to-the-OS chunks, allowing both SABR tracks to drain
// even while FFmpeg is queued or currently reading only the other track.
type DiskInput struct {
	ctx             context.Context
	spool           *blockSpool
	writeMu, readMu sync.Mutex
	written, read   int64
}

func NewDiskInput(ctx context.Context, root string, owners ...*tempfiles.Scope) (*DiskInput, error) {
	spool, err := newBlockSpool(root, owners...)
	if err != nil {
		return nil, err
	}
	return &DiskInput{ctx: ctx, spool: spool}, nil
}
func (s *DiskInput) Write(p []byte) (int, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.ctx.Err(); err != nil {
		return 0, err
	}
	s.spool.mu.Lock()
	ended := s.spool.ended
	s.spool.mu.Unlock()
	if ended {
		return 0, io.ErrClosedPipe
	}
	n, err := s.spool.WriteAt(p, s.written)
	s.written += int64(n)
	return n, err
}
func (s *DiskInput) End(err error) { s.writeMu.Lock(); defer s.writeMu.Unlock(); s.spool.end(err) }
func (s *DiskInput) Read(p []byte) (int, error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if err := s.ctx.Err(); err != nil {
			return 0, err
		}
		n, err, wake := s.spool.readAt(p, s.read)
		if n > 0 {
			previous := s.read / SpoolBlockSize
			s.read += int64(n)
			if s.read/SpoolBlockSize > previous {
				if e := s.spool.discard(previous); err == nil {
					err = e
				}
			}
			return n, err
		}
		if err != nil {
			return 0, err
		}
		s.spool.mu.Lock()
		ended := s.spool.ended
		s.spool.mu.Unlock()
		// End can race the missing-data check; recheck after observing it.
		if ended {
			n, err, _ = s.spool.readAt(p, s.read)
			if n > 0 {
				// Use the regular consumption path, including block eviction.
				continue
			}
			if err != nil {
				return 0, err
			}
			if err := s.spool.discard(s.read / SpoolBlockSize); err != nil {
				return 0, err
			}
			return 0, io.EOF
		}
		select {
		case <-wake:
		case <-s.ctx.Done():
			return 0, s.ctx.Err()
		}
	}
}
func (s *DiskInput) Close() error                 { return s.spool.Close() }
func (*DiskInput) Seek(int64, int) (int64, error) { return 0, ErrSeek }
func (*DiskInput) Size() int64                    { return -1 }
func (*DiskInput) Seekable() bool                 { return false }
func (s *DiskInput) Progress() (int64, int64)     { return s.spool.progress() }

// fail is called with mu held. The first error wakes a queued FFmpeg job.
func (s *blockSpool) fail(err error) {
	if err == nil || s.err != nil {
		return
	}
	s.err = err
	select {
	case s.failures <- err:
	default:
	}
}
func (s *DiskInput) Failures() <-chan error { return s.spool.failures }
