package http

import (
	"bytes"
	"context"
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
	Status     base.Status
	Begin      int64
	End        int64
	Downloaded int64
}

func newChunk(begin int64, end int64) *chunk {
	return &chunk{
		Status: base.DownloadStatusReady,
		Begin:  begin,
		End:    end,
	}
}

type Fetcher struct {
	ctl    *controller.Controller
	doneCh chan error

	meta   *fetcher.FetcherMeta
	status base.Status
	chunks []*chunk

	file    *os.File
	ctx     context.Context
	cancel  context.CancelFunc
	pauseCh chan interface{}
}

func (f *Fetcher) Name() string {
	return "http"
}

func (f *Fetcher) Setup(ctl *controller.Controller) (err error) {
	f.ctl = ctl
	f.doneCh = make(chan error, 1)
	f.pauseCh = make(chan interface{}, 1)
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	return
}

func (f *Fetcher) Resolve(req *base.Request) error {
	if err := base.ParseReqExtra[fhttp.ReqExtra](req); err != nil {
		return err
	}
	httpReq, err := buildRequest(nil, req)
	if err != nil {
		return err
	}
	client := buildClient()
	// 只访问一个字节，测试资源是否支持Range请求
	httpReq.Header.Set(base.HttpHeaderRange, fmt.Sprintf(base.HttpHeaderRangeFormat, 0, 0))
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	// 拿到响应头就关闭，不用加defer
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
		filename := params["filePath"]
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
		file.Name = "unknown"
	}
	res.Files = append(res.Files, file)
	res.Name = file.Name
	f.meta.Req = req
	f.meta.Res = res
	return nil
}

func (f *Fetcher) Create(opts *base.Options) error {
	f.meta.Opts = opts
	f.status = base.DownloadStatusReady

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
		var cfg config
		exist, err := f.ctl.GetConfig(&cfg)
		if err != nil {
			return err
		}
		if exist {
			extra.Connections = cfg.Connections
		} else {
			extra.Connections = 1
		}
	}
	return nil
}

func (f *Fetcher) Start() (err error) {
	// 创建文件
	name := f.filepath()
	f.file, err = f.ctl.Touch(name, f.meta.Res.Size)
	if err != nil {
		return err
	}
	f.status = base.DownloadStatusRunning
	f.chunks = f.splitChunk()
	f.fetch()
	return
}

func (f *Fetcher) Pause() (err error) {
	if base.DownloadStatusRunning != f.status {
		return
	}
	f.status = base.DownloadStatusPause

	if f.cancel != nil {
		f.cancel()
		<-f.pauseCh
	}
	return
}

func (f *Fetcher) Continue() (err error) {
	if base.DownloadStatusRunning == f.status || base.DownloadStatusDone == f.status {
		return
	}
	f.status = base.DownloadStatusRunning
	f.file, err = os.OpenFile(f.filepath(), os.O_RDWR, os.ModeAppend)
	if err != nil {
		return err
	}
	f.fetch()
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

func (f *Fetcher) filepath() string {
	return f.meta.Filepath(f.meta.Res.Files[0])
}

func (f *Fetcher) fetch() {
	f.ctx, f.cancel = context.WithCancel(context.Background())
	eg, _ := errgroup.WithContext(f.ctx)
	for i := 0; i < f.meta.Opts.Extra.(*fhttp.OptsExtra).Connections; i++ {
		i := i
		eg.Go(func() error {
			return f.fetchChunk(i)
		})
	}

	go func() {
		err := eg.Wait()
		// 下载停止，关闭文件句柄
		f.file.Close()
		if f.status == base.DownloadStatusPause {
			f.pauseCh <- nil
		} else {
			if err != nil {
				f.status = base.DownloadStatusError
			} else {
				f.status = base.DownloadStatusDone
			}
			f.doneCh <- err
		}
	}()
}

func (f *Fetcher) fetchChunk(index int) (err error) {
	chunk := f.chunks[index]

	httpReq, err := buildRequest(f.ctx, f.meta.Req)
	if err != nil {
		return err
	}
	var (
		client = buildClient()
		buf    = make([]byte, 8192)
	)
	// 重试5次
	for i := 0; i < 5; i++ {
		// 如果下载完成直接返回
		if chunk.Status == base.DownloadStatusDone {
			return
		}
		// 如果已暂停直接跳出
		if f.status == base.DownloadStatusPause {
			break
		}
		var (
			resp  *http.Response
			retry bool
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
			// 请求失败重试
			continue
		}
		// 请求成功就重置错误次数，连续失败5次才终止
		i = 0
		retry, err = func() (bool, error) {
			defer resp.Body.Close()
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, err := f.file.WriteAt(buf[:n], chunk.Begin+chunk.Downloaded)
					if err != nil {
						return false, err
					}
					chunk.Downloaded += int64(n)
				}
				if err != nil {
					if err != io.EOF {
						return true, err
					}
					break
				}
			}
			return false, nil
		}()
		if !retry {
			// 下载成功，跳出重试
			break
		}
	}

	if err != nil {
		chunk.Status = base.DownloadStatusError
		return
	}

	isComplete := f.meta.Res.Range && chunk.Downloaded >= chunk.End-chunk.Begin+1
	if f.status == base.DownloadStatusPause && !isComplete {
		chunk.Status = base.DownloadStatusPause
		return
	}

	chunk.Status = base.DownloadStatusDone
	return
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
		Jar:     jar,
		Timeout: time.Second * 60,
	}
}

func buildRequest(ctx context.Context, req *base.Request) (httpReq *http.Request, err error) {
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
		fetcher.status = base.DownloadStatusPause
		base.ParseReqExtra[fhttp.ReqExtra](fetcher.meta.Req)
		base.ParseOptsExtra[fhttp.OptsExtra](fetcher.meta.Opts)
		if len(fd.Chunks) == 0 {
			fetcher.chunks = fetcher.splitChunk()
		} else {
			fetcher.chunks = fd.Chunks
		}
		return fetcher
	}
}
