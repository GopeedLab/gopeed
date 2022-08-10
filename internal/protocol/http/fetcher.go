package http

import (
	"bytes"
	"context"
	"fmt"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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

type Fetcher struct {
	*fetcher.DefaultFetcher

	res     *base.Resource
	opts    *base.Options
	status  base.Status
	chunks  []*Chunk
	clients []*http.Response

	ctx     context.Context
	cancel  context.CancelFunc
	pauseCh chan interface{}
}

func (f *Fetcher) Resolve(req *base.Request) (*base.Resource, error) {
	httpReq, err := buildRequest(nil, req)
	if err != nil {
		return nil, err
	}
	client := buildClient()
	// 只访问一个字节，测试资源是否支持Range请求
	httpReq.Header.Set(base.HttpHeaderRange, fmt.Sprintf(base.HttpHeaderRangeFormat, 0, 0))
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	// 拿到响应头就关闭，不用加defer
	httpResp.Body.Close()
	res := &base.Resource{
		Req:   req,
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
				return nil, err
			}
			res.Length = parse
		}
	} else if base.HttpCodeOK == httpResp.StatusCode {
		// 返回200响应码，不支持断点下载，通过Content-Length头获取文件大小，获取不到的话可能是chunked编码
		contentLength := httpResp.Header.Get(base.HttpHeaderContentLength)
		if contentLength != "" {
			parse, err := strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				return nil, err
			}
			res.Length = parse
		}
	} else {
		return nil, NewRequestError(httpResp.StatusCode, httpResp.Status)
	}
	file := &base.FileInfo{
		Size: res.Length,
	}
	contentDisposition := httpResp.Header.Get(base.HttpHeaderContentDisposition)
	if contentDisposition != "" {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := params["filename"]
		if filename != "" {
			file.Name = filename
		}
	}
	// Get file filename by URL
	if file.Name == "" && strings.Count(req.URL, "/") > 2 {
		file.Name = filepath.Base(req.URL)
	}
	// unknown file filename
	if file.Name == "" {
		file.Name = "unknown"
	}
	res.Files = append(res.Files, file)
	return res, nil
}

func (f *Fetcher) Create(res *base.Resource, opts *base.Options) error {
	f.res = res
	f.opts = opts
	f.status = base.DownloadStatusReady
	return nil
}

func (f *Fetcher) Start() (err error) {
	// 创建文件
	name := f.filename()
	_, err = f.Ctl.Touch(name, f.res.Length)
	if err != nil {
		return err
	}
	f.status = base.DownloadStatusStart
	if f.res.Range {
		// 每个连接平均需要下载的分块大小
		chunkSize := f.res.Length / int64(f.opts.Connections)
		f.chunks = make([]*Chunk, f.opts.Connections)
		f.clients = make([]*http.Response, f.opts.Connections)
		for i := 0; i < f.opts.Connections; i++ {
			var (
				begin = chunkSize * int64(i)
				end   int64
			)
			if i == f.opts.Connections-1 {
				// 最后一个分块需要保证把文件下载完
				end = f.res.Length - 1
			} else {
				end = begin + chunkSize - 1
			}
			chunk := NewChunk(begin, end)
			f.chunks[i] = chunk
		}
	} else {
		// 只支持单连接下载
		f.chunks = make([]*Chunk, 1)
		f.clients = make([]*http.Response, 1)
		f.chunks[0] = NewChunk(0, 0)
	}
	f.fetch()
	return
}

func (f *Fetcher) Pause() (err error) {
	if base.DownloadStatusStart != f.status {
		return
	}
	f.status = base.DownloadStatusPause
	f.cancel()
	<-f.pauseCh
	return
}

func (f *Fetcher) Continue() (err error) {
	if base.DownloadStatusStart == f.status || base.DownloadStatusDone == f.status {
		return
	}
	f.status = base.DownloadStatusStart
	var name = f.filename()
	_, err = f.Ctl.Open(name)
	if err != nil {
		return err
	}
	f.fetch()
	return
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

func (f *Fetcher) filename() string {
	// 创建文件
	var filename = f.opts.Name
	if filename == "" {
		filename = f.res.Files[0].Name
	}
	return filepath.Join(f.opts.Path, filename)
}

func (f *Fetcher) fetch() {
	f.ctx, f.cancel = context.WithCancel(context.Background())
	eg, _ := errgroup.WithContext(f.ctx)
	for i := 0; i < f.opts.Connections; i++ {
		i := i
		eg.Go(func() error {
			return f.fetchChunk(i)
		})
	}

	go func() {
		err := eg.Wait()
		// 下载停止，关闭文件句柄
		f.Ctl.Close(f.filename())
		if f.status == base.DownloadStatusPause {
			f.pauseCh <- nil
		} else {
			if err != nil {
				f.status = base.DownloadStatusError
			} else {
				f.status = base.DownloadStatusDone
			}
			f.DoneCh <- err
		}
	}()
}

func (f *Fetcher) fetchChunk(index int) (err error) {
	filename := f.filename()
	chunk := f.chunks[index]

	httpReq, err := buildRequest(f.ctx, f.res.Req)
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
		if f.res.Range {
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
			f.clients[index] = resp
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
					_, err := f.Ctl.Write(filename, chunk.Begin+chunk.Downloaded, buf[:n])
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

	if f.status == base.DownloadStatusPause {
		chunk.Status = base.DownloadStatusPause
	} else if chunk.Downloaded >= chunk.End-chunk.Begin+1 {
		chunk.Status = base.DownloadStatusDone
	} else {
		if err != nil {
			chunk.Status = base.DownloadStatusError
		} else {
			chunk.Status = base.DownloadStatusDone
		}
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
		extra := req.Extra.(Extra)
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
			body = ioutil.NopCloser(bytes.NewBufferString(extra.Body))
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

type FetcherData struct {
	Chunks []*Chunk
}

type FetcherBuilder struct {
}

var schemes = []string{"HTTP", "HTTPS"}

func (fb *FetcherBuilder) Schemes() []string {
	return schemes
}

func (fb *FetcherBuilder) Build() fetcher.Fetcher {
	return &Fetcher{
		DefaultFetcher: new(fetcher.DefaultFetcher),
		pauseCh:        make(chan interface{}),
	}
}

func (fb *FetcherBuilder) Store(f fetcher.Fetcher) interface{} {
	_f := f.(*Fetcher)
	return &FetcherData{
		Chunks: _f.chunks,
	}
}

func (fb *FetcherBuilder) Resume(res *base.Resource, opts *base.Options, data interface{}) fetcher.Fetcher {
	fetcher := new(FetcherBuilder).Build().(*Fetcher)
	fs := data.(*FetcherData)
	fetcher.res = res
	fetcher.opts = opts
	fetcher.chunks = fs.Chunks
	fetcher.status = base.DownloadStatusPause
	return fetcher
}
