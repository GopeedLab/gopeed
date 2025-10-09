package xhr

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/formdata"
	"github.com/GopeedLab/gopeed/pkg/download/engine/util"
	"github.com/dop251/goja"
	"github.com/imroc/req/v3"
)

const (
	eventLoad             = "load"
	eventReadystatechange = "readystatechange"
	eventProgress         = "progress"
	eventAbort            = "abort"
	eventError            = "error"
	eventTimeout          = "timeout"
)

const (
	redirectError  = "error"
	redirectFollow = "follow"
	redirectManual = "manual"
)

type ProgressEvent struct {
	Type             string `json:"type"`
	LengthComputable bool   `json:"lengthComputable"`
	Loaded           int64  `json:"loaded"`
	Total            int64  `json:"total"`
}

type EventProp struct {
	eventListeners map[string]func(event *ProgressEvent)
	Onload         func(event *ProgressEvent) `json:"onload"`
	Onprogress     func(event *ProgressEvent) `json:"onprogress"`
	Onabort        func(event *ProgressEvent) `json:"onabort"`
	Onerror        func(event *ProgressEvent) `json:"onerror"`
	Ontimeout      func(event *ProgressEvent) `json:"ontimeout"`
}

func (ep *EventProp) AddEventListener(event string, cb func(event *ProgressEvent)) {
	ep.eventListeners[event] = cb
}

func (ep *EventProp) RemoveEventListener(event string) {
	delete(ep.eventListeners, event)
}

func (ep *EventProp) callOnload() {
	event := &ProgressEvent{
		Type:             eventLoad,
		LengthComputable: false,
	}
	if ep.Onload != nil {
		ep.Onload(event)
	}
	ep.callEventListener(event)
}

func (ep *EventProp) callOnprogress(loaded, total int64) {
	event := &ProgressEvent{
		Type:             eventProgress,
		LengthComputable: true,
		Loaded:           loaded,
		Total:            total,
	}
	if ep.Onprogress != nil {
		ep.Onprogress(event)
	}
	ep.callEventListener(event)
}

func (ep *EventProp) callOnabort() {
	event := &ProgressEvent{
		Type:             eventAbort,
		LengthComputable: false,
	}
	if ep.Onabort != nil {
		ep.Onabort(event)
	}
	ep.callEventListener(event)
}

func (ep *EventProp) callOnerror() {
	event := &ProgressEvent{
		Type:             eventError,
		LengthComputable: false,
	}
	if ep.Onerror != nil {
		ep.Onerror(event)
	}
	ep.callEventListener(event)
}

func (ep *EventProp) callOntimeout() {
	event := &ProgressEvent{
		Type:             eventTimeout,
		LengthComputable: false,
	}
	if ep.Ontimeout != nil {
		ep.Ontimeout(event)
	}
	ep.callEventListener(event)
}

func (ep *EventProp) callEventListener(event *ProgressEvent) {
	if cb, ok := ep.eventListeners[event.Type]; ok {
		cb(event)
	}
}

type XMLHttpRequestUpload struct {
	*EventProp
}

type XMLHttpRequest struct {
	method          string
	url             string
	requestHeaders  http.Header
	responseHeaders http.Header
	aborted         bool
	client          *req.Client
	fingerprint     string

	WithCredentials bool                  `json:"withCredentials"`
	Upload          *XMLHttpRequestUpload `json:"upload"`
	Timeout         int                   `json:"timeout"`
	ReadyState      int                   `json:"readyState"`
	Status          int                   `json:"status"`
	StatusText      string                `json:"statusText"`
	Response        string                `json:"response"`
	ResponseText    string                `json:"responseText"`
	// https://developer.mozilla.org/zh-CN/docs/Web/API/XMLHttpRequest/responseURL
	// https://xhr.spec.whatwg.org/#the-responseurl-attribute
	ResponseUrl string `json:"responseURL"`
	// extend fetch redirect
	// https://developer.mozilla.org/en-US/docs/Web/API/RequestInit#redirect
	// https://fetch.spec.whatwg.org/#concept-request-redirect-mode
	Redirect string `json:"redirect"`
	*EventProp
	Onreadystatechange func(event *ProgressEvent) `json:"onreadystatechange"`
}

func (xhr *XMLHttpRequest) Open(method, url string) {
	xhr.method = method
	xhr.url = url
	xhr.requestHeaders = make(http.Header)
	xhr.responseHeaders = make(http.Header)
	xhr.doReadystatechange(1)
}

