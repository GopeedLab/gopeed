package blob

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const urlPathPrefix = "/__blob/"

const rangeSourceFailureLimit = 2

// unclaimedSourceTTL bounds how long a session-backed source may keep its
// engine alive without ever being claimed by a download task.
var unclaimedSourceTTL = 10 * time.Minute

var (
	ErrInvalidURL      = errors.New("invalid blob url")
	ErrInvalidOptions  = errors.New("invalid blob options")
	ErrSourceNotFound  = errors.New("blob source not found")
	ErrSourceRevoked   = errors.New("blob source revoked")
	ErrSourceClosed    = errors.New("blob source closed")
	ErrRangeNotAllowed = errors.New("blob range not allowed")
)

type SessionRef interface {
	Retain()
	Release()
}

type OpenRequest struct {
	Offset int64
	End    int64
}

type OpenFunc func(ctx context.Context, req OpenRequest) (io.ReadCloser, error)

type CreateOptions struct {
	ContentType string
	Size        int64
	Range       bool
	Session     SessionRef
}

type Metadata struct {
	ContentType string
	Size        int64
	Range       bool
}

type Source struct {
	ID          string
	URL         string
	ContentType string

	mu            sync.Mutex
	size          int64
	rangeEnabled  bool
	revoked       bool
	taskRefs      int
	readErr       error
	rangeFailures int
	session       SessionRef
	open          OpenFunc
	unclaimed     *time.Timer
}

type Registry struct {
	dir string

	mu      sync.RWMutex
	sources map[string]*Source

	serverMu sync.Mutex
	listener net.Listener
	server   *http.Server
	baseURL  string
}

func NewRegistry(dir string) *Registry {
	return &Registry{
		dir:     dir,
		sources: make(map[string]*Source),
	}
}

func (r *Registry) Dir() string {
	if r == nil {
		return ""
	}
	return r.dir
}

func (r *Registry) IsURL(raw string) bool {
	if r == nil {
		return false
	}
	src, err := r.get(raw)
	return err == nil && src != nil
}

func (r *Registry) CreateBlob(data []byte, contentType string) (string, error) {
	buf := append([]byte(nil), data...)
	return r.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		if req.Offset < 0 || req.Offset > int64(len(buf)) {
			return nil, ErrSourceNotFound
		}
		end := int64(len(buf))
		if req.End >= req.Offset && req.End+1 < end {
			end = req.End + 1
		}
		return io.NopCloser(bytes.NewReader(buf[req.Offset:end])), nil
	}, &CreateOptions{
		ContentType: contentType,
		Size:        int64(len(buf)),
		Range:       len(buf) > 0,
	})
}

func (r *Registry) CreateOpener(open OpenFunc, opts *CreateOptions) (string, error) {
	if r == nil {
		return "", ErrSourceNotFound
	}
	if open == nil {
		return "", ErrInvalidOptions
	}
	if opts == nil {
		opts = &CreateOptions{}
	}
	if opts.Size < 0 {
		opts.Size = 0
	}
	if opts.Range && opts.Size <= 0 {
		return "", fmt.Errorf("%w: range requires positive size", ErrInvalidOptions)
	}
	// Keep server generation selection and source insertion atomic with Close.
	r.serverMu.Lock()
	baseURL, err := r.ensureServerLocked()
	if err != nil {
		r.serverMu.Unlock()
		return "", err
	}
	id, err := randomID(18)
	if err != nil {
		r.serverMu.Unlock()
		return "", err
	}
	if opts.Session != nil {
		opts.Session.Retain()
	}
	srcURL := fmt.Sprintf("%s%s", baseURL, id)
	src := &Source{
		ID:           id,
		URL:          srcURL,
		ContentType:  opts.ContentType,
		size:         opts.Size,
		rangeEnabled: opts.Range,
		session:      opts.Session,
		open:         open,
	}
	r.mu.Lock()
	r.sources[id] = src
	r.mu.Unlock()
	r.serverMu.Unlock()
	if opts.Session != nil && unclaimedSourceTTL > 0 {
		src.mu.Lock()
		if !src.revoked {
			src.unclaimed = time.AfterFunc(unclaimedSourceTTL, func() {
				r.expireUnclaimed(src)
			})
		}
		src.mu.Unlock()
	}
	return src.URL, nil
}

func (r *Registry) Metadata(raw string) (Metadata, error) {
	src, err := r.get(raw)
	if err != nil {
		return Metadata{}, err
	}
	src.mu.Lock()
	defer src.mu.Unlock()
	return Metadata{
		ContentType: src.ContentType,
		Size:        src.size,
		Range:       src.rangeEnabled,
	}, nil
}

// Acquire retains a source for a task. A source remains available until every
// task that acquired it calls Release, unless it is explicitly revoked.
func (r *Registry) Acquire(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	return src.acquireTask()
}

