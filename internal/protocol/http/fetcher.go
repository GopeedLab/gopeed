package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"golang.org/x/sync/errgroup"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
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

	retryTimes int
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

	meta   *fetcher.FetcherMeta
	chunks []*chunk

	file   *os.File
	cancel context.CancelFunc
	eg     *errgroup.Group
}

func (f *Fetcher) Name() string {
	return "http"
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	f.doneCh = make(chan error, 1)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	exist := f.ctl.GetConfig(&f.config)
	if !exist {
		f.config = &config{
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
			Connections: 1,
		}
	}
	return
}

func (f *Fetcher) Resolve(req *base.Request) error {
	if err := base.ParseReqExtra[fhttp.ReqExtra](req); err != nil {
		return err
	}
	httpReq, err := f.buildRequest(nil, req)
	if err != nil {
		return err
	}
	client := buildClient()
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
	if base.HttpCodePartialContent == httpResp.StatusCode {
		// 返回206响应码表示支持断点下载
		res.Range = true
		// 解析资源大小: bytes 0-1000/1001 => 1001
		contentTotal := path.Base(httpResp.Header.Get(base.HttpHeaderContentRange))
		if contentTotal != "" {
			parse, err := strconv.ParseInt(contentTotal, 10, 64)
			if err != nil {
				return err
			}
			res.Size = parse
		}
	} else if base.HttpCodeOK == httpResp.StatusCode {
		// 返回200响应码，不支持断点下载，通过Content-Length头获取文件大小，获取不到的话可能是chunked编码
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
	file := &base.FileInfo{
		Size: res.Size,
	}
	contentDisposition := httpResp.Header.Get(base.HttpHeaderContentDisposition)
	if contentDisposition != "" {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := params["filename"]
		if filename != "" {
			file.Name = filename
		}
	}
	// Get file filePath by URL
	if file.Name == "" {
		file.Name = path.Base(httpReq.URL.Path)
	}
	// unknown file filePath
	if file.Name == "" || file.Name == "/" || file.Name == "." {
		file.Name = httpReq.URL.Hostname()
	}
	res.Files = append(res.Files, file)
	f.meta.Req = req
	f.meta.Res = res
	return nil
}

func (f *Fetcher) Create(opts *base.Options) error {
	f.meta.Opts = opts

	if err := base.ParseReqExtra[fhttp.ReqExtra](f.meta.Req); err != nil {
		return err
	}
	if err := base.ParseOptsExtra[fhttp.OptsExtra](f.meta.Opts); err != nil {
		return err
	}
	if opts.Extra == nil {
		opts.Extra = &fhttp.OptsExtra{}
	}
	extra := opts.Extra.(*fhttp.OptsExtra)
	if extra.Connections == 0 {
		extra.Connections = f.config.Connections
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
	if f.chunks == nil {
		f.chunks = f.splitChunk()
	}
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

func (f *Fetcher) Progress() fetcher.Progress {
	p := make(fetcher.Progress, 0)
	if len(f.chunks) > 0 {
		total := int64(0)
		for _, chunk := range f.chunks {
			total += chunk.Downloaded
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
	for i := 0; i < len(f.chunks); i++ {
		i := i
		f.eg.Go(func() error {
			return f.fetchChunk(i, ctx)
		})
	}

	go func() {
		err := f.eg.Wait()
		// check if canceled
		if errors.Is(err, context.Canceled) {
			return
		}
		f.file.Close()
		f.doneCh <- err
	}()
}

func (f *Fetcher) fetchChunk(index int, ctx context.Context) (err error) {
	chunk := f.chunks[index]
	chunk.retryTimes = 0

	httpReq, err := f.buildRequest(ctx, f.meta.Req)
	if err != nil {
		return err
	}
	var (
		client     = buildClient()
		buf        = make([]byte, 8192)
		maxRetries = 3
	)
	// retry until all remain chunks failed
	for true {
		// if chunk is completed, return
		if f.meta.Res.Range && chunk.Downloaded >= chunk.End-chunk.Begin+1 {
			return
		}

		if chunk.retryTimes >= maxRetries {
			if !f.meta.Res.Range {
				return
			}
			// check if all failed
			allFailed := true
			for _, c := range f.chunks {
				if chunk.Downloaded < chunk.End-chunk.Begin+1 && c.retryTimes < maxRetries {
					allFailed = false
					break
				}
			}
			if allFailed {
				return
			}
		}

		var (
			resp *http.Response
		)
		if f.meta.Res.Range {
			httpReq.Header.Set(base.HttpHeaderRange,
				fmt.Sprintf(base.HttpHeaderRangeFormat, chunk.Begin+chunk.Downloaded, chunk.End))
		} else {
			chunk.Downloaded = 0
		}
		err = func() error {
			resp, err = client.Do(httpReq)
			if err != nil {
				return err
			}
			if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
				err = NewRequestError(resp.StatusCode, resp.Status)
				return err
			}
			return nil
		}()
		if err != nil {
			// If canceled, do not retry
			if errors.Is(err, context.Canceled) {
				return
			}
			// retry request after 1 second
			chunk.retryTimes = chunk.retryTimes + 1
			time.Sleep(time.Second)
			continue
		}
		// Http request success, reset retry times
		chunk.retryTimes = 0
		err = func() error {
			defer resp.Body.Close()
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, err := f.file.WriteAt(buf[:n], chunk.Begin+chunk.Downloaded)
					if err != nil {
						return err
					}
					chunk.Downloaded += int64(n)
				}
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
			}
		}()
		// if chunk is completed, reset other not complete chunks retry times
		if err == nil {
			for _, c := range f.chunks {
				if c != chunk && c.retryTimes > 0 {
					c.retryTimes = 0
				}
			}
		}
		break
	}
	return
}

func (f *Fetcher) buildRequest(ctx context.Context, req *base.Request) (httpReq *http.Request, err error) {
	url, err := url.Parse(req.URL)
	if err != nil {
		return
	}

	var (
		method string
		body   io.Reader
	)
	headers := make(map[string][]string)
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
				headers[k] = []string{v}
			}
		}
		if extra.Body != "" {
			body = bytes.NewBufferString(extra.Body)
		}
	}
	if _, ok := headers[base.HttpHeaderUserAgent]; !ok {
		// load user agent from config
		headers[base.HttpHeaderUserAgent] = []string{f.config.UserAgent}
	}

	if ctx != nil {
		httpReq, err = http.NewRequestWithContext(ctx, method, url.String(), body)
	} else {
		httpReq, err = http.NewRequest(method, url.String(), body)
	}
	if err != nil {
		return
	}
	httpReq.Header = headers
	return httpReq, nil
}

