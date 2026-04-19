package gblob

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const Scheme = "gblob"

var (
	ErrInvalidURL     = errors.New("invalid gblob url")
	ErrSourceNotFound = errors.New("gblob source not found")
	ErrSourceRevoked  = errors.New("gblob source revoked")
	ErrSourceClosed   = errors.New("gblob source closed")
	ErrSourceAborted  = errors.New("gblob source aborted")
)

type SourceType string

const (
	SourceTypeBlob           SourceType = "blob"
	SourceTypeWritableStream SourceType = "writable_stream"
)

type SourceState string

const (
	SourceStateOpen    SourceState = "open"
	SourceStateClosed  SourceState = "closed"
	SourceStateAborted SourceState = "aborted"
	SourceStateFailed  SourceState = "failed"
)

type SessionRef interface {
	Retain()
	Release()
}

type Snapshot struct {
	Written int64
	State   SourceState
	Err     error
	WaitCh  <-chan struct{}
}

type Source struct {
	ID          string
	URL         string
	Path        string
	Type        SourceType
	ContentType string

	mu              sync.Mutex
	file            *os.File
	written         int64
	state           SourceState
	err             error
	revoked         bool
	pins            int
	waitCh          chan struct{}
	session         SessionRef
	sessionReleased bool
}

func (s *Source) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{
		Written: s.written,
		State:   s.state,
		Err:     s.err,
		WaitCh:  s.waitCh,
	}
}

func (s *Source) State() SourceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Source) IsRevoked() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revoked
}

func (s *Source) pin() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pins++
}

func (s *Source) unpin() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pins > 0 {
		s.pins--
	}
	return s.pins
}

func (s *Source) shouldCleanupLocked() bool {
	return s.revoked && s.pins == 0 && s.state != SourceStateOpen
}

func (s *Source) notifyLocked() {
	close(s.waitCh)
	s.waitCh = make(chan struct{})
}

func (s *Source) releaseSessionLocked() {
	if s.session != nil && !s.sessionReleased {
		s.session.Release()
		s.sessionReleased = true
	}
}

func (s *Source) write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		return ErrSourceRevoked
	}
	if s.state != SourceStateOpen {
		if s.state == SourceStateAborted || s.state == SourceStateFailed {
			if s.err != nil {
				return s.err
			}
			return ErrSourceAborted
		}
		return ErrSourceClosed
	}
	if len(data) == 0 {
		return nil
	}
	n, err := s.file.Write(data)
	if err != nil {
		s.state = SourceStateFailed
		s.err = err
		s.file.Close()
		s.file = nil
		s.releaseSessionLocked()
		s.notifyLocked()
		return err
	}
	s.written += int64(n)
	s.notifyLocked()
	return nil
}

func (s *Source) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != SourceStateOpen {
		return nil
	}
	s.state = SourceStateClosed
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			s.err = err
		}
		s.file = nil
	}
	s.releaseSessionLocked()
	s.notifyLocked()
	return nil
}

func (s *Source) abort(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != SourceStateOpen {
		return nil
	}
	s.state = SourceStateAborted
	if err != nil {
		s.err = err
	}
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
	s.releaseSessionLocked()
	s.notifyLocked()
	return nil
}

func (s *Source) revoke() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked = true
	return s.shouldCleanupLocked()
}

func (s *Source) waitForReadable(ctx context.Context, offset int64) error {
	for {
		snapshot := s.Snapshot()
		if offset < snapshot.Written {
			return nil
		}
		switch snapshot.State {
		case SourceStateClosed:
			return nil
		case SourceStateAborted, SourceStateFailed:
			if snapshot.Err != nil {
				return snapshot.Err
			}
			return ErrSourceAborted
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-snapshot.WaitCh:
		}
	}
}

type Registry struct {
	dir string

	mu      sync.RWMutex
	sources map[string]*Source
}

func NewRegistry(baseDir string) *Registry {
	dir := baseDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "gopeed-gblob")
	}
	return &Registry{
		dir:     dir,
		sources: make(map[string]*Source),
	}
}

