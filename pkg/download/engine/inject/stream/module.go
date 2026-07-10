package stream

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/formdata"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/xhr"
	"github.com/GopeedLab/gopeed/pkg/download/engine/util"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/imroc/req/v3"
)

//go:embed stream.js
var script string

type Config struct {
	CreateObjectURL func(opts *ObjectURLOptions, open ObjectURLOpener) (string, error)
	RevokeObjectURL func(url string) error
	ProxyHandler    func(r *http.Request) (*url.URL, error)
	RegisterCleanup func(cleanup func())
}

type ObjectURLOptions struct {
	ContentType string
	Size        int64
	Range       bool
}

type ObjectURLOpenRequest struct {
	Offset int64
	End    int64
}

type ObjectURLOpener func(ctx context.Context, req ObjectURLOpenRequest) (io.ReadCloser, error)

func Enable(runtime *goja.Runtime, loop *eventloop.EventLoop, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	blobPipes := newBlobPipeRegistry()
	if err := runtime.Set("__gopeed_create_blob_object_url", func(call goja.FunctionCall) goja.Value {
		if cfg.CreateObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("blob object url handler not configured")))
		}
		openValue := call.Argument(0)
		if _, ok := goja.AssertFunction(openValue); !ok {
			panic(runtime.NewGoError(fmt.Errorf("blob opener must be callable")))
		}
		optsObj := call.Argument(1).ToObject(runtime)
		size := int64(0)
		contentType := ""
		rangeEnabled := false
		if optsObj != nil {
			size = optsObj.Get("size").ToInteger()
			contentType = optsObj.Get("contentType").String()
			rangeEnabled = optsObj.Get("range").ToBoolean()
		}
		if size < 0 {
			size = 0
		}
		opts := &ObjectURLOptions{
			ContentType: contentType,
			Size:        size,
			Range:       rangeEnabled,
		}
		url, err := cfg.CreateObjectURL(opts, func(ctx context.Context, req ObjectURLOpenRequest) (io.ReadCloser, error) {
			return openBlobObjectURLReader(ctx, loop, blobPipes, openValue, req)
		})
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		return runtime.ToValue(url)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_revoke_object_url", func(url string) {
		if cfg.RevokeObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("blob revoke handler not configured")))
		}
		if err := cfg.RevokeObjectURL(url); err != nil {
			panic(runtime.NewGoError(err))
		}
	}); err != nil {
		return err
	}
	fetchRegistry := newFetchRegistry()
	if cfg.RegisterCleanup != nil {
		cfg.RegisterCleanup(fetchRegistry.CloseAll)
	}
	if err := runtime.Set("__gopeed_fetch_open", func(call goja.FunctionCall) goja.Value {
		reqMeta, err := exportFetchRequest(runtime, call.Argument(0))
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		fingerprint := util.SafeGet[string](runtime, xhr.FingerprintMagicKey)
		promise, resolve, reject := runtime.NewPromise()
		go func() {
			meta, err := fetchRegistry.Open(fingerprint, cfg.ProxyHandler, reqMeta)
			ok := loop.RunOnLoop(func(runtime *goja.Runtime) {
				if err != nil {
					reject(runtime.NewGoError(err))
					return
				}
				resolve(runtime.ToValue(meta))
			})
			if !ok && meta != nil {
				fetchRegistry.Close(meta.ID)
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
			chunk, done, err := fetchRegistry.Read(id, chunkSize)
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
				fetchRegistry.Close(id)
			}
		}()
		return runtime.ToValue(promise)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_close", func(id string) {
		fetchRegistry.Close(id)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_abort", func(id string, reason string) {
		fetchRegistry.Abort(id, reason)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_blob_pipe_chunk", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		ok := blobPipes.Push(id, exportBytes(call.Argument(1)))
		return runtime.ToValue(ok)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_blob_pipe_close", func(id string) {
		blobPipes.Close(id, nil)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_blob_pipe_error", func(id string, message string) {
		blobPipes.Close(id, errors.New(message))
	}); err != nil {
		return err
	}
	_, err := runtime.RunString(script)
	return err
}

