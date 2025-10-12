package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/xiaoqidun/setft"
	"golang.org/x/sync/errgroup"
)

const (
	connectTimeout = 15 * time.Second
	readTimeout    = 15 * time.Second
	helpMinSize    = 1 * 1024 * 1024
)

type RequestError struct {
	Code int
	Msg  string
}

func NewRequestError(code int, msg string) *RequestError {
	return &RequestError{Code: code, Msg: msg}
}

func (re *RequestError) Error() string {
	return fmt.Sprintf("http request fail,code:%d", re.Code)
}

type chunk struct {
	Begin      int64
	End        int64
	Downloaded int64
}

type connection struct {
	Chunk      *chunk
	Downloaded int64
	Completed  bool

	failed     bool
	retryTimes int
}

// get remain to download bytes
func (c *chunk) remain() int64 {
	return c.End - c.Begin + 1 - c.Downloaded
}

func newChunk(begin int64, end int64) *chunk {
	return &chunk{
		Begin: begin,
		End:   end,
	}
}

type Fetcher struct {
	ctl    *controller.Controller
	config *config
	doneCh chan error

	meta         *fetcher.FetcherMeta
	connections  []*connection
	helpLock     sync.Mutex
	redirectURL  string
	redirectLock sync.Mutex

	file   *os.File
	cancel context.CancelFunc
	eg     *errgroup.Group
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	f.doneCh = make(chan error, 1)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	f.ctl.GetConfig(&f.config)
	return
}

func (f *Fetcher) Resolve(req *base.Request) error {
	if err := base.ParseReqExtra[fhttp.ReqExtra](req); err != nil {
		return err
	}
	f.meta.Req = req
	httpReq, err := f.buildRequest(nil, req)
	if err != nil {
		return err
	}
	client := f.buildClient()
	// send Range request to check whether the server supports breakpoint continuation
	// just test one byte, Range: bytes=0-0
	httpReq.Header.Set(base.HttpHeaderRange, fmt.Sprintf(base.HttpHeaderRangeFormat, 0, 0))
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	// close response body immediately
	httpResp.Body.Close()
	res := &base.Resource{
		Range: false,
		Files: []*base.FileInfo{},
	}

	if base.HttpCodePartialContent == httpResp.StatusCode || (base.HttpCodeOK == httpResp.StatusCode && httpResp.Header.Get(base.HttpHeaderAcceptRanges) == base.HttpHeaderBytes && strings.HasPrefix(httpResp.Header.Get(base.HttpHeaderContentRange), base.HttpHeaderBytes)) {
		// response 206 status code, support breakpoint continuation
		res.Range = true
		// parse content length from Content-Range header, eg: bytes 0-1000/1001 or bytes 0-0/*
		contentTotal := path.Base(httpResp.Header.Get(base.HttpHeaderContentRange))
		if contentTotal != "" && contentTotal != "*" {
			parse, err := strconv.ParseInt(contentTotal, 10, 64)
			if err != nil {
				return err
			}
			res.Size = parse
		}
	} else if base.HttpCodeOK == httpResp.StatusCode {
		// response 200 status code, not support breakpoint continuation, get file size by Content-Length header
		// if not found, maybe chunked encoding
		contentLength := httpResp.Header.Get(base.HttpHeaderContentLength)
		if contentLength != "" {
			parse, err := strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				return err
			}
			res.Size = parse
		}
	} else {
		return NewRequestError(httpResp.StatusCode, httpResp.Status)
	}
	// Parse last modified time
	var lastModifiedTime *time.Time
	lastModified := httpResp.Header.Get(base.HttpHeaderLastModified)
	if lastModified != "" {
		// ignore parse error
		t, _ := time.Parse(time.RFC1123, lastModified)
		lastModifiedTime = &t
	}
	file := &base.FileInfo{
		Size:  res.Size,
		Ctime: lastModifiedTime,
	}
	contentDisposition := httpResp.Header.Get(base.HttpHeaderContentDisposition)
	if contentDisposition != "" {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := params["filename"]
		if filename != "" {
			// Check if the filename is MIME encoded-word
			if strings.HasPrefix(filename, "=?") {
				decoder := new(mime.WordDecoder)
				filename = strings.Replace(filename, "UTF8", "UTF-8", 1)
				file.Name, _ = decoder.Decode(filename)
			} else {
				file.Name = util.TryUrlQueryUnescape(filename)
			}
		} else {
			substr := "attachment; filename="
			index := strings.Index(contentDisposition, substr)
			if index != -1 {
				file.Name = util.TryUrlQueryUnescape(contentDisposition[index+len(substr):])
			}
		}
	}
	// get file filePath by URL
	if file.Name == "" {
		file.Name = path.Base(httpReq.URL.Path)
		// Url decode
		if file.Name != "" {
			file.Name, _ = url.QueryUnescape(file.Name)
		}
	}
	// unknown file filePath
	if file.Name == "" || file.Name == "/" || file.Name == "." {
		file.Name = httpReq.URL.Hostname()
	}
	res.Files = append(res.Files, file)
	f.meta.Res = res
	return nil
}

