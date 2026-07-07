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
)

const urlPathPrefix = "/__gopeed_blob/"

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
	Token       string
	URL         string
	ContentType string

	mu           sync.Mutex
	size         int64
	rangeEnabled bool
	revoked      bool
	session      SessionRef
	open         OpenFunc
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
		Range:       true,
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
	baseURL, err := r.ensureServer()
	if err != nil {
		return "", err
	}
	id, err := randomID(18)
	if err != nil {
		return "", err
	}
	token, err := randomID(24)
	if err != nil {
		return "", err
	}
	if opts.Session != nil {
		opts.Session.Retain()
	}
	srcURL := fmt.Sprintf("%s%s?token=%s", baseURL, id, url.QueryEscape(token))
	src := &Source{
		ID:           id,
		Token:        token,
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
	r.serverMu.Unlock()
	if server != nil {
		_ = server.Close()
	}
	if listener != nil {
		_ = listener.Close()
	}
	r.mu.Lock()
	sources := make([]*Source, 0, len(r.sources))
	for _, src := range r.sources {
		sources = append(sources, src)
	}
	r.sources = make(map[string]*Source)
	r.mu.Unlock()
	for _, src := range sources {
		src.close()
	}
	return nil
}

func (r *Registry) ensureServer() (string, error) {
	r.serverMu.Lock()
	defer r.serverMu.Unlock()
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
	id, token := parseRequest(req)
	if id == "" || token == "" {
		http.NotFound(w, req)
		return
	}
	src, err := r.getByID(id)
	if err != nil || src.Token != token {
		http.NotFound(w, req)
		return
	}
	meta, open := src.snapshot()
	if open == nil {
		http.NotFound(w, req)
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
		http.NotFound(w, req)
		return
	}
	defer reader.Close()

	writeHeaders(w, meta, start, end, ranged)
	if ranged {
		w.WriteHeader(http.StatusPartialContent)
	}
	if end >= start {
		_, _ = copyWithFlush(w, reader, end-start+1)
		return
	}
	_, _ = copyWithFlush(w, reader, -1)
}

func (s *Source) snapshot() (Metadata, OpenFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		return Metadata{}, nil
	}
	return Metadata{
		ContentType: s.ContentType,
		Size:        s.size,
		Range:       s.rangeEnabled,
	}, s.open
}

func (s *Source) close() {
	s.mu.Lock()
	alreadyRevoked := s.revoked
	s.revoked = true
	s.open = nil
	session := s.session
	s.session = nil
	s.mu.Unlock()
	if !alreadyRevoked && session != nil {
		session.Release()
	}
}

func (r *Registry) removeSource(src *Source) {
	r.mu.Lock()
	if current := r.sources[src.ID]; current == src {
		delete(r.sources, src.ID)
	}
	r.mu.Unlock()
	src.close()
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

func copyWithFlush(w http.ResponseWriter, reader io.Reader, limit int64) (int64, error) {
	if limit >= 0 {
		reader = io.LimitReader(reader, limit)
	}
	buf := make([]byte, 32*1024)
	var written int64
	flusher, canFlush := w.(http.Flusher)
	for {
		nr, er := reader.Read(buf)
		if nr > 0 {
			nw, ew := w.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				if canFlush {
					flusher.Flush()
				}
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				return written, nil
			}
			return written, er
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
	id, token, ok := r.parseURL(raw)
	if !ok {
		return nil, ErrInvalidURL
	}
	src, err := r.getByID(id)
	if err != nil {
		return nil, err
	}
	if src.Token != token {
		return nil, ErrInvalidURL
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

func (r *Registry) parseURL(raw string) (id string, token string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || !strings.HasPrefix(u.Path, urlPathPrefix) {
		return "", "", false
	}
	r.serverMu.Lock()
	baseURL := r.baseURL
	r.serverMu.Unlock()
	if baseURL == "" || !strings.HasPrefix(raw, baseURL) {
		return "", "", false
	}
	id = strings.TrimPrefix(u.Path, urlPathPrefix)
	if id == "" || path.Base(id) != id {
		return "", "", false
	}
	return id, u.Query().Get("token"), true
}

func parseRequest(req *http.Request) (id string, token string) {
	id = strings.TrimPrefix(req.URL.Path, urlPathPrefix)
	if id == "" || id == req.URL.Path || path.Base(id) != id {
		return "", ""
	}
	return id, req.URL.Query().Get("token")
}

func randomID(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
