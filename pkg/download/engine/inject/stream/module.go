package stream

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	protogblob "github.com/GopeedLab/gopeed/internal/protocol/gblob"
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
	CreateBlobObjectURL           func(data []byte, contentType string) (string, error)
	CreateWritableStreamObjectURL func(opts *WritableStreamObjectURLOptions) (string, error)
	RegisterWritableStreamResume  func(url string, reopen func(offset int64) error) error
	WriteWritableStreamObjectURL  func(url string, data any) error
	CloseWritableStreamObjectURL  func(url string) error
	AbortWritableStreamObjectURL  func(url string, reason string) error
	RevokeObjectURL               func(url string) error
	ProxyHandler                  func(r *http.Request) (*url.URL, error)
}

type WritableStreamObjectURLOptions struct {
	Reopenable bool
}

func isIgnorableGBlobError(err error) bool {
	return errors.Is(err, protogblob.ErrSourceRevoked) ||
		errors.Is(err, protogblob.ErrSourceNotFound) ||
		errors.Is(err, protogblob.ErrSourceClosed) ||
		errors.Is(err, protogblob.ErrSourceAborted)
}

func Enable(runtime *goja.Runtime, loop *eventloop.EventLoop, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	if err := runtime.Set("__gopeed_create_blob_object_url", func(data []byte, contentType string) string {
		if cfg.CreateBlobObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("gblob blob object url handler not configured")))
		}
		url, err := cfg.CreateBlobObjectURL(data, contentType)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		return url
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_create_writable_stream_object_url", func(reopenable bool) string {
		if cfg.CreateWritableStreamObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("gblob writable stream object url handler not configured")))
		}
		opts := &WritableStreamObjectURLOptions{Reopenable: reopenable}
		url, err := cfg.CreateWritableStreamObjectURL(opts)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		if opts.Reopenable && cfg.RegisterWritableStreamResume != nil {
			if err := cfg.RegisterWritableStreamResume(url, func(offset int64) error {
				return openWritableStreamObjectURL(loop, url, offset)
			}); err != nil {
				panic(runtime.NewGoError(err))
			}
		}
		return url
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_write_writable_stream_object_url", func(url string, data any) {
		if cfg.WriteWritableStreamObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("gblob writable stream write handler not configured")))
		}
		if err := cfg.WriteWritableStreamObjectURL(url, data); err != nil {
			if isIgnorableGBlobError(err) {
				return
			}
			panic(runtime.NewGoError(err))
		}
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_close_writable_stream_object_url", func(url string) {
		if cfg.CloseWritableStreamObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("gblob writable stream close handler not configured")))
		}
		if err := cfg.CloseWritableStreamObjectURL(url); err != nil {
			if isIgnorableGBlobError(err) {
				return
			}
			panic(runtime.NewGoError(err))
		}
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_abort_writable_stream_object_url", func(url string, reason string) {
		if cfg.AbortWritableStreamObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("gblob writable stream abort handler not configured")))
		}
		if err := cfg.AbortWritableStreamObjectURL(url, reason); err != nil {
			if isIgnorableGBlobError(err) {
				return
			}
			panic(runtime.NewGoError(err))
		}
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_revoke_object_url", func(url string) {
		if cfg.RevokeObjectURL == nil {
			panic(runtime.NewGoError(fmt.Errorf("gblob revoke handler not configured")))
		}
		if err := cfg.RevokeObjectURL(url); err != nil {
			panic(runtime.NewGoError(err))
		}
	}); err != nil {
		return err
	}
	fetchRegistry := newFetchRegistry()
	if err := runtime.Set("__gopeed_fetch_open", func(call goja.FunctionCall) goja.Value {
		reqMeta, err := exportFetchRequest(runtime, call.Argument(0))
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		meta, err := fetchRegistry.Open(runtime, cfg.ProxyHandler, reqMeta)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		return runtime.ToValue(meta)
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_fetch_read", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		chunkSize := int(call.Argument(1).ToInteger())
		chunk, done, err := fetchRegistry.Read(id, chunkSize)
		if err != nil {
			panic(runtime.NewGoError(err))
		}
		if done {
			return goja.Null()
		}
		return runtime.ToValue(runtime.NewArrayBuffer(chunk))
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
	_, err := runtime.RunString(script)
	return err
}

func openWritableStreamObjectURL(loop *eventloop.EventLoop, url string, offset int64) error {
	type result struct {
		err error
	}
	ch := make(chan result, 1)
	ok := loop.RunOnLoop(func(runtime *goja.Runtime) {
		send := func(err error) {
			select {
			case ch <- result{err: err}:
			default:
			}
		}
		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case error:
					send(v)
				case goja.Value:
					send(exportJSError(v))
				default:
					send(fmt.Errorf("panic: %v", r))
				}
			}
		}()

		fnVal := runtime.Get("__gopeed_open_writable_stream_object_url")
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			send(fmt.Errorf("gblob writable stream open helper is not callable"))
			return
		}
		value, err := fn(nil, runtime.ToValue(url), runtime.ToValue(offset))
		if err != nil {
			send(err)
			return
		}
		if promise, ok := value.Export().(*goja.Promise); ok {
			switch promise.State() {
			case goja.PromiseStateFulfilled:
				send(nil)
				return
			case goja.PromiseStateRejected:
				send(exportJSError(promise.Result()))
				return
			default:
				send(nil)
				return
			}
		}
		send(nil)
	})
	if !ok {
		return errors.New("engine loop terminated")
	}
	return (<-ch).err
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
}