func openBlobObjectURLReader(ctx context.Context, loop *eventloop.EventLoop, pipes *blobPipeRegistry, openValue goja.Value, req ObjectURLOpenRequest) (io.ReadCloser, error) {
	if pipes == nil {
		return nil, fmt.Errorf("blob pipe registry not configured")
	}
	pipe := newBlobPipe(ctx, loop, pipes)
	pipes.Add(pipe)
	_, err := runOnLoop(loop, func(runtime *goja.Runtime) (goja.Value, error) {
		fnVal := runtime.Get("__gopeed_blob_pipe_source")
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			return nil, fmt.Errorf("blob pipe helper is not callable")
		}
		request := map[string]any{
			"offset": req.Offset,
			"end":    req.End,
		}
		return fn(nil, openValue, runtime.ToValue(request), runtime.ToValue(pipe.id))
	})
	if err != nil {
		pipe.Close()
		return nil, err
	}
	return pipe, nil
}

type blobPipeRegistry struct {
	mu    sync.Mutex
	pipes map[string]*blobPipe
	next  atomic.Uint64
}

func newBlobPipeRegistry() *blobPipeRegistry {
	return &blobPipeRegistry{
		pipes: map[string]*blobPipe{},
	}
}

func (r *blobPipeRegistry) Add(pipe *blobPipe) {
	r.mu.Lock()
	r.pipes[pipe.id] = pipe
	r.mu.Unlock()
}

func (r *blobPipeRegistry) Remove(id string) {
	r.mu.Lock()
	delete(r.pipes, id)
	r.mu.Unlock()
}

func (r *blobPipeRegistry) Push(id string, chunk []byte) bool {
	r.mu.Lock()
	pipe := r.pipes[id]
	r.mu.Unlock()
	if pipe == nil {
		return false
	}
	return pipe.Push(chunk)
}

func (r *blobPipeRegistry) Close(id string, err error) {
	r.mu.Lock()
	pipe := r.pipes[id]
	r.mu.Unlock()
	if pipe != nil {
		pipe.CloseWithError(err)
	}
}

func (r *blobPipeRegistry) NewID() string {
	return fmt.Sprintf("pipe-%d", r.next.Add(1))
}

type blobPipeItem struct {
	chunk []byte
	err   error
	done  bool
}

type blobPipe struct {
	id       string
	loop     *eventloop.EventLoop
	registry *blobPipeRegistry
	ctx      context.Context
	cancel   context.CancelFunc
	ch       chan blobPipeItem

	mu     sync.Mutex
	buf    []byte
	closed bool
	once   sync.Once
}

func newBlobPipe(ctx context.Context, loop *eventloop.EventLoop, registry *blobPipeRegistry) *blobPipe {
	pipeCtx, cancel := context.WithCancel(ctx)
	return &blobPipe{
		id:       registry.NewID(),
		loop:     loop,
		registry: registry,
		ctx:      pipeCtx,
		cancel:   cancel,
		ch:       make(chan blobPipeItem, 8),
	}
}

func (p *blobPipe) Push(chunk []byte) bool {
	if len(chunk) == 0 {
		return true
	}
	buf := append([]byte(nil), chunk...)
	select {
	case p.ch <- blobPipeItem{chunk: buf}:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *blobPipe) CloseWithError(err error) {
	p.once.Do(func() {
		p.registry.Remove(p.id)
		select {
		case p.ch <- blobPipeItem{done: true, err: err}:
		case <-p.ctx.Done():
		}
	})
}

func (p *blobPipe) Read(dst []byte) (int, error) {
	p.mu.Lock()
	if len(p.buf) > 0 {
		n := copy(dst, p.buf)
		p.buf = p.buf[n:]
		p.mu.Unlock()
		return n, nil
	}
	p.mu.Unlock()

	select {
	case item := <-p.ch:
		if item.done {
			if item.err != nil {
				return 0, item.err
			}
			return 0, io.EOF
		}
		n := copy(dst, item.chunk)
		if n < len(item.chunk) {
			p.mu.Lock()
			p.buf = append(p.buf, item.chunk[n:]...)
			p.mu.Unlock()
		}
		return n, nil
	case <-p.ctx.Done():
		return 0, p.ctx.Err()
	}
}

func (p *blobPipe) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	p.registry.Remove(p.id)
	p.cancel()
	p.cancelSourceReader()
	return nil
}

