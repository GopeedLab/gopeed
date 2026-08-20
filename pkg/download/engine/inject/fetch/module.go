package fetch

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"github.com/GopeedLab/gopeed/internal/httpclient"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/util"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"golang.org/x/net/publicsuffix"
)

const FingerprintMagicKey = "__gopeed_fetch_fingerprint"

//go:embed fetch.js
var script string

type Config struct {
	ProxyHandler    func(*http.Request) (*url.URL, error)
	RegisterCleanup func(func())
}

func Enable(runtime *goja.Runtime, loop *eventloop.EventLoop, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	registry := newRegistry()
	if cfg.RegisterCleanup != nil {
		cfg.RegisterCleanup(registry.CloseAll)
	}
	if err := runtime.Set("__gopeed_setFingerprint", func(fingerprint string) {
		_ = runtime.Set(FingerprintMagicKey, fingerprint)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_open", func(call goja.FunctionCall) goja.Value {
		request, err := exportRequest(runtime, call.Argument(0))
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		fingerprint := util.SafeGet[string](runtime, FingerprintMagicKey)
		promise, resolve, reject := runtime.NewPromise()
		go func() {
			metadata, err := registry.Open(fingerprint, cfg.ProxyHandler, request)
			ok := loop.RunOnLoop(func(runtime *goja.Runtime) {
				if err != nil {
					reject(runtime.NewGoError(err))
					return
				}
				resolve(runtime.ToValue(metadata))
			})
			if !ok && metadata != nil {
				registry.Close(metadata.ID)
			}
		}()
		return runtime.ToValue(promise)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_read", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		chunkSize := int(call.Argument(1).ToInteger())
		promise, resolve, reject := runtime.NewPromise()
		go func() {
			chunk, done, err := registry.Read(id, chunkSize)
			ok := loop.RunOnLoop(func(runtime *goja.Runtime) {
				if err != nil {
					reject(runtime.NewGoError(err))
					return
				}
				if done {
					resolve(goja.Null())
					return
				}
				resolve(runtime.ToValue(runtime.NewArrayBuffer(chunk)))
			})
			if !ok {
				registry.Close(id)
			}
		}()
		return runtime.ToValue(promise)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_close", func(id string) {
		registry.Close(id)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_abort", func(id, reason string) {
		registry.Abort(id, reason)
	}); err != nil {
		return err
	}
	_, err := runtime.RunString(script)
	return err
}

type request struct {
	ID          string
	URL         string
	Method      string
	Headers     [][2]string
	Body        any
	Redirect    string
	Credentials string
}

type responseMetadata struct {
	ID         string      `json:"id"`
	Status     int         `json:"status"`
	StatusText string      `json:"statusText"`
	URL        string      `json:"url"`
	Redirected bool        `json:"redirected"`
	Headers    [][2]string `json:"headers"`
}

type formDataEntry struct {
	name  string
	value any
}

type formDataSnapshot struct {
	entries  []formDataEntry
	boundary string
}

func exportRequest(runtime *goja.Runtime, value goja.Value) (*request, error) {
	obj := value.ToObject(runtime)
	if obj == nil {
		return nil, fmt.Errorf("invalid fetch request")
	}
	result := &request{
		ID:          obj.Get("id").String(),
		URL:         obj.Get("url").String(),
		Method:      obj.Get("method").String(),
		Redirect:    obj.Get("redirect").String(),
		Credentials: obj.Get("credentials").String(),
	}
	if result.Method == "" {
		result.Method = http.MethodGet
	}
	if headersValue := obj.Get("headers"); headersValue != nil && !goja.IsUndefined(headersValue) && !goja.IsNull(headersValue) {
		if exported, ok := headersValue.Export().([]any); ok {
			for _, item := range exported {
				pair, ok := item.([]any)
				if !ok || len(pair) != 2 {
					continue
				}
				result.Headers = append(result.Headers, [2]string{fmt.Sprint(pair[0]), fmt.Sprint(pair[1])})
			}
		}
	}
	bodyValue := obj.Get("body")
	if bodyValue != nil && !goja.IsUndefined(bodyValue) && !goja.IsNull(bodyValue) {
		var body any
		var err error
		if obj.Get("bodyType").String() == "formdata" {
			body, err = snapshotFormData(bodyValue.Export())
		} else {
			body, err = snapshotBody(bodyValue.Export())
		}
		if err != nil {
			return nil, err
		}
		if snapshot, ok := body.(*formDataSnapshot); ok {
			snapshot.boundary = obj.Get("bodyBoundary").String()
		}
		result.Body = body
	}
	return result, nil
}