func (f *Fetcher) Create(opts *base.Options) error {
	f.meta.Opts = opts

	if err := base.ParseOptsExtra[fhttp.OptsExtra](f.meta.Opts); err != nil {
		return err
	}
	if opts.Extra == nil {
		opts.Extra = &fhttp.OptsExtra{}
	}
	extra := opts.Extra.(*fhttp.OptsExtra)
	if extra.Connections <= 0 {
		extra.Connections = f.config.Connections
		// Avoid zero connections configuration
		if extra.Connections <= 0 {
			extra.Connections = 1
		}
	}
	return nil
}

func (f *Fetcher) Start() (err error) {
	name := f.meta.SingleFilepath()
	// if file not exist, create it, else open it
	_, err = os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			f.file, err = f.ctl.Touch(name, f.meta.Res.Size)
		} else {
			return
		}
	} else {
		f.file, err = os.OpenFile(name, os.O_RDWR, os.ModeAppend)
	}
	if err != nil {
		return err
	}

	// Avoid request extra modified by extension
	if err = base.ParseReqExtra[fhttp.ReqExtra](f.meta.Req); err != nil {
		return err
	}

	if f.connections == nil {
		f.connections = f.splitConnection()
	}
	f.redirectURL = ""
	f.fetch()
	return
}

func (f *Fetcher) Pause() (err error) {
	if f.cancel != nil {
		f.cancel()
		// wait for pause handle complete
		f.eg.Wait()
		f.file.Close()
	}
	return
}