func (p *blobPipe) cancelSourceReader() {
	if p.loop == nil {
		return
	}
	// Invoke the JavaScript cancellation before returning from Close. The
	// source session may be released immediately after the HTTP handler closes
	// this reader, so merely queueing the callback could let the engine stop
	// before reader.cancel() is called.
	_, _ = runOnLoop(p.loop, func(runtime *goja.Runtime) (goja.Value, error) {
		fnVal := runtime.Get("__gopeed_blob_cancel_pipe_source")
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			return goja.Undefined(), nil
		}
		_, err := fn(nil, runtime.ToValue(p.id), runtime.ToValue("blob request closed"))
		return goja.Undefined(), err
	})
}

func exportBytes(value goja.Value) []byte {
	if value == nil || goja.IsNull(value) || goja.IsUndefined(value) {
		return nil
	}
	if ab, ok := value.Export().(goja.ArrayBuffer); ok {
		return ab.Bytes()
	}
	if b, ok := value.Export().([]byte); ok {
		return b
	}
	return nil
}

type blobObjectURLReader struct {
	ctx  context.Context
	loop *eventloop.EventLoop
	id   string

	mu     sync.Mutex
	buf    []byte
	closed bool
}

func (r *blobObjectURLReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	if len(r.buf) > 0 {
		n := copy(p, r.buf)
		r.buf = r.buf[n:]
		r.mu.Unlock()
		return n, nil
	}
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return 0, io.EOF
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	chunkSize := len(p)
	if chunkSize <= 0 {
		chunkSize = 32 * 1024
	}
	value, err := runOnLoop(r.loop, func(runtime *goja.Runtime) (goja.Value, error) {
		fnVal := runtime.Get("__gopeed_blob_read_source")
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			return nil, fmt.Errorf("blob read helper is not callable")
		}
		return fn(nil, runtime.ToValue(r.id), runtime.ToValue(chunkSize))
	})
	if err != nil {
		return 0, err
	}
	if goja.IsNull(value) || goja.IsUndefined(value) {
		return 0, io.EOF
	}
	var chunk []byte
	if ab, ok := value.Export().(goja.ArrayBuffer); ok {
		chunk = ab.Bytes()
	} else if b, ok := value.Export().([]byte); ok {
		chunk = b
	} else {
		return 0, fmt.Errorf("blob read helper returned %T", value.Export())
	}
	if len(chunk) == 0 {
		return 0, nil
	}
	n := copy(p, chunk)
	if n < len(chunk) {
		r.mu.Lock()
		r.buf = append(r.buf, chunk[n:]...)
		r.mu.Unlock()
	}
	return n, nil
}

func (r *blobObjectURLReader) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()
	_, err := runOnLoop(r.loop, func(runtime *goja.Runtime) (goja.Value, error) {
		fnVal := runtime.Get("__gopeed_blob_close_source")
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			return nil, nil
		}
		return fn(nil, runtime.ToValue(r.id))
	})
	return err
}

func runOnLoop(loop *eventloop.EventLoop, fn func(runtime *goja.Runtime) (goja.Value, error)) (goja.Value, error) {
	type result struct {
		value goja.Value
		err   error
	}
	ch := make(chan result, 1)
	ok := loop.RunOnLoop(func(runtime *goja.Runtime) {
		send := func(value goja.Value, err error) {
			select {
			case ch <- result{value: value, err: err}:
			default:
			}
		}
		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case error:
					send(nil, v)
				case goja.Value:
					send(nil, exportJSError(v))
				default:
					send(nil, fmt.Errorf("panic: %v", r))
				}
			}
		}()
		value, err := fn(runtime)
		if err != nil {
			send(nil, err)
			return
		}
		if promise, ok := value.Export().(*goja.Promise); ok {
			switch promise.State() {
			case goja.PromiseStateFulfilled:
				send(promise.Result(), nil)
				return
			case goja.PromiseStateRejected:
				send(nil, exportJSError(promise.Result()))
				return
			default:
				thenVal := value.ToObject(runtime).Get("then")
				thenFn, ok := goja.AssertFunction(thenVal)
				if !ok {
					send(nil, errors.New("promise.then is not callable"))
					return
				}
				onFulfilled := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
					send(call.Argument(0), nil)
					return goja.Undefined()
				})
				onRejected := runtime.ToValue(func(call goja.FunctionCall) goja.Value {
					send(nil, exportJSError(call.Argument(0)))
					return goja.Undefined()
				})
				if _, err := thenFn(value, onFulfilled, onRejected); err != nil {
					send(nil, err)
				}
				return
			}
		}
		send(value, nil)
	})
	if !ok {
		return nil, errors.New("engine loop terminated")
	}
	res := <-ch
	return res.value, res.err
}