func (xhr *XMLHttpRequest) SetRequestHeader(key, value string) {
	xhr.requestHeaders.Add(key, value)
}

func (xhr *XMLHttpRequest) Send(data goja.Value) {
	setFingerprint(xhr.client, xhr.fingerprint)

	d := xhr.parseData(data)
	var (
		contentType   string
		contentLength int64
		isStringBody  bool
	)

	// Create request using req library
	reqBuilder := xhr.client.R()

	// Set headers first
	if xhr.requestHeaders != nil {
		for key, values := range xhr.requestHeaders {
			if len(values) > 0 {
				// Merge multiple values with comma separator (HTTP standard)
				mergedValue := strings.Join(values, ", ")
				reqBuilder.SetHeader(key, mergedValue)
			}
		}
	}

	// Handle request body
	if d != nil && xhr.method != "GET" && xhr.method != "HEAD" {
		switch v := d.(type) {
		case string:
			reqBuilder.SetBody(v)
			contentType = "text/plain;charset=UTF-8"
			contentLength = int64(len(v))
			isStringBody = true
		case *file.File:
			reqBuilder.SetBody(v.Reader)
			contentType = "application/octet-stream"
			contentLength = v.Size
		case *formdata.FormData:
			pr, pw := io.Pipe()
			mw := NewMultipart(pw)
			for _, e := range v.Entries() {
				arr := e.([]any)
				k := arr[0].(string)
				v := arr[1]
				switch v := v.(type) {
				case string:
					mw.WriteField(k, v)
				case *file.File:
					mw.WriteFile(k, v)
				}
			}
			go func() {
				defer pw.Close()
				defer mw.Close()
				mw.Send()
			}()
			reqBuilder.SetBody(pr)
			contentType = mw.FormDataContentType()
			contentLength = mw.Size()
		}
	}

	// Only string body can specify Content-Type header by user
	if contentType != "" && (!isStringBody || xhr.requestHeaders.Get("Content-Type") == "") {
		reqBuilder.SetHeader("Content-Type", contentType)
	}

	// Set timeout
	if xhr.Timeout > 0 {
		xhr.client.SetTimeout(time.Duration(xhr.Timeout) * time.Millisecond)
	}

	// Configure redirect behavior
	xhr.client.SetRedirectPolicy(func(req *http.Request, via []*http.Request) error {
		if xhr.Redirect == redirectManual {
			return http.ErrUseLastResponse
		}
		if xhr.Redirect == redirectError {
			return errors.New("redirect failed")
		}
		if len(via) > 20 {
			return errors.New("too many redirects")
		}
		return nil
	})

	// Execute request
	resp, err := reqBuilder.Send(xhr.method, xhr.url)
	if err != nil {
		// handle timeout error
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			if xhr.Timeout > 0 {
				xhr.Upload.callOntimeout()
				xhr.callOntimeout()
			}
			return
		}
		xhr.Upload.callOnerror()
		xhr.callOnerror()
		return
	}

	xhr.Upload.callOnprogress(contentLength, contentLength)
	if !xhr.aborted {
		xhr.Upload.callOnload()
	}

	// Set response URL (final URL after redirects)
	if resp.Response != nil && resp.Response.Request != nil && resp.Response.Request.URL != nil {
		responseUrl := resp.Response.Request.URL
		responseUrl.Fragment = ""
		xhr.ResponseUrl = responseUrl.String()
	} else {
		xhr.ResponseUrl = xhr.url
	}

	// Set response headers
	xhr.responseHeaders = resp.Header
	xhr.Status = resp.StatusCode
	xhr.StatusText = resp.Status
	xhr.doReadystatechange(2)

	bodyBytes := resp.Bytes()
	xhr.doReadystatechange(3)
	xhr.Response = string(bodyBytes)
	xhr.ResponseText = xhr.Response
	xhr.doReadystatechange(4)
	respBodyLen := int64(len(bodyBytes))
	xhr.callOnprogress(respBodyLen, respBodyLen)
	if !xhr.aborted {
		xhr.callOnload()
	}
}

func (xhr *XMLHttpRequest) Abort() {
	xhr.doReadystatechange(0)
	xhr.aborted = true
	xhr.Upload.callOnabort()
	xhr.callOnabort()
}

func (xhr *XMLHttpRequest) GetResponseHeader(key string) string {
	if xhr.responseHeaders == nil {
		return ""
	}
	return strings.Join(xhr.responseHeaders.Values(key), ", ")
}

