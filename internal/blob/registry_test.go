package blob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type terminalErrorReader struct {
	err error
}

func (r *terminalErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type contextErrorReader struct {
	ctx context.Context
}

func (r *contextErrorReader) Read([]byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (r *contextErrorReader) Close() error { return nil }

type failingResponseWriter struct {
	header http.Header
	err    error
}

func (w *failingResponseWriter) Header() http.Header {
	return w.header
}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func (w *failingResponseWriter) WriteHeader(int) {}

type countingSession struct {
	mu       sync.Mutex
	refs     int
	zero     chan struct{}
	zeroOnce sync.Once
}

func newCountingSession() *countingSession {
	return &countingSession{zero: make(chan struct{})}
}

func (s *countingSession) Retain() {
	s.mu.Lock()
	s.refs++
	s.mu.Unlock()
}

func (s *countingSession) Release() {
	s.mu.Lock()
	s.refs--
	refs := s.refs
	s.mu.Unlock()
	if refs == 0 {
		s.zeroOnce.Do(func() { close(s.zero) })
	}
}

func (s *countingSession) RefCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refs
}

func setUnclaimedSourceTTLForTest(t *testing.T, ttl time.Duration) {
	t.Helper()
	previous := unclaimedSourceTTL
	unclaimedSourceTTL = ttl
	t.Cleanup(func() {
		unclaimedSourceTTL = previous
	})
}

func TestRegistryBlobHTTPGetAndRange(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateBlob([]byte("hello world"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if !registry.IsURL(url) {
		t.Fatal("expected registry to recognize created url")
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected GET status: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Length"); got != "11" {
		t.Fatalf("unexpected content length: %q", got)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("unexpected accept ranges: %q", got)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("unexpected content type: %q", got)
	}
	if string(body) != "hello world" {
		t.Fatalf("unexpected GET body: %q", string(body))
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=6-10")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("unexpected range status: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Range"); got != "bytes 6-10/11" {
		t.Fatalf("unexpected content range: %q", got)
	}
	if string(body) != "world" {
		t.Fatalf("unexpected range body: %q", string(body))
	}
}

func TestRegistryEmptyBlob(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateBlob(nil, "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := registry.Metadata(url)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Size != 0 || meta.Range {
		t.Fatalf("unexpected empty blob metadata: %#v", meta)
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if len(body) != 0 {
		t.Fatalf("unexpected body: %q", body)
	}
	if got := resp.Header.Get("Content-Length"); got != "0" {
		t.Fatalf("unexpected content length: %q", got)
	}
}

func TestRegistryOpenerSizeWithoutRange(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	payload := []byte("hello opener")
	url, err := registry.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		if req.Offset != 0 || req.End != -1 {
			t.Fatalf("unexpected non-range request: %#v", req)
		}
		return io.NopCloser(bytes.NewReader(payload)), nil
	}, &CreateOptions{
		ContentType: "text/plain",
		Size:        int64(len(payload)),
		Range:       false,
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=6-11")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Length"); got != "12" {
		t.Fatalf("unexpected content length: %q", got)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "" {
		t.Fatalf("unexpected accept ranges: %q", got)
	}
	if string(body) != string(payload) {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestRegistryOpenerRangeConcurrentRequests(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	payload := []byte("abcdefghijklmnopqrstuvwxyz")
	var mu sync.Mutex
	var calls []OpenRequest
	url, err := registry.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		mu.Lock()
		calls = append(calls, req)
		mu.Unlock()

		start := int(req.Offset)
		end := len(payload)
		if req.End >= 0 {
			end = int(req.End) + 1
		}
		if start < 0 || start > end || end > len(payload) {
			return nil, errors.New("bad test range")
		}
		return io.NopCloser(bytes.NewReader(payload[start:end])), nil
	}, &CreateOptions{
		Size:  int64(len(payload)),
		Range: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ranges := []struct {
		header string
		want   string
	}{
		{header: "bytes=0-4", want: "abcde"},
		{header: "bytes=10-15", want: "klmnop"},
		{header: "bytes=20-25", want: "uvwxyz"},
	}
	var wg sync.WaitGroup
	for _, item := range ranges {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Set("Range", item.header)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				t.Error(err)
				return
			}
			if resp.StatusCode != http.StatusPartialContent {
				t.Errorf("unexpected status for %s: %d", item.header, resp.StatusCode)
			}
			if string(body) != item.want {
				t.Errorf("unexpected body for %s: %q", item.header, string(body))
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != len(ranges) {
		t.Fatalf("unexpected opener call count: %d", len(calls))
	}
	seen := map[OpenRequest]bool{}
	for _, call := range calls {
		seen[call] = true
	}
	for _, want := range []OpenRequest{{Offset: 0, End: 4}, {Offset: 10, End: 15}, {Offset: 20, End: 25}} {
		if !seen[want] {
			t.Fatalf("missing opener request %#v, got %#v", want, calls)
		}
	}
}

func TestRegistryRejectsRangeWithoutSize(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	_, err := registry.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("")), nil
	}, &CreateOptions{Range: true})
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("expected invalid options error, got %v", err)
	}
}

func TestRegistryTracksSourceErrors(t *testing.T) {
	t.Run("reader error", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		boom := errors.New("reader failed")
		url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
			return io.NopCloser(io.MultiReader(strings.NewReader("partial"), &terminalErrorReader{err: boom})), nil
		}, &CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			t.Fatalf("the HTTP transport should end normally, got %v", readErr)
		}
		if string(body) != "partial" {
			t.Fatalf("unexpected partial body: %q", body)
		}
		if got := registry.SourceError(url); !errors.Is(got, boom) {
			t.Fatalf("expected source error %v, got %v", boom, got)
		}
	})

	t.Run("short known-size reader", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("short")), nil
		}, &CreateOptions{Size: 10})
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if got := registry.SourceError(url); !errors.Is(got, io.ErrUnexpectedEOF) {
			t.Fatalf("expected unexpected EOF, got %v", got)
		}
	})

	t.Run("range reader error becomes terminal", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		boom := errors.New("range reader failed")
		url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
			return io.NopCloser(io.MultiReader(strings.NewReader("partial"), &terminalErrorReader{err: boom})), nil
		}, &CreateOptions{Size: 10, Range: true})
		if err != nil {
			t.Fatal(err)
		}

		for attempt := 0; attempt < rangeSourceFailureLimit; attempt++ {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Range", "bytes=0-9")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		if got := registry.SourceError(url); !errors.Is(got, boom) {
			t.Fatalf("expected range source error %v, got %v", boom, got)
		}

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusGone {
			t.Fatalf("expected terminal source status %d, got %d", http.StatusGone, resp.StatusCode)
		}
	})

	t.Run("short range reader", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("short")), nil
		}, &CreateOptions{Size: 10, Range: true})
		if err != nil {
			t.Fatal(err)
		}
		for attempt := 0; attempt < rangeSourceFailureLimit; attempt++ {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Range", "bytes=0-9")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		if got := registry.SourceError(url); !errors.Is(got, io.ErrUnexpectedEOF) {
			t.Fatalf("expected range unexpected EOF, got %v", got)
		}
	})

	t.Run("successful range retry clears transient failure", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		attempt := 0
		boom := errors.New("transient range failure")
		url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
			attempt++
			if attempt == 1 {
				return io.NopCloser(&terminalErrorReader{err: boom}), nil
			}
			return io.NopCloser(strings.NewReader("0123456789")), nil
		}, &CreateOptions{Size: 10, Range: true})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Range", "bytes=0-9")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		if got := registry.SourceError(url); got != nil {
			t.Fatalf("successful retry left transient source error: %v", got)
		}
	})

	t.Run("request cancellation is not a source error", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		opened := make(chan struct{})
		url, err := registry.CreateOpener(func(ctx context.Context, _ OpenRequest) (io.ReadCloser, error) {
			close(opened)
			return &contextErrorReader{ctx: ctx}, nil
		}, &CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			resp, _ := http.DefaultClient.Do(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
			close(done)
		}()
		<-opened
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("cancelled request did not finish")
		}
		if got := registry.SourceError(url); got != nil {
			t.Fatalf("request cancellation was recorded as source error: %v", got)
		}
	})

	t.Run("nil reader", func(t *testing.T) {
		registry := NewRegistry(t.TempDir())
		defer registry.Close()

		url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
			return nil, nil
		}, &CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusGone {
			t.Fatalf("expected nil reader status %d, got %d", http.StatusGone, resp.StatusCode)
		}
		if got := registry.SourceError(url); !errors.Is(got, ErrSourceClosed) {
			t.Fatalf("expected closed source error, got %v", got)
		}
	})
}