// snapshotBody runs on the JavaScript event loop. Go request goroutines must
// not retain mutable ArrayBuffer or FormData storage owned by goja.
func snapshotBody(body any) (any, error) {
	switch value := body.(type) {
	case nil, string:
		return value, nil
	case []byte:
		return append([]byte(nil), value...), nil
	case goja.ArrayBuffer:
		return append([]byte(nil), value.Bytes()...), nil
	case *file.File:
		return snapshotFile(value), nil
	default:
		if bytesValue, ok := value.(interface{ Bytes() []byte }); ok {
			return append([]byte(nil), bytesValue.Bytes()...), nil
		}
		return value, nil
	}
}

func snapshotFormData(body any) (*formDataSnapshot, error) {
	entries, ok := body.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid FormData entry list %T", body)
	}
	snapshot := &formDataSnapshot{entries: make([]formDataEntry, 0, len(entries))}
	for _, item := range entries {
		pair, ok := item.([]any)
		if !ok || len(pair) != 2 {
			return nil, fmt.Errorf("invalid FormData entry %T", item)
		}
		name, ok := pair[0].(string)
		if !ok {
			return nil, fmt.Errorf("invalid FormData field name %T", pair[0])
		}
		value := pair[1]
		if formFile, ok := value.(*file.File); ok {
			value = snapshotFile(formFile)
		}
		snapshot.entries = append(snapshot.entries, formDataEntry{name: name, value: value})
	}
	return snapshot, nil
}

func snapshotFile(source *file.File) *file.File {
	if source == nil {
		return nil
	}
	return &file.File{
		Reader: source.Reader,
		Closer: source.Closer,
		Name:   source.Name,
		Size:   source.Size,
	}
}

type registry struct {
	mu         sync.Mutex
	operations map[string]*operation
	ctx        context.Context
	cancel     context.CancelFunc
	jar        http.CookieJar
	closed     bool
}

type operation struct {
	ctx       context.Context
	cancel    context.CancelFunc
	readMu    sync.Mutex
	stateMu   sync.Mutex
	body      io.ReadCloser
	pending   error
	closed    bool
	closeOnce sync.Once
}

func newRegistry() *registry {
	ctx, cancel := context.WithCancel(context.Background())
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return &registry{
		operations: make(map[string]*operation),
		ctx:        ctx,
		cancel:     cancel,
		jar:        jar,
	}
}

