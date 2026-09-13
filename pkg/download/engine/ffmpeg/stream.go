package ffmpeg

import (
	"context"
	"errors"
	"io"
	"sync"
)

// A bounded memory ring lets both JS inputs drain independently, including when
// they share a SABR producer. It does not make a sequential input seekable.
// A full ring waits for consumption or cancellation. Producers must propagate
// backpressure and allow each track to make progress independently.
const StreamBufferLimit = 32 * 1024 * 1024

type StreamInput struct {
	ctx           context.Context
	mu            sync.Mutex
	buffer        []byte
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
	for int64(len(p)) > StreamBufferLimit-(s.written-s.read) && !s.closed && !s.ended {
		s.mu.Unlock()
		var err error
		select {
		case <-s.space:
		case <-s.ctx.Done():
			err = s.ctx.Err()
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
	// Grow only as needed. Rebase unread bytes when the ring grows.
	need := int(s.written-s.read) + len(p)
	if need > len(s.buffer) {
		capacity := min(StreamBufferLimit, max(256*1024, max(need, len(s.buffer)*2)))
		buffer := make([]byte, capacity)
		unread := int(s.written - s.read)
		if unread > 0 {
			off := int(s.read % int64(len(s.buffer)))
			first := min(unread, len(s.buffer)-off)
			copy(buffer, s.buffer[off:off+first])
			copy(buffer[first:], s.buffer[:unread-first])
		}
		s.buffer = buffer
		s.read, s.written = 0, int64(unread)
	}
	off := int(s.written % int64(len(s.buffer)))
	first := min(len(p), len(s.buffer)-off)
	copy(s.buffer[off:], p[:first])
	copy(s.buffer, p[first:])
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
			off := s.read % int64(len(s.buffer))
			n := min(int64(len(p)), s.written-s.read, int64(len(s.buffer))-off)
			count := copy(p, s.buffer[off:off+n])
			s.read += int64(count)
			s.mu.Unlock()
			select {
			case s.space <- struct{}{}:
			default:
			}
			return count, nil
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
	s.buffer = nil
	return nil
}
func (*StreamInput) Seek(int64, int) (int64, error) { return 0, ErrSeek }
func (*StreamInput) Size() int64                    { return -1 }
func (*StreamInput) Seekable() bool                 { return false }
