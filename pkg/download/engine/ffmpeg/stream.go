package ffmpeg

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

// A bounded disk ring lets both JS inputs drain independently, including when
// they share a SABR producer. It does not make a sequential input seekable.
// A full ring applies backpressure. A prolonged stall fails instead of
// deadlocking a shared upstream or growing forever.
const StreamBufferLimit = 64 * 1024 * 1024

type StreamInput struct {
	ctx           context.Context
	mu            sync.Mutex
	file          *os.File
	read, written int64
	ended, closed bool
	err           error
	wake          chan struct{}
	space         chan struct{}
}

func NewStreamInput(ctx context.Context) *StreamInput {
	return &StreamInput{ctx: ctx, wake: make(chan struct{}, 1), space: make(chan struct{}, 1)}
}

func (s *StreamInput) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(p) > StreamBufferLimit {
		return 0, errors.New("ffmpeg: input chunk exceeds buffer capacity")
	}
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for int64(len(p)) > StreamBufferLimit-(s.written-s.read) && !s.closed && !s.ended {
		if timer == nil {
			timer = time.NewTimer(30 * time.Second)
		}
		s.mu.Unlock()
		var err error
		select {
		case <-s.space:
		case <-s.ctx.Done():
			err = s.ctx.Err()
		case <-timer.C:
			err = errors.New("ffmpeg: input buffer stalled for 30 seconds; upstream tracks may be too far apart")
		}
		s.mu.Lock()
		if err != nil {
			return 0, err
		}
	}
	if s.closed || s.ended {
		return 0, io.ErrClosedPipe
	}
	if err := s.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	if s.file == nil {
		f, err := os.CreateTemp("", "gopeed-ffmpeg-*")
		if err != nil {
			return 0, err
		}
		s.file = f
	}
	off := s.written % StreamBufferLimit
	first := min(len(p), StreamBufferLimit-int(off))
	if _, err := s.file.WriteAt(p[:first], off); err != nil {
		return 0, err
	}
	if first < len(p) {
		if _, err := s.file.WriteAt(p[first:], 0); err != nil {
			return 0, err
		}
	}
	s.written += int64(len(p))
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return len(p), nil
}

func (s *StreamInput) End(err error) {
	s.mu.Lock()
	s.ended = true
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
	select {
	case s.space <- struct{}{}:
	default:
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *StreamInput) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if err := s.ctx.Err(); err != nil {
			return 0, err
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return 0, io.ErrClosedPipe
		}
		if s.err != nil {
			err := s.err
			s.mu.Unlock()
			return 0, err
		}
		if s.read < s.written {
			off := s.read % StreamBufferLimit
			n := min(int64(len(p)), s.written-s.read, StreamBufferLimit-off)
			count, err := s.file.ReadAt(p[:n], off)
			s.read += int64(count)
			s.mu.Unlock()
			select {
			case s.space <- struct{}{}:
			default:
			}
			return count, err
		}
		ended := s.ended
		s.mu.Unlock()
		if ended {
			return 0, io.EOF
		}
		select {
		case <-s.wake:
		case <-s.ctx.Done():
			return 0, s.ctx.Err()
		}
	}
}

func (s *StreamInput) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	select {
	case s.space <- struct{}{}:
	default:
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	if s.file != nil {
		name := s.file.Name()
		s.file.Close()
		s.file = nil
		return os.Remove(name)
	}
	return nil
}
func (*StreamInput) Seek(int64, int) (int64, error) { return 0, ErrSeek }
func (*StreamInput) Size() int64                    { return -1 }
func (*StreamInput) Seekable() bool                 { return false }