func (r *registry) Open(fingerprint string, proxyHandler func(*http.Request) (*url.URL, error), metadata *request) (*responseMetadata, error) {
	ctx, cancel := context.WithCancel(r.ctx)
	op := &operation{ctx: ctx, cancel: cancel}
	if err := r.add(metadata.ID, op); err != nil {
		cancel()
		return nil, err
	}
	keepOperation := false
	defer func() {
		if !keepOperation {
			r.Close(metadata.ID)
		}
	}()

	contentType, body, contentLength, err := buildBody(metadata.Body)
	if err != nil {
		return nil, err
	}
	if metadata.Method == http.MethodGet || metadata.Method == http.MethodHead {
		body = nil
		contentLength = 0
	}
	httpRequest, err := http.NewRequestWithContext(ctx, metadata.Method, metadata.URL, body)
	if err != nil {
		return nil, err
	}
	for _, header := range metadata.Headers {
		httpRequest.Header.Add(header[0], header[1])
	}
	if host := httpRequest.Header.Get("Host"); host != "" {
		httpRequest.Host = host
	}
	if body != nil {
		if contentType != "" && !hasHeader(metadata.Headers, "Content-Type") {
			httpRequest.Header.Set("Content-Type", contentType)
		}
		httpRequest.ContentLength = contentLength
	}

	var jar http.CookieJar
	if metadata.Credentials != "omit" {
		jar = r.jar
	}
	redirected := false
	client, err := httpclient.NewClient(httpclient.Options{
		Client: httpclient.ClientOptions{
			Jar: jar,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				switch metadata.Redirect {
				case "manual":
					return http.ErrUseLastResponse
				case "error":
					return fmt.Errorf("redirect failed")
				default:
					redirected = len(via) > 0
					if len(via) > 20 {
						return fmt.Errorf("too many redirects")
					}
					return nil
				}
			},
		},
		Transport:     httpclient.TransportOptions{Proxy: proxyHandler},
		Impersonation: impersonationOptions(fingerprint),
	})
	if err != nil {
		return nil, err
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return nil, fmt.Errorf("Network request timed out")
		}
		return nil, fmt.Errorf("Network request failed: %w", err)
	}
	if !op.setBody(response.Body) {
		return nil, context.Canceled
	}
	keepOperation = true
	result := &responseMetadata{
		ID:         metadata.ID,
		Status:     response.StatusCode,
		StatusText: http.StatusText(response.StatusCode),
		URL:        metadata.URL,
	}
	if response.Request != nil && response.Request.URL != nil {
		responseURL := *response.Request.URL
		responseURL.Fragment = ""
		result.URL = responseURL.String()
	}
	result.Redirected = redirected
	for key, values := range response.Header {
		for _, value := range values {
			result.Headers = append(result.Headers, [2]string{key, value})
		}
	}
	return result, nil
}

func (r *registry) add(id string, op *operation) error {
	if id == "" {
		return fmt.Errorf("fetch request ID is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return context.Canceled
	}
	if _, exists := r.operations[id]; exists {
		return fmt.Errorf("duplicate fetch request ID %q", id)
	}
	r.operations[id] = op
	return nil
}

func (r *registry) Read(id string, chunkSize int) ([]byte, bool, error) {
	op := r.get(id)
	if op == nil {
		return nil, true, nil
	}
	chunk, done, err := op.read(chunkSize)
	if done || err != nil {
		r.Close(id)
	}
	return chunk, done, err
}

func (r *registry) Close(id string) {
	r.mu.Lock()
	op := r.operations[id]
	delete(r.operations, id)
	r.mu.Unlock()
	if op != nil {
		op.close()
	}
}

func (r *registry) Abort(id, _ string) {
	r.Close(id)
}

func (r *registry) CloseAll() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	operations := r.operations
	r.operations = make(map[string]*operation)
	r.mu.Unlock()
	r.cancel()
	for _, op := range operations {
		op.close()
	}
}

func (r *registry) get(id string) *operation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.operations[id]
}

func (o *operation) setBody(body io.ReadCloser) bool {
	o.stateMu.Lock()
	defer o.stateMu.Unlock()
	if o.closed {
		_ = body.Close()
		return false
	}
	o.body = body
	return true
}

func (o *operation) read(chunkSize int) ([]byte, bool, error) {
	o.readMu.Lock()
	defer o.readMu.Unlock()
	o.stateMu.Lock()
	body := o.body
	pending := o.pending
	o.pending = nil
	closed := o.closed
	o.stateMu.Unlock()
	if pending != nil {
		if errors.Is(pending, io.EOF) {
			return nil, true, nil
		}
		return nil, false, pending
	}
	if closed || body == nil {
		return nil, true, nil
	}
	if chunkSize <= 0 {
		chunkSize = 64 * 1024
	}
	buffer := make([]byte, chunkSize)
	n, err := body.Read(buffer)
	if n > 0 {
		if err != nil {
			o.stateMu.Lock()
			o.pending = err
			o.stateMu.Unlock()
		}
		return buffer[:n], false, nil
	}
	if errors.Is(err, io.EOF) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte{}, false, nil
}