type fetchStream struct {
	body io.ReadCloser
	mu   sync.Mutex
}

type fetchRequest struct {
	URL         string
	Method      string
	Headers     [][2]string
	Body        any
	Redirect    string
	Credentials string
}

type fetchOpenMeta struct {
	ID         string      `json:"id"`
	Status     int         `json:"status"`
	StatusText string      `json:"statusText"`
	URL        string      `json:"url"`
	Headers    [][2]string `json:"headers"`
}

func newFetchRegistry() *fetchRegistry {
	return &fetchRegistry{
		streams: make(map[string]*fetchStream),
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
		meta.Body = bodyVal.Export()
	}
	return meta, nil
}

func (r *fetchRegistry) Open(runtime *goja.Runtime, proxyHandler func(r *http.Request) (*url.URL, error), reqMeta *fetchRequest) (*fetchOpenMeta, error) {
	client := req.C()
	if proxyHandler != nil {
		client.SetProxy(proxyHandler)
	}
	setFetchFingerprint(client, util.SafeGet[string](runtime, xhr.FingerprintMagicKey))
	contentType, body, err := buildFetchBody(reqMeta.Body)
	if err != nil {
		return nil, err
	}
	reqBuilder := client.R()
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
	r.mu.Lock()
	r.streams[id] = &fetchStream{body: bodyCloser}
	r.mu.Unlock()
	return meta, nil
}

func (r *fetchRegistry) Read(id string, chunkSize int) ([]byte, bool, error) {
	stream := r.get(id)
	if stream == nil {
		return nil, true, nil
	}
	if chunkSize <= 0 {
		chunkSize = 64 * 1024
	}
	buf := make([]byte, chunkSize)
	stream.mu.Lock()
	n, err := stream.body.Read(buf)
	stream.mu.Unlock()
	if n > 0 {
		return buf[:n], false, nil
	}
	if err == io.EOF {
		r.Close(id)
		return nil, true, nil
	}
	if err != nil {
		r.Close(id)
		return nil, false, err
	}
	return nil, false, nil
}

func (r *fetchRegistry) Close(id string) {
	r.mu.Lock()
	stream := r.streams[id]
	delete(r.streams, id)
	r.mu.Unlock()
	if stream != nil {
		stream.mu.Lock()
		_ = stream.body.Close()
		stream.mu.Unlock()
	}
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
	case *formdata.FormData:
		pr, pw := io.Pipe()
		mw := xhr.NewMultipart(pw)
		for _, e := range v.Entries() {
			arr := e.([]any)
			key := arr[0].(string)
			val := arr[1]
			switch vv := val.(type) {
			case string:
				mw.WriteField(key, vv)
			case *file.File:
				mw.WriteFile(key, vv)
			}
		}
		go func() {
			defer pw.Close()
			defer mw.Close()
			mw.Send()
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
