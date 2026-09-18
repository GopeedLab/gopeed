// Package tempfiles tracks temporary paths owned by one downloader, without
// requiring a private parent directory or scanning the shared system temp root.
package tempfiles

import (
	"errors"
	"io"
	"os"
	"sync"
)

type Scope struct {
	mu     sync.Mutex
	paths  map[string]struct{}
	closed bool
}

// Track adopts a newly created unique path. Late creations are removed when
// shutdown has already begun. A nil scope leaves cleanup to the resource owner.
func (s *Scope) Track(path string) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.Join(io.ErrClosedPipe, os.RemoveAll(path))
	}
	if s.paths == nil {
		s.paths = make(map[string]struct{})
	}
	s.paths[path] = struct{}{}
	return nil
}
func (s *Scope) Remove(path string) error {
	if s == nil {
		return os.RemoveAll(path)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	delete(s.paths, path)
	return nil
}
func (s *Scope) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	var result error
	for path := range s.paths {
		if err := os.RemoveAll(path); err != nil {
			result = errors.Join(result, err)
		} else {
			delete(s.paths, path)
		}
	}
	return result
}