func (o *operation) close() {
	o.closeOnce.Do(func() {
		o.cancel()
		o.stateMu.Lock()
		o.closed = true
		body := o.body
		o.stateMu.Unlock()
		if body != nil {
			_ = body.Close()
		}
	})
}

func buildBody(body any) (string, io.Reader, int64, error) {
	switch value := body.(type) {
	case nil:
		return "", nil, 0, nil
	case string:
		return "text/plain;charset=UTF-8", strings.NewReader(value), int64(len(value)), nil
	case []byte:
		return "", bytes.NewReader(value), int64(len(value)), nil
	case goja.ArrayBuffer:
		data := value.Bytes()
		return "", bytes.NewReader(data), int64(len(data)), nil
	case *file.File:
		return "application/octet-stream", value.Reader, value.Size, nil
	case *formDataSnapshot:
		return buildMultipartBody(value)
	default:
		if bytesValue, ok := value.(interface{ Bytes() []byte }); ok {
			data := bytesValue.Bytes()
			return "", bytes.NewReader(data), int64(len(data)), nil
		}
		data := fmt.Sprint(value)
		return "", strings.NewReader(data), int64(len(data)), nil
	}
}

type countingWriter struct {
	n int64
}

func (w *countingWriter) Write(data []byte) (int, error) {
	w.n += int64(len(data))
	return len(data), nil
}

func buildMultipartBody(snapshot *formDataSnapshot) (string, io.Reader, int64, error) {
	boundary := snapshot.boundary
	if boundary == "" {
		boundaryWriter := multipart.NewWriter(io.Discard)
		boundary = boundaryWriter.Boundary()
		_ = boundaryWriter.Close()
	}
	counter := &countingWriter{}
	statWriter := multipart.NewWriter(counter)
	if err := statWriter.SetBoundary(boundary); err != nil {
		return "", nil, 0, err
	}
	for _, entry := range snapshot.entries {
		switch value := entry.value.(type) {
		case string:
			if err := statWriter.WriteField(entry.name, value); err != nil {
				return "", nil, 0, err
			}
		case *file.File:
			if _, err := statWriter.CreateFormFile(entry.name, value.Name); err != nil {
				return "", nil, 0, err
			}
			counter.n += value.Size
		}
	}
	if err := statWriter.Close(); err != nil {
		return "", nil, 0, err
	}

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	if err := multipartWriter.SetBoundary(boundary); err != nil {
		_ = reader.Close()
		_ = writer.CloseWithError(err)
		return "", nil, 0, err
	}
	go func() {
		for _, entry := range snapshot.entries {
			var err error
			switch value := entry.value.(type) {
			case string:
				err = multipartWriter.WriteField(entry.name, value)
			case *file.File:
				var part io.Writer
				part, err = multipartWriter.CreateFormFile(entry.name, value.Name)
				if err == nil {
					_, err = io.Copy(part, value)
				}
			}
			if err != nil {
				_ = writer.CloseWithError(err)
				return
			}
		}
		if err := multipartWriter.Close(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()
	return multipartWriter.FormDataContentType(), reader, counter.n, nil
}

func hasHeader(headers [][2]string, key string) bool {
	for _, header := range headers {
		if strings.EqualFold(header[0], key) {
			return true
		}
	}
	return false
}

func impersonationOptions(fingerprint string) httpclient.ImpersonationOptions {
	switch fingerprint {
	case "chrome":
		return httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationFixed, Browser: httpclient.BrowserChrome}
	case "firefox":
		return httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationFixed, Browser: httpclient.BrowserFirefox}
	case "safari":
		return httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationFixed, Browser: httpclient.BrowserSafari}
	default:
		return httpclient.ImpersonationOptions{Mode: httpclient.ImpersonationNative}
	}
}