func (f *Fetcher) Close() (err error) {
	if err = f.Pause(); err != nil {
		return
	}
	return
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Stats() any {
	statsConnections := make([]*fhttp.StatsConnection, 0)
	for _, connection := range f.connections {
		statsConnections = append(statsConnections, &fhttp.StatsConnection{
			Downloaded: connection.Downloaded,
			Completed:  connection.Completed,
			Failed:     connection.failed,
			RetryTimes: connection.retryTimes,
		})
	}
	return &fhttp.Stats{
		Connections: statsConnections,
	}
}

func (f *Fetcher) Progress() fetcher.Progress {
	p := make(fetcher.Progress, 0)
	if len(f.connections) > 0 {
		total := int64(0)
		for _, connection := range f.connections {
			total += connection.Downloaded
		}
		p = append(p, total)
	}
	return p
}

func (f *Fetcher) Wait() (err error) {
	return <-f.doneCh
}

func (f *Fetcher) fetch() {
	var ctx context.Context
	ctx, f.cancel = context.WithCancel(context.Background())
	f.eg, _ = errgroup.WithContext(ctx)
	connectionErrs := make([]error, len(f.connections))
	for i := 0; i < len(f.connections); i++ {
		i := i
		f.eg.Go(func() error {
			err := f.run(i, ctx)
			// if canceled, fail fast
			if errors.Is(err, context.Canceled) {
				return err
			}
			connectionErrs[i] = err
			return nil
		})
	}

	go func() {
		err := f.eg.Wait()
		// error returned only if canceled, just return
		if err != nil {
			return
		}
		// check all fetch results, if any error, return
		for _, chunkErr := range connectionErrs {
			if chunkErr != nil {
				err = chunkErr
				break
			}
		}

		f.file.Close()
		// Update file last modified time
		if f.config.UseServerCtime && f.meta.Res.Files[0].Ctime != nil {
			setft.SetFileTime(f.file.Name(), time.Now(), *f.meta.Res.Files[0].Ctime, *f.meta.Res.Files[0].Ctime)
		}
		f.doneCh <- err
	}()
}

func (f *Fetcher) run(index int, ctx context.Context) (err error) {
	connection := f.connections[index]
	connection.failed = false
	connection.retryTimes = 0
	var (
		client = f.buildClient()
		buf    = make([]byte, 8192)
	)

	downloadChunk := func(chunk *chunk) (err error) {
		// retry until all remain chunks failed
		for {
			// if chunk is completed, return
			if f.meta.Res.Range && chunk.remain() <= 0 {
				return nil
			}
			// if all chunks failed, return
			if connection.failed {
				allFailed := true
				for _, c := range f.connections {
					if c.Completed {
						continue
					}
					if !c.failed {
						allFailed = false
						break
					}
				}
				if allFailed {
					if connection.retryTimes >= 3 {
						return
					} else {
						connection.retryTimes++
					}
				}
			}

			err = func() error {
				var (
					httpReq *http.Request
					resp    *http.Response
				)
				f.redirectLock.Lock()
				redirectURL := f.redirectURL
				if redirectURL != "" {
					f.redirectLock.Unlock()
				} else {
					// Only hold the lock when we need to potentially set redirectURL
					defer func() {
						if redirectURL == "" && err == nil {
							f.redirectURL = resp.Request.URL.String()
						}
						f.redirectLock.Unlock()
					}()
				}

				err = func() (err error) {
					httpReq, err = f.buildRequest(ctx, f.meta.Req)
					if err != nil {
						return
					}
					if f.meta.Res.Range {
						httpReq.Header.Set(base.HttpHeaderRange,
							fmt.Sprintf(base.HttpHeaderRangeFormat, chunk.Begin+chunk.Downloaded, chunk.End))
					} else {
						chunk.Downloaded = 0
					}
					resp, err = client.Do(httpReq)
					if err != nil {
						return
					}
					return
				}()
				if err != nil {
					return err
				}

				defer resp.Body.Close()
				if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
					err = NewRequestError(resp.StatusCode, resp.Status)
					return err
				}
				connection.failed = false
				reader := NewTimeoutReader(resp.Body, readTimeout)
				for {
					n, err := reader.Read(buf)
					if n > 0 {
						finished := false
						if f.meta.Res.Range {
							remain := chunk.remain()
							// If downloaded bytes exceed the remain bytes, only write remain bytes
							if remain < int64(n) {
								n = int(remain)
								finished = true
							}
						}

						_, err := f.file.WriteAt(buf[:n], chunk.Begin+chunk.Downloaded)
						if err != nil {
							return err
						}
						chunk.Downloaded += int64(n)
						connection.Downloaded += int64(n)

						if finished {
							return nil
						}
					}
					if err != nil {
						if err == io.EOF {
							return nil
						}
						return err
					}
				}
			}()
			if err != nil {
				// If canceled, do not retry
				if errors.Is(err, context.Canceled) {
					return
				}
				// retry request after 1 second
				connection.failed = true
				time.Sleep(time.Second)
				continue
			}
			break
		}
		return
	}

	for {
		if err = downloadChunk(connection.Chunk); err != nil {
			return
		}

		// check this connection is completed
		if !f.meta.Res.Range || !f.helpOtherConnection(connection) {
			connection.Completed = true
			return
		}
	}
}

func (f *Fetcher) helpOtherConnection(helper *connection) bool {
	f.helpLock.Lock()
	defer f.helpLock.Unlock()

	// find the slowest connection
	var maxRemainConnection *connection
	var maxRemain int64
	for _, r := range f.connections {
		if r == helper || r.Completed {
			continue
		}

		remain := r.Chunk.remain()
		if remain > maxRemain && remain > helpMinSize {
			maxRemainConnection = r
			maxRemain = remain
		}
	}

	if maxRemainConnection == nil {
		return false
	}

	// re-calculate the chunk range
	helper.Chunk.Begin = maxRemainConnection.Chunk.End - maxRemainConnection.Chunk.remain()/2
	helper.Chunk.End = maxRemainConnection.Chunk.End
	helper.Chunk.Downloaded = 0
	maxRemainConnection.Chunk.End = helper.Chunk.Begin - 1
	return true
}