func exportJSError(value goja.Value) error {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return errors.New("promise rejected")
	}
	if err, ok := value.Export().(error); ok {
		return err
	}
	stack := value.String()
	if ro, ok := value.(*goja.Object); ok {
		stackVal := ro.Get("stack")
		if stackVal != nil && stackVal.String() != "" {
			stack = stackVal.String()
		}
	}
	return errors.New(stack)
}

type fetchRegistry struct {
	mu      sync.Mutex
	streams map[string]*fetchStream
	ctx     context.Context
	cancel  context.CancelFunc
	closed  bool
}

type fetchStream struct {
	body      io.ReadCloser
	cancel    context.CancelFunc
	closeOnce sync.Once
	ctx       context.Context
	ch        chan fetchChunk
	readMu    sync.Mutex
	pending   []byte
}

type fetchChunk struct {
	data []byte
	err  error
}

type fetchRequest struct {
	URL         string
	Method      string
	Headers     [][2]string
	Body        any
	Redirect    string
	Credentials string
}

type fetchFormDataEntry struct {
	name  string
	value any
}

type fetchFormDataSnapshot struct {
	entries []fetchFormDataEntry
}

type fetchOpenMeta struct {
	ID         string      `json:"id"`
	Status     int         `json:"status"`
	StatusText string      `json:"statusText"`
	URL        string      `json:"url"`
	Headers    [][2]string `json:"headers"`
}