// Release releases a task's reference to a source. Releasing the final task
// reference revokes the source. Sources that have never been acquired remain
// registered until Revoke or Registry.Close is called; session-backed sources
// are also bounded by the unclaimed-source TTL.
func (r *Registry) Release(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	session, last, err := src.releaseTask()
	if err != nil {
		return err
	}
	if !last {
		return nil
	}
	r.deleteSource(src)
	if session != nil {
		session.Release()
	}
	return nil
}

// SourceError returns the first error encountered while opening or reading a
// source. Invalid and missing sources have no source error. HTTP response write
// errors are deliberately not recorded here.
func (r *Registry) SourceError(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return nil
	}
	return src.sourceError()
}

func (r *Registry) Revoke(raw string) error {
	src, err := r.get(raw)
	if err != nil {
		return err
	}
	r.removeSource(src)
	return nil
}

func (r *Registry) Close() error {
	if r == nil {
		return nil
	}
	r.serverMu.Lock()
	server := r.server
	listener := r.listener
	r.server = nil
	r.listener = nil
	r.baseURL = ""
	r.mu.Lock()
	sources := make([]*Source, 0, len(r.sources))
	for _, src := range r.sources {
		sources = append(sources, src)
	}
	r.sources = make(map[string]*Source)
	r.mu.Unlock()
	r.serverMu.Unlock()
	if server != nil {
		_ = server.Close()
	}
	if listener != nil {
		_ = listener.Close()
	}
	for _, src := range sources {
		src.close()
	}
	return nil
}

// ensureServerLocked returns the current server URL or starts a new server.
// The caller must hold serverMu.
func (r *Registry) ensureServerLocked() (string, error) {
	if r.baseURL != "" {
		return r.baseURL, nil
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	r.listener = listener
	r.baseURL = "http://" + listener.Addr().String() + urlPathPrefix
	server := &http.Server{Handler: r}
	r.server = server
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = r.Close()
		}
	}()
	return r.baseURL, nil
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := parseRequest(req)
	if id == "" {
		http.NotFound(w, req)
		return
	}
	src, err := r.getByID(id)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	meta, open, session := src.acquireOpen()
	if open == nil {
		http.NotFound(w, req)
		return
	}
	if session != nil {
		defer session.Release()
	}
	if src.sourceError() != nil {
		http.Error(w, "blob source unavailable", http.StatusGone)
		return
	}
	start, end, ranged, err := parseRange(req.Header.Get("Range"), meta.Size, meta.Range)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
		return
	}
	if !meta.Range {
		start, end, ranged = 0, -1, false
	}
	reader, err := open(req.Context(), OpenRequest{
		Offset: start,
		End:    end,
	})
	if err != nil {
		if req.Context().Err() == nil || !errors.Is(err, req.Context().Err()) {
			src.recordSourceError(err, meta.Range)
		}
		http.Error(w, "blob source unavailable", http.StatusGone)
		return
	}
	if reader == nil {
		err = fmt.Errorf("%w: opener returned a nil reader", ErrSourceClosed)
		src.recordSourceError(err, meta.Range)
		http.Error(w, "blob source unavailable", http.StatusGone)
		return
	}
	defer reader.Close()

	writeHeaders(w, meta, start, end, ranged)
	if ranged {
		w.WriteHeader(http.StatusPartialContent)
	}
	limit := int64(-1)
	if end >= start {
		limit = end - start + 1
	} else if meta.Size > 0 {
		limit = meta.Size
	}
	_, readErr, writeErr := copyWithFlush(w, reader, limit)
	if readErr != nil {
		if ctxErr := req.Context().Err(); ctxErr == nil || !errors.Is(readErr, ctxErr) {
			src.recordSourceError(readErr, meta.Range)
		}
	} else if writeErr == nil && req.Context().Err() == nil {
		src.recordSourceSuccess(meta.Range)
	}
}

func (s *Source) acquireOpen() (Metadata, OpenFunc, SessionRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		return Metadata{}, nil, nil
	}
	if s.session != nil {
		// The source's own retain guarantees that the session cannot become
		// closed while this additional active-reader retain is acquired.
		s.session.Retain()
	}
	return Metadata{
		ContentType: s.ContentType,
		Size:        s.size,
		Range:       s.rangeEnabled,
	}, s.open, s.session
}

func (s *Source) acquireTask() error {
	s.mu.Lock()
	if s.revoked {
		s.mu.Unlock()
		return ErrSourceRevoked
	}
	s.taskRefs++
	timer := s.unclaimed
	s.unclaimed = nil
	s.mu.Unlock()
	if timer != nil {
		timer.Stop()
	}
	return nil
}

func (s *Source) releaseTask() (session SessionRef, last bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		return nil, false, ErrSourceRevoked
	}
	if s.taskRefs == 0 {
		return nil, false, ErrSourceClosed
	}
	s.taskRefs--
	if s.taskRefs > 0 {
		return nil, false, nil
	}
	s.revoked = true
	s.open = nil
	session = s.session
	s.session = nil
	return session, true, nil
}

func (s *Source) sourceError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readErr
}