func (xhr *XMLHttpRequest) GetAllResponseHeaders() string {
	var buf bytes.Buffer
	for k, v := range xhr.responseHeaders {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(strings.Join(v, ", "))
		buf.WriteString("\r\n")
	}
	return buf.String()
}

func (xhr *XMLHttpRequest) callOnreadystatechange() {
	event := &ProgressEvent{
		Type:             eventReadystatechange,
		LengthComputable: false,
	}
	if xhr.Onreadystatechange != nil {
		xhr.Onreadystatechange(event)
	}
	xhr.callEventListener(event)
}

func (xhr *XMLHttpRequest) doReadystatechange(state int) {
	if xhr.aborted {
		return
	}
	xhr.ReadyState = state
	xhr.callOnreadystatechange()
}

// parse js data to go struct
func (xhr *XMLHttpRequest) parseData(data goja.Value) any {
	// check if data is null or undefined
	if data == nil || goja.IsNull(data) || goja.IsUndefined(data) || goja.IsNaN(data) {
		return nil
	}
	// check if data is File
	f, ok := data.Export().(*file.File)
	if ok {
		return f
	}
	// check if data is FormData
	fd, ok := data.Export().(*formdata.FormData)
	if ok {
		return fd
	}
	// otherwise, return data as string
	return data.String()
}

func Enable(runtime *goja.Runtime, proxyHandler func(r *http.Request) (*url.URL, error)) error {
	progressEvent := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) < 1 {
			util.ThrowTypeError(runtime, "Failed to construct 'ProgressEvent': 1 argument required, but only 0 present.")
		}
		instance := &ProgressEvent{
			Type: call.Argument(0).String(),
		}
		instanceValue := runtime.ToValue(instance).(*goja.Object)
		instanceValue.SetPrototype(call.This.Prototype())
		return instanceValue
	})
	xhr := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		// Create req client with proxy support
		client := req.C()
		if proxyHandler != nil {
			client.SetProxy(proxyHandler)
		}

		instance := &XMLHttpRequest{
			client:      client,
			fingerprint: util.SafeGet[string](runtime, FingerprintMagicKey),
			Upload: &XMLHttpRequestUpload{
				EventProp: &EventProp{
					eventListeners: make(map[string]func(event *ProgressEvent)),
				},
			},
			EventProp: &EventProp{
				eventListeners: make(map[string]func(event *ProgressEvent)),
			},
		}
		instanceValue := runtime.ToValue(instance).(*goja.Object)
		instanceValue.SetPrototype(call.This.Prototype())
		return instanceValue
	})
	runtime.Set("__gopeed_setFingerprint", func(fingerprint string) {
		runtime.Set(FingerprintMagicKey, fingerprint)
	})
	if err := runtime.Set("ProgressEvent", progressEvent); err != nil {
		return err
	}
	if err := runtime.Set("XMLHttpRequest", xhr); err != nil {
		return err
	}
	return nil
}

// Wrap multipart.Writer and stat content length
type multipartWrapper struct {
	statBuffer *bytes.Buffer
	statWriter *multipart.Writer
	writer     *multipart.Writer
	fields     map[string]any
}

func NewMultipart(w io.Writer) *multipartWrapper {
	var buf bytes.Buffer
	return &multipartWrapper{
		statBuffer: &buf,
		statWriter: multipart.NewWriter(&buf),
		writer:     multipart.NewWriter(w),
		fields:     make(map[string]any),
	}
}

func (w *multipartWrapper) WriteField(fieldname string, value string) error {
	w.fields[fieldname] = value
	return w.statWriter.WriteField(fieldname, value)
}

func (w *multipartWrapper) WriteFile(fieldname string, file *file.File) error {
	w.fields[fieldname] = file
	_, err := w.statWriter.CreateFormFile(fieldname, file.Name)
	if err != nil {
		return err
	}
	return nil
}

func (w *multipartWrapper) Size() int64 {
	w.statWriter.Close()
	size := int64(w.statBuffer.Len())
	for _, v := range w.fields {
		switch v := v.(type) {
		case *file.File:
			size += v.Size
		}
	}
	return size
}

func (w *multipartWrapper) Send() error {
	for k, v := range w.fields {
		switch v := v.(type) {
		case string:
			if err := w.writer.WriteField(k, v); err != nil {
				return err
			}
		case *file.File:
			fw, err := w.writer.CreateFormFile(k, v.Name)
			if err != nil {
				return err
			}
			if _, err = io.Copy(fw, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *multipartWrapper) FormDataContentType() string {
	return w.writer.FormDataContentType()
}

func (w *multipartWrapper) Close() error {
	return w.writer.Close()
}