func TestRegistryDoesNotRecordResponseWriteError(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("payload")), nil
	}, &CreateOptions{Size: 7})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	writerErr := errors.New("client connection closed")
	registry.ServeHTTP(&failingResponseWriter{header: make(http.Header), err: writerErr}, req)
	if got := registry.SourceError(url); got != nil {
		t.Fatalf("response write error was recorded as a source error: %v", got)
	}
}

func TestRegistryTaskReferencesShareSource(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateBlob([]byte("shared"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Acquire(url); err != nil {
		t.Fatal(err)
	}
	if err := registry.Acquire(url); err != nil {
		t.Fatal(err)
	}
	if err := registry.Release(url); err != nil {
		t.Fatal(err)
	}
	if !registry.IsURL(url) {
		t.Fatal("the first task release revoked a source still used by another task")
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil || string(body) != "shared" {
		t.Fatalf("unexpected shared response body %q, error %v", body, readErr)
	}

	if err := registry.Release(url); err != nil {
		t.Fatal(err)
	}
	if registry.IsURL(url) {
		t.Fatal("the final task release did not revoke the source")
	}
}

func TestRegistryUnclaimedSessionSourceExpires(t *testing.T) {
	setUnclaimedSourceTTLForTest(t, 25*time.Millisecond)
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	session := newCountingSession()
	url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("unused")), nil
	}, &CreateOptions{Size: 6, Session: session})
	if err != nil {
		t.Fatal(err)
	}
	if got := session.RefCount(); got != 1 {
		t.Fatalf("unexpected initial session ref count: %d", got)
	}

	select {
	case <-session.zero:
	case <-time.After(2 * time.Second):
		t.Fatal("unclaimed source did not expire")
	}
	if registry.IsURL(url) {
		t.Fatal("expired source is still registered")
	}
	if got := session.RefCount(); got != 0 {
		t.Fatalf("expired source retained its session: %d", got)
	}
}