func newFetchRegistry() *fetchRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	return &fetchRegistry{
		streams: make(map[string]*fetchStream),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func exportFetchRequest(runtime *goja.Runtime, value goja.Value) (*fetchRequest, error) {
	obj := value.ToObject(runtime)
	if obj == nil {
		return nil, fmt.Errorf("invalid fetch request")
	}
	meta := &fetchRequest{
		URL:         obj.Get("url").String(),
		Method:      obj.Get("method").String(),
		Redirect:    obj.Get("redirect").String(),
		Credentials: obj.Get("credentials").String(),
	}
	if meta.Method == "" {
		meta.Method = http.MethodGet
	}
	if headersVal := obj.Get("headers"); headersVal != nil && !goja.IsUndefined(headersVal) && !goja.IsNull(headersVal) {
		if exported, ok := headersVal.Export().([]any); ok {
			for _, item := range exported {
				pair, ok := item.([]any)
				if !ok || len(pair) != 2 {
					continue
				}
				meta.Headers = append(meta.Headers, [2]string{fmt.Sprint(pair[0]), fmt.Sprint(pair[1])})
			}
		}
	}
	bodyVal := obj.Get("body")
	if bodyVal != nil && !goja.IsUndefined(bodyVal) && !goja.IsNull(bodyVal) {
		body, err := snapshotFetchBody(bodyVal.Export())
		if err != nil {
			return nil, err
		}
		meta.Body = body
	}
	return meta, nil
}

// snapshotFetchBody runs on the JavaScript event loop before fetch work is
// handed to a background goroutine. Values exported by goja may still share
// mutable storage with JavaScript (notably ArrayBuffer and FormData), so the
// background request must only retain immutable Go-owned data.
func snapshotFetchBody(body any) (any, error) {
	switch v := body.(type) {
	case nil, string:
		return v, nil
	case []byte:
		return append([]byte(nil), v...), nil
	case goja.ArrayBuffer:
		return append([]byte(nil), v.Bytes()...), nil
	case *file.File:
		return snapshotFetchFile(v), nil
	case *formdata.FormData:
		entries := v.Entries()
		snapshot := &fetchFormDataSnapshot{
			entries: make([]fetchFormDataEntry, 0, len(entries)),
		}
		for _, entry := range entries {
			pair, ok := entry.([]any)
			if !ok || len(pair) != 2 {
				return nil, fmt.Errorf("invalid FormData entry %T", entry)
			}
			name, ok := pair[0].(string)
			if !ok {
				return nil, fmt.Errorf("invalid FormData field name %T", pair[0])
			}
			value := pair[1]
			if formFile, ok := value.(*file.File); ok {
				value = snapshotFetchFile(formFile)
			}
			snapshot.entries = append(snapshot.entries, fetchFormDataEntry{
				name:  name,
				value: value,
			})
		}
		return snapshot, nil
	default:
		if typed, ok := v.(interface{ Bytes() []byte }); ok {
			return append([]byte(nil), typed.Bytes()...), nil
		}
		return v, nil
	}
}

func snapshotFetchFile(src *file.File) *file.File {
	if src == nil {
		return nil
	}
	return &file.File{
		Reader: src.Reader,
		Closer: src.Closer,
		Name:   src.Name,
		Size:   src.Size,
	}
}

func (r *fetchRegistry) Open(fingerprint string, proxyHandler func(r *http.Request) (*url.URL, error), reqMeta *fetchRequest) (*fetchOpenMeta, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, context.Canceled
	}
	registryCtx := r.ctx
	r.mu.Unlock()

	client := req.C()
	if proxyHandler != nil {
		client.SetProxy(proxyHandler)
	}
	setFetchFingerprint(client, fingerprint)
	contentType, body, err := buildFetchBody(reqMeta.Body)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(registryCtx)
	keepCancel := false
	defer func() {
		if !keepCancel {
			cancel()
		}
	}()
	reqBuilder := client.R()
	reqBuilder.SetContext(ctx)
	reqBuilder.DisableAutoReadResponse()
	for _, header := range reqMeta.Headers {
		reqBuilder.SetHeader(header[0], header[1])
	}
	if body != nil && reqMeta.Method != http.MethodGet && reqMeta.Method != http.MethodHead {
		reqBuilder.SetBody(body)
		if contentType != "" && !hasHeader(reqMeta.Headers, "Content-Type") {
			reqBuilder.SetHeader("Content-Type", contentType)
		}
	}
	client.SetRedirectPolicy(func(req *http.Request, via []*http.Request) error {
		switch reqMeta.Redirect {
		case "manual":
			return http.ErrUseLastResponse
		case "error":
			return fmt.Errorf("redirect failed")
		default:
			if len(via) > 20 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		}
	})
	resp, err := reqBuilder.Send(reqMeta.Method, reqMeta.URL)
	if err != nil {
		var ne net.Error
		if errorsAsTimeout(err, &ne) {
			return nil, fmt.Errorf("Network request timed out")
		}
		return nil, fmt.Errorf("Network request failed: %w", err)
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	meta := &fetchOpenMeta{
		ID:         id,
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		URL:        reqMeta.URL,
	}
	if resp.Response != nil && resp.Response.Request != nil && resp.Response.Request.URL != nil {
		responseURL := *resp.Response.Request.URL
		responseURL.Fragment = ""
		meta.URL = responseURL.String()
	}
	for key, values := range resp.Header {
		meta.Headers = append(meta.Headers, [2]string{key, strings.Join(values, ", ")})
	}
	bodyCloser := io.NopCloser(strings.NewReader(""))
	if resp.Response != nil && resp.Response.Body != nil {
		bodyCloser = resp.Response.Body
	}
	stream := &fetchStream{
		body:   bodyCloser,
		cancel: cancel,
		ctx:    ctx,
		ch:     make(chan fetchChunk, 8),
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		closeFetchStream(stream)
		return nil, context.Canceled
	}
	r.streams[id] = stream
	r.mu.Unlock()
	keepCancel = true
	go stream.readLoop()
	return meta, nil
}

func (r *fetchRegistry) Read(id string, chunkSize int) ([]byte, bool, error) {
	stream := r.get(id)
	if stream == nil {
		return nil, true, nil
	}
	stream.readMu.Lock()
	defer stream.readMu.Unlock()
	if len(stream.pending) > 0 {
		n := len(stream.pending)
		if chunkSize > 0 && n > chunkSize {
			n = chunkSize
		}
		chunk := append([]byte(nil), stream.pending[:n]...)
		stream.pending = stream.pending[n:]
		return chunk, false, nil
	}
	select {
	case item, ok := <-stream.ch:
		if !ok {
			r.Close(id)
			return nil, true, nil
		}
		if item.err != nil {
			r.Close(id)
			return nil, false, item.err
		}
		if chunkSize > 0 && len(item.data) > chunkSize {
			head := append([]byte(nil), item.data[:chunkSize]...)
			stream.pending = append(stream.pending[:0], item.data[chunkSize:]...)
			return head, false, nil
		}
		return item.data, false, nil
	case <-stream.ctx.Done():
		return nil, false, stream.ctx.Err()
	}
}

func (s *fetchStream) readLoop() {
	defer close(s.ch)
	buf := make([]byte, 64*1024)
	for {
		n, err := s.body.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			select {
			case s.ch <- fetchChunk{data: chunk}:
			case <-s.ctx.Done():
				return
			}
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			select {
			case s.ch <- fetchChunk{err: err}:
			case <-s.ctx.Done():
			}
			return
		}
	}
}

