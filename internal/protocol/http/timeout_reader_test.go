package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	gohttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/httpclient"
)

func TestTimeoutReader_Read(t *testing.T) {
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)
	timeoutReader := NewTimeoutReader(reader, 1*time.Second)

	buf := make([]byte, len(data))
	n, err := timeoutReader.Read(buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected to read %d bytes, read %d", len(data), n)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("expected %s, got %s", data, buf)
	}
}

func TestTimeoutReader_ReadTimeout(t *testing.T) {
	reader := &slowReader{delay: 2 * time.Second}
	timeoutReader := NewTimeoutReader(reader, 1*time.Second)

	buf := make([]byte, 8192)
	_, err := timeoutReader.Read(buf)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
	}
}

type slowReader struct {
	delay time.Duration
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(sr.delay)
	return 0, io.EOF
}

// The timer must only cover time spent inside Read, not caller work between reads.
func TestTimeoutReader_IdleBetweenReads(t *testing.T) {
	reader := &closeRecorder{Reader: bytes.NewReader([]byte("abcdef"))}
	timeoutReader := NewTimeoutReader(reader, 50*time.Millisecond)

	buf := make([]byte, 3)
	if _, err := timeoutReader.Read(buf); err != nil {
		t.Fatalf("first read: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := timeoutReader.Read(buf); err != nil {
		t.Fatalf("second read after idle: %v", err)
	}
	if reader.closed.Load() {
		t.Fatal("reader was closed while idle between reads")
	}
}

type closeRecorder struct {
	io.Reader
	closed atomic.Bool
}

func (c *closeRecorder) Close() error {
	c.closed.Store(true)
	return nil
}

// A timed-out Read must not return while the underlying Read can still write
// into the caller's buffer.
func TestTimeoutReader_TimedOutReadDoesNotOutliveCaller(t *testing.T) {
	reader := &lateWriteReader{closed: make(chan struct{})}
	timeoutReader := NewTimeoutReader(reader, 50*time.Millisecond)

	buf := make([]byte, 8)
	_, err := timeoutReader.Read(buf)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
	}
	if !reader.returned.Load() {
		t.Fatal("Read returned before the underlying Read finished")
	}
	if _, err := timeoutReader.Read(buf); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read after timeout: expected %v, got %v", context.DeadlineExceeded, err)
	}
}

// lateWriteReader blocks until closed, then writes into p before returning.
type lateWriteReader struct {
	closeOnce sync.Once
	closed    chan struct{}
	returned  atomic.Bool
}

func (r *lateWriteReader) Read(p []byte) (int, error) {
	<-r.closed
	time.Sleep(20 * time.Millisecond)
	p[0] = 'x'
	r.returned.Store(true)
	return 0, errors.New("read on closed reader")
}

func (r *lateWriteReader) Close() error {
	r.closeOnce.Do(func() { close(r.closed) })
	return nil
}

type timeoutReaderClientCase struct {
	name   string
	http2  bool
	client func(t *testing.T, server *httptest.Server) *gohttp.Client
}

func timeoutReaderClientCases() []timeoutReaderClientCase {
	newClient := func(mode httpclient.ImpersonationMode) func(t *testing.T, server *httptest.Server) *gohttp.Client {
		return func(t *testing.T, server *httptest.Server) *gohttp.Client {
			var tlsConfig *tls.Config
			if server.TLS != nil {
				roots := x509.NewCertPool()
				roots.AddCert(server.Certificate())
				tlsConfig = &tls.Config{RootCAs: roots}
			}
			client, err := httpclient.NewClient(httpclient.Options{
				Transport: httpclient.TransportOptions{
					TLSClientConfig: tlsConfig,
				},
				Impersonation: httpclient.ImpersonationOptions{
					Mode:    mode,
					Browser: httpclient.BrowserChrome,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			return client
		}
	}
	var cases []timeoutReaderClientCase
	for _, mode := range []httpclient.ImpersonationMode{httpclient.ImpersonationNative, httpclient.ImpersonationFixed} {
		cases = append(cases,
			timeoutReaderClientCase{name: string(mode) + "/http1", client: newClient(mode)},
			timeoutReaderClientCase{name: string(mode) + "/http2", http2: true, client: newClient(mode)},
		)
	}
	return cases
}

func startTimeoutReaderServer(t *testing.T, useHTTP2 bool, handler gohttp.HandlerFunc) *httptest.Server {
	server := httptest.NewUnstartedServer(handler)
	if useHTTP2 {
		server.EnableHTTP2 = true
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)
	return server
}

func readBodyWithTimeout(t *testing.T, client *gohttp.Client, url string, timeout time.Duration, wantProto int) ([]byte, error) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.ProtoMajor != wantProto {
		t.Fatalf("expected HTTP/%d, got %s", wantProto, resp.Proto)
	}

	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(NewTimeoutReader(resp.Body, timeout))
		done <- result{data, err}
	}()
	select {
	case r := <-done:
		return r.data, r.err
	case <-time.After(timeout + 10*time.Second):
		t.Fatal("Read did not return after the read timeout")
		return nil, nil
	}
}

func TestTimeoutReader_StalledHTTPBodyTimesOut(t *testing.T) {
	for _, tc := range timeoutReaderClientCases() {
		t.Run(tc.name, func(t *testing.T) {
			server := startTimeoutReaderServer(t, tc.http2, func(w gohttp.ResponseWriter, r *gohttp.Request) {
				w.Header().Set("Content-Length", "1048576")
				w.Write(bytes.Repeat([]byte("a"), 1024))
				w.(gohttp.Flusher).Flush()
				<-r.Context().Done()
			})

			wantProto := 1
			if tc.http2 {
				wantProto = 2
			}
			const timeout = 200 * time.Millisecond
			start := time.Now()
			data, err := readBodyWithTimeout(t, tc.client(t, server), server.URL, timeout, wantProto)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
			}
			if len(data) != 1024 {
				t.Fatalf("expected the 1024 bytes sent before the stall, got %d", len(data))
			}
			if elapsed := time.Since(start); elapsed < timeout {
				t.Fatalf("timed out after %v, before the %v read timeout", elapsed, timeout)
			}
		})
	}
}

func TestTimeoutReader_SlowSteadyHTTPBodyDoesNotTimeOut(t *testing.T) {
	const (
		timeout  = 300 * time.Millisecond
		interval = 30 * time.Millisecond
		chunks   = 40 // 1.2s in total, four times the read timeout
	)
	for _, tc := range timeoutReaderClientCases() {
		t.Run(tc.name, func(t *testing.T) {
			server := startTimeoutReaderServer(t, tc.http2, func(w gohttp.ResponseWriter, r *gohttp.Request) {
				for i := 0; i < chunks; i++ {
					w.Write([]byte("0123456789"))
					w.(gohttp.Flusher).Flush()
					time.Sleep(interval)
				}
			})

			wantProto := 1
			if tc.http2 {
				wantProto = 2
			}
			data, err := readBodyWithTimeout(t, tc.client(t, server), server.URL, timeout, wantProto)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(data) != chunks*10 {
				t.Fatalf("expected %d bytes, got %d", chunks*10, len(data))
			}
		})
	}
}