func (f *Fetcher) splitChunk() (chunks []*chunk) {
	if f.meta.Res.Range {
		connections := f.meta.Opts.Extra.(*fhttp.OptsExtra).Connections
		// 每个连接平均需要下载的分块大小
		chunkSize := f.meta.Res.Size / int64(connections)
		chunks = make([]*chunk, connections)
		for i := 0; i < connections; i++ {
			var (
				begin = chunkSize * int64(i)
				end   int64
			)
			if i == connections-1 {
				// 最后一个分块需要保证把文件下载完
				end = f.meta.Res.Size - 1
			} else {
				end = begin + chunkSize - 1
			}
			chunk := newChunk(begin, end)
			chunks[i] = chunk
		}
	} else {
		// 只支持单连接下载
		chunks = make([]*chunk, 1)
		chunks[0] = newChunk(0, 0)
	}
	return
}

func buildClient() *http.Client {
	// Cookie handle
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar: jar,
	}
}

type fetcherData struct {
	Chunks []*chunk
}

type FetcherBuilder struct {
}

var schemes = []string{"HTTP", "HTTPS"}

func (fb *FetcherBuilder) Schemes() []string {
	return schemes
}

func (fb *FetcherBuilder) Build() fetcher.Fetcher {
	return &Fetcher{}
}

func (fb *FetcherBuilder) Store(f fetcher.Fetcher) (data any, err error) {
	_f := f.(*Fetcher)
	return &fetcherData{
		Chunks: _f.chunks,
	}, nil
}

func (fb *FetcherBuilder) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return &fetcherData{}, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		fd := v.(*fetcherData)
		fb := &FetcherBuilder{}
		fetcher := fb.Build().(*Fetcher)
		fetcher.meta = meta
		base.ParseReqExtra[fhttp.ReqExtra](fetcher.meta.Req)
		base.ParseOptsExtra[fhttp.OptsExtra](fetcher.meta.Opts)
		if len(fd.Chunks) > 0 {
			fetcher.chunks = fd.Chunks
		}
		return fetcher
	}
}