func (r *fetchRegistry) Close(id string) {
	r.mu.Lock()
	stream := r.streams[id]
	delete(r.streams, id)
	r.mu.Unlock()
	closeFetchStream(stream)
}

// CloseAll permanently closes the registry and cancels both active and
// in-flight fetches. It is safe to call more than once.
func (r *fetchRegistry) CloseAll() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	if r.cancel != nil {
		r.cancel()
	}
	streams := r.streams
	r.streams = make(map[string]*fetchStream)
	r.mu.Unlock()

	for _, stream := range streams {
		closeFetchStream(stream)
	}
}

func closeFetchStream(stream *fetchStream) {
	if stream == nil {
		return
	}
	stream.closeOnce.Do(func() {
		if stream.cancel != nil {
			stream.cancel()
		}
		_ = stream.body.Close()
	})
}

func (r *fetchRegistry) Abort(id string, _ string) {
	r.Close(id)
}

func (r *fetchRegistry) get(id string) *fetchStream {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.streams[id]
}

func buildFetchBody(body any) (string, any, error) {
	switch v := body.(type) {
	case nil:
		return "", nil, nil
	case string:
		return "text/plain;charset=UTF-8", v, nil
	case []byte:
		return "application/octet-stream", v, nil
	case goja.ArrayBuffer:
		return "application/octet-stream", v.Bytes(), nil
	case *file.File:
		return "application/octet-stream", v.Reader, nil
	case *fetchFormDataSnapshot:
		pr, pw := io.Pipe()
		mw := xhr.NewMultipart(pw)
		for _, entry := range v.entries {
			switch vv := entry.value.(type) {
			case string:
				if err := mw.WriteField(entry.name, vv); err != nil {
					_ = pw.CloseWithError(err)
					return "", nil, err
				}
			case *file.File:
				if err := mw.WriteFile(entry.name, vv); err != nil {
					_ = pw.CloseWithError(err)
					return "", nil, err
				}
			}
		}
		go func() {
			if err := mw.Send(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if err := mw.Close(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()
		return mw.FormDataContentType(), pr, nil
	default:
		if typed, ok := v.(interface{ Bytes() []byte }); ok {
			return "application/octet-stream", typed.Bytes(), nil
		}
		return "", fmt.Sprint(v), nil
	}
}

func hasHeader(headers [][2]string, key string) bool {
	for _, header := range headers {
		if strings.EqualFold(header[0], key) {
			return true
		}
	}
	return false
}

func setFetchFingerprint(client *req.Client, fingerprint string) {
	switch fingerprint {
	case "chrome":
		client.ImpersonateChrome()
	case "firefox":
		client.ImpersonateFirefox()
	case "safari":
		client.ImpersonateSafari()
	}
}

func errorsAsTimeout(err error, target *net.Error) bool {
	return errors.As(err, target) && (*target).Timeout()
}