func (s *Source) recordSourceError(err error, ranged bool) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.readErr == nil && ranged {
		s.rangeFailures++
		if s.rangeFailures < rangeSourceFailureLimit {
			s.mu.Unlock()
			return
		}
	}
	if s.readErr == nil {
		s.readErr = err
	}
	s.mu.Unlock()
}

func (s *Source) recordSourceSuccess(ranged bool) {
	if !ranged {
		return
	}
	s.mu.Lock()
	if s.readErr == nil {
		s.rangeFailures = 0
	}
	s.mu.Unlock()
}

func (s *Source) close() {
	s.mu.Lock()
	alreadyRevoked := s.revoked
	s.revoked = true
	s.open = nil
	session := s.session
	s.session = nil
	timer := s.unclaimed
	s.unclaimed = nil
	s.mu.Unlock()
	if timer != nil {
		timer.Stop()
	}
	if !alreadyRevoked && session != nil {
		session.Release()
	}
}

func (s *Source) expireUnclaimed() (SessionRef, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unclaimed = nil
	if s.revoked || s.taskRefs > 0 {
		return nil, false
	}
	s.revoked = true
	s.open = nil
	session := s.session
	s.session = nil
	return session, true
}

func (r *Registry) expireUnclaimed(src *Source) {
	session, expired := src.expireUnclaimed()
	if !expired {
		return
	}
	r.deleteSource(src)
	if session != nil {
		session.Release()
	}
}

func (r *Registry) removeSource(src *Source) {
	src.close()
	r.deleteSource(src)
}

func (r *Registry) deleteSource(src *Source) {
	r.mu.Lock()
	if current := r.sources[src.ID]; current == src {
		delete(r.sources, src.ID)
	}
	r.mu.Unlock()
}

func writeHeaders(w http.ResponseWriter, meta Metadata, start, end int64, ranged bool) {
	if meta.ContentType != "" {
		w.Header().Set("Content-Type", meta.ContentType)
	}
	if meta.Range {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	if meta.Size > 0 {
		if ranged {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, meta.Size))
			w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
			return
		}
		w.Header().Set("Content-Length", strconv.FormatInt(meta.Size, 10))
	}
}

func copyWithFlush(w http.ResponseWriter, reader io.Reader, limit int64) (written int64, readErr error, writeErr error) {
	if limit >= 0 {
		reader = io.LimitReader(reader, limit)
	}
	buf := make([]byte, 32*1024)
	flusher, canFlush := w.(http.Flusher)
	for {
		nr, er := reader.Read(buf)
		if er != nil && !errors.Is(er, io.EOF) {
			readErr = er
		}
		if nr > 0 {
			nw, ew := w.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				if canFlush {
					flusher.Flush()
				}
			}
			if ew != nil {
				return written, readErr, ew
			}
			if nr != nw {
				return written, readErr, io.ErrShortWrite
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				if limit >= 0 && written < limit {
					return written, io.ErrUnexpectedEOF, nil
				}
				return written, nil, nil
			}
			return written, readErr, nil
		}
	}
}

func parseRange(header string, size int64, rangeEnabled bool) (start int64, end int64, ranged bool, err error) {
	if header == "" || !rangeEnabled {
		return 0, -1, false, nil
	}
	if size <= 0 {
		return 0, 0, false, ErrRangeNotAllowed
	}
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, false, fmt.Errorf("unsupported range")
	}
	parts := strings.SplitN(strings.TrimPrefix(header, "bytes="), "-", 2)
	if len(parts) != 2 || parts[0] == "" {
		return 0, 0, false, fmt.Errorf("unsupported range")
	}
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false, fmt.Errorf("invalid range")
	}
	if start >= size {
		return 0, 0, false, fmt.Errorf("range out of bounds")
	}
	end = size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, false, fmt.Errorf("invalid range")
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end, true, nil
}

func (r *Registry) get(raw string) (*Source, error) {
	id, ok := r.parseURL(raw)
	if !ok {
		return nil, ErrInvalidURL
	}
	src, err := r.getByID(id)
	if err != nil {
		return nil, err
	}
	src.mu.Lock()
	revoked := src.revoked
	src.mu.Unlock()
	if revoked {
		return nil, ErrSourceRevoked
	}
	return src, nil
}

func (r *Registry) getByID(id string) (*Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.sources[id]
	if src == nil {
		return nil, ErrSourceNotFound
	}
	return src, nil
}

func (r *Registry) parseURL(raw string) (id string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || !strings.HasPrefix(u.Path, urlPathPrefix) {
		return "", false
	}
	r.serverMu.Lock()
	baseURL := r.baseURL
	r.serverMu.Unlock()
	if baseURL == "" || !strings.HasPrefix(raw, baseURL) {
		return "", false
	}
	id = strings.TrimPrefix(u.Path, urlPathPrefix)
	if id == "" || path.Base(id) != id {
		return "", false
	}
	return id, true
}

func parseRequest(req *http.Request) string {
	id := strings.TrimPrefix(req.URL.Path, urlPathPrefix)
	if id == "" || id == req.URL.Path || path.Base(id) != id {
		return ""
	}
	return id
}

func randomID(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