func BuildURL(id string) string {
	return Scheme + ":" + id
}

func ParseURL(raw string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(raw), Scheme+":") {
		return "", ErrInvalidURL
	}
	id := raw[len(Scheme)+1:]
	if idx := strings.IndexByte(id, '/'); idx >= 0 {
		id = id[:idx]
	}
	if id == "" {
		return "", ErrInvalidURL
	}
	return id, nil
}

func (r *Registry) createSource(sourceType SourceType, contentType string, session SessionRef) (*Source, error) {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return nil, err
	}
	id, err := gonanoid.New()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(r.dir, id)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	if session != nil {
		session.Retain()
	}
	src := &Source{
		ID:          id,
		URL:         BuildURL(id),
		Path:        filePath,
		Type:        sourceType,
		ContentType: contentType,
		file:        file,
		state:       SourceStateOpen,
		waitCh:      make(chan struct{}),
		session:     session,
	}
	r.mu.Lock()
	r.sources[id] = src
	r.mu.Unlock()
	return src, nil
}

func (r *Registry) CreateBlob(data []byte, contentType string) (string, error) {
	src, err := r.createSource(SourceTypeBlob, contentType, nil)
	if err != nil {
		return "", err
	}
	if err := src.write(data); err != nil {
		_ = os.Remove(src.Path)
		return "", err
	}
	if err := src.close(); err != nil {
		_ = os.Remove(src.Path)
		return "", err
	}
	return src.URL, nil
}

func (r *Registry) CreateWritableStream(session SessionRef) (string, error) {
	src, err := r.createSource(SourceTypeWritableStream, "", session)
	if err != nil {
		return "", err
	}
	return src.URL, nil
}

func (r *Registry) get(raw string) (*Source, error) {
	id, err := ParseURL(raw)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.sources[id]
	if src == nil {
		return nil, ErrSourceNotFound
	}
	return src, nil
}

func (r *Registry) Get(raw string) (*Source, error) {
	return r.get(raw)
}

func (r *Registry) Write(raw string, data []byte) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	return src.write(data)
}

func (r *Registry) CloseSource(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	if err := src.close(); err != nil {
		return err
	}
	r.cleanupIfNeeded(src)
	return nil
}

func (r *Registry) AbortSource(raw string, err error) error {
	src, gerr := r.get(raw)
	if gerr != nil {
		return gerr
	}
	if err := src.abort(err); err != nil {
		return err
	}
	r.cleanupIfNeeded(src)
	return nil
}

func (r *Registry) Revoke(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	if src.revoke() {
		r.cleanupIfNeeded(src)
	}
	return nil
}

func (r *Registry) Pin(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	if src.IsRevoked() {
		return ErrSourceRevoked
	}
	src.pin()
	return nil
}

func (r *Registry) Unpin(raw string) {
	src, err := r.get(raw)
	if err != nil {
		return
	}
	src.unpin()
	r.cleanupIfNeeded(src)
}

func (r *Registry) WaitForReadable(ctx context.Context, raw string, offset int64) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	return src.waitForReadable(ctx, offset)
}

func (r *Registry) cleanupIfNeeded(src *Source) {
	src.mu.Lock()
	shouldCleanup := src.shouldCleanupLocked()
	file := src.file
	src.mu.Unlock()
	if !shouldCleanup {
		return
	}
	r.mu.Lock()
	delete(r.sources, src.ID)
	r.mu.Unlock()
	if file != nil {
		_ = file.Close()
	}
	_ = os.Remove(src.Path)
}

func (r *Registry) Close() error {
	r.mu.Lock()
	sources := r.sources
	r.sources = make(map[string]*Source)
	r.mu.Unlock()
	var lastErr error
	for _, src := range sources {
		src.mu.Lock()
		if src.file != nil {
			if err := src.file.Close(); err != nil {
				lastErr = err
			}
			src.file = nil
		}
		src.releaseSessionLocked()
		src.notifyLocked()
		src.mu.Unlock()
		if err := os.Remove(src.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			lastErr = err
		}
	}
	return lastErr
}