func (f *Fetcher) buildRequest(ctx context.Context, req *base.Request) (httpReq *http.Request, err error) {
	var reqUrl string
	if f.redirectURL != "" {
		reqUrl = f.redirectURL
	} else {
		reqUrl = req.URL
	}

	var (
		method string
		body   io.Reader
	)
	headers := http.Header{}
	if req.Extra == nil {
		method = http.MethodGet
	} else {
		extra := req.Extra.(*fhttp.ReqExtra)
		if extra.Method != "" {
			method = extra.Method
		} else {
			method = http.MethodGet
		}
		if len(extra.Header) > 0 {
			for k, v := range extra.Header {
				headers.Set(k, strings.TrimSpace(v))
			}
		}
		if extra.Body != "" {
			body = bytes.NewBufferString(extra.Body)
		}
	}
	if _, ok := headers[base.HttpHeaderUserAgent]; !ok {
		headers.Set(base.HttpHeaderUserAgent, strings.TrimSpace(f.config.UserAgent))
	}

	if ctx != nil {
		httpReq, err = http.NewRequestWithContext(ctx, method, reqUrl, body)
	} else {
		httpReq, err = http.NewRequest(method, reqUrl, body)
	}
	if err != nil {
		return
	}
	httpReq.Header = headers
	// Override Host header
	if host := headers.Get(base.HttpHeaderHost); host != "" {
		httpReq.Host = host
	}
	return httpReq, nil
}

func (f *Fetcher) splitConnection() (connections []*connection) {
	if f.meta.Res.Range {
		optConnections := f.meta.Opts.Extra.(*fhttp.OptsExtra).Connections
		// 每个连接平均需要下载的分块大小
		chunkSize := f.meta.Res.Size / int64(optConnections)
		connections = make([]*connection, optConnections)
		for i := 0; i < optConnections; i++ {
			var (
				begin = chunkSize * int64(i)
				end   int64
			)
			if i == optConnections-1 {
				// 最后一个分块需要保证把文件下载完
				end = f.meta.Res.Size - 1
			} else {
				end = begin + chunkSize - 1
			}
			connections[i] = &connection{
				Chunk: newChunk(begin, end),
			}
		}
	} else {
		// 只支持单连接下载
		connections = make([]*connection, 1)
		connections[0] = &connection{
			Chunk: newChunk(0, 0),
		}
	}
	return
}

func (f *Fetcher) buildClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
		Proxy: f.ctl.GetProxy(f.meta.Req.Proxy),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: f.meta.Req.SkipVerifyCert,
		},
	}
	// Cookie handle
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport: transport,
		Jar:       jar,
	}
}

func decodeMangledString(mangled string) string {
	rawBytes := make([]byte, 0, len(mangled)*3)
	for _, r := range mangled {
		rawBytes = append(rawBytes, byte(r))
	}
	return string(rawBytes)
}

type fetcherData struct {
	Connections []*connection
}

type FetcherManager struct {
}

func (fm *FetcherManager) Name() string {
	return "http"
}

func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: "HTTP",
		},
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: "HTTPS",
		},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{}
}

func (fm *FetcherManager) ParseName(u string) string {
	var name string
	url, err := url.Parse(u)
	if err != nil {
		return ""
	}
	// Get filePath by URL
	name = path.Base(url.Path)
	// If file name is empty, use host name
	if name == "" || name == "/" || name == "." {
		name = url.Hostname()
	}
	return name
}

func (fm *FetcherManager) AutoRename() bool {
	return true
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		Connections: 16,
	}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (data any, err error) {
	_f := f.(*Fetcher)
	return &fetcherData{
		Connections: _f.connections,
	}, nil
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherData{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		fd := v.(*fetcherData)
		fb := &FetcherManager{}
		fetcher := fb.Build().(*Fetcher)
		fetcher.meta = meta
		base.ParseReqExtra[fhttp.ReqExtra](fetcher.meta.Req)
		base.ParseOptsExtra[fhttp.OptsExtra](fetcher.meta.Opts)
		if len(fd.Connections) > 0 {
			fetcher.connections = fd.Connections
		}
		return fetcher
	}
}

func (fm *FetcherManager) Close() error {
	return nil
}