func TestRegistryAcquireStopsUnclaimedSourceExpiration(t *testing.T) {
	const ttl = 25 * time.Millisecond
	setUnclaimedSourceTTLForTest(t, ttl)
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	session := newCountingSession()
	url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("claimed")), nil
	}, &CreateOptions{Size: 7, Session: session})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Acquire(url); err != nil {
		t.Fatal(err)
	}

	select {
	case <-session.zero:
		t.Fatal("claimed source expired")
	case <-time.After(4 * ttl):
	}
	if !registry.IsURL(url) {
		t.Fatal("claimed source was removed")
	}
	if err := registry.Release(url); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.zero:
	case <-time.After(2 * time.Second):
		t.Fatal("final task release did not release the session")
	}
}

func TestRegistryUnclaimedExpirationKeepsActiveReaderSession(t *testing.T) {
	setUnclaimedSourceTTLForTest(t, 100*time.Millisecond)
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	session := newCountingSession()
	reader, writer := io.Pipe()
	opened := make(chan struct{})
	url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
		close(opened)
		return reader, nil
	}, &CreateOptions{Size: 7, Session: session})
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		body []byte
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			resultCh <- result{err: err}
			return
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resultCh <- result{body: body, err: err}
	}()

	select {
	case <-opened:
	case got := <-resultCh:
		t.Fatalf("request ended before the opener ran: body %q, error %v", got.body, got.err)
	case <-time.After(2 * time.Second):
		t.Fatal("active request did not open")
	}
	deadline := time.NewTimer(2 * time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for registry.IsURL(url) {
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("unclaimed source did not expire while its reader was active")
		}
	}
	if got := session.RefCount(); got != 1 {
		t.Fatalf("expiration released the active-reader session ref: %d", got)
	}
	if _, err := writer.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()

	select {
	case got := <-resultCh:
		if got.err != nil || string(got.body) != "payload" {
			t.Fatalf("active response was interrupted: body %q, error %v", got.body, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active response did not finish after source expiration")
	}
	select {
	case <-session.zero:
		if got := session.RefCount(); got != 0 {
			t.Fatalf("unexpected final session ref count: %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active-reader session ref was not released")
	}
}

func TestRegistryCloseCancelsUnclaimedExpiration(t *testing.T) {
	const ttl = 25 * time.Millisecond
	setUnclaimedSourceTTLForTest(t, ttl)
	registry := NewRegistry(t.TempDir())

	session := newCountingSession()
	_, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("unused")), nil
	}, &CreateOptions{Size: 6, Session: session})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.zero:
	case <-time.After(2 * time.Second):
		t.Fatal("registry close did not release the session")
	}
	<-time.After(4 * ttl)
	if got := session.RefCount(); got != 0 {
		t.Fatalf("expiration released the session again after close: %d", got)
	}
}

func TestRegistryRevokeKeepsActiveReaderSession(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	session := newCountingSession()
	reader, writer := io.Pipe()
	opened := make(chan struct{})
	url, err := registry.CreateOpener(func(context.Context, OpenRequest) (io.ReadCloser, error) {
		close(opened)
		return reader, nil
	}, &CreateOptions{Size: 7, Session: session})
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		body []byte
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			resultCh <- result{err: err}
			return
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resultCh <- result{body: body, err: err}
	}()

	<-opened
	if got := session.RefCount(); got != 2 {
		t.Fatalf("expected source and active-reader session refs, got %d", got)
	}
	if err := registry.Revoke(url); err != nil {
		t.Fatal(err)
	}
	if got := session.RefCount(); got != 1 {
		t.Fatalf("revoke released the active-reader session ref: %d", got)
	}
	if _, err := writer.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()

	select {
	case got := <-resultCh:
		if got.err != nil || string(got.body) != "payload" {
			t.Fatalf("active response was interrupted: body %q, error %v", got.body, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active response did not finish after revoke")
	}
	select {
	case <-session.zero:
		if got := session.RefCount(); got != 0 {
			t.Fatalf("unexpected final session ref count: %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active-reader session ref was not released")
	}
}

func TestRegistryHTTPErrorStatuses(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateBlob([]byte("secret"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	badURL := url + "-missing"
	resp, err := http.Get(badURL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected missing source status: %d", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected method status: %d", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=99-100")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("unexpected invalid range status: %d", resp.StatusCode)
	}

	if err := registry.Revoke(url); err != nil {
		t.Fatal(err)
	}
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected revoked source status: %d", resp.StatusCode)
	}
	if registry.IsURL(url) {
		t.Fatal("revoked source should not be usable")
	}
}
