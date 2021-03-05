package http

import (
	"bytes"
	"fmt"
	"github.com/monkeyWie/gopeed-core/download/base"
	"github.com/monkeyWie/gopeed-core/download/http/model"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"path/filepath"
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

type Fetcher struct {
	*base.DefaultFetcher

	res     *base.Resource
	opts    *base.Options
	status  base.Status
	clients []*http.Response
	chunks  []*model.Chunk

	pauseCh chan interface{}
	doneCh  chan error
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		DefaultFetcher: new(base.DefaultFetcher),
		pauseCh:        make(chan interface{}),
		doneCh:         make(chan error),
	}
}

var protocols = []string{"HTTP", "HTTPS"}

func FetcherBuilder() ([]string, func() base.Fetcher) {
	return protocols, func() base.Fetcher {
		return NewFetcher()
	}
}

func (f *Fetcher) Resolve(req *base.Request) (*base.Resource, error) {
	httpReq, err := buildRequest(req)
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
			res.Size = parse
		}
	} else if base.HttpCodeOK == httpResp.StatusCode {
		// 返回200响应码，不支持断点下载，通过Content-Length头获取文件大小，获取不到的话可能是chunked编码
		contentLength := httpResp.Header.Get(base.HttpHeaderContentLength)
		if contentLength != "" {
			parse, err := strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				return nil, err
			}
			res.Size = parse
		}
	} else {
		return nil, NewRequestError(httpResp.StatusCode, httpResp.Status)
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
	// Get file name by URL
	if file.Name == "" {
		file.Name = path.Base(req.URL)
	}
	// unknown file name
	if file.Name == "" {
		file.Name = "unknown"
	}
	res.Files = append(res.Files, file)
	return res, nil
}

func (f *Fetcher) Create(res *base.Resource, opts *base.Options) error {
	if opts.Connections != 1 && !res.Range {
		opts.Connections = 1
	}
	f.res = res
	f.opts = opts
	f.status = base.DownloadStatusReady
	return nil
}

func (f *Fetcher) Start() (err error) {
	// 创建文件
	name := f.name()
	_, err = f.Ctl.Touch(name, f.res.Size)
	if err != nil {
		return err
	}
	defer f.Ctl.Close(name)
	f.status = base.DownloadStatusStart
	if f.res.Range {
		// 每个连接平均需要下载的分块大小
		chunkSize := f.res.Size / int64(f.opts.Connections)
		f.chunks = make([]*model.Chunk, f.opts.Connections)
		f.clients = make([]*http.Response, f.opts.Connections)
		for i := 0; i < f.opts.Connections; i++ {
			var (
				begin = chunkSize * int64(i)
				end   int64
			)
			if i == f.opts.Connections-1 {
				// 最后一个分块需要保证把文件下载完
				end = f.res.Size
			} else {
				end = begin + chunkSize
			}
			chunk := model.NewChunk(begin, end)
			f.chunks[i] = chunk
		}
	} else {
		// 只支持单连接下载
		f.chunks = make([]*model.Chunk, 1)
		f.clients = make([]*http.Response, 1)
		f.chunks[0] = model.NewChunk(0, 0)
	}

	return f.fetch()
}

func (f *Fetcher) Pause() (err error) {
	if base.DownloadStatusStart != f.status {
		return
	}
	f.status = base.DownloadStatusPause
	f.stop()
	<-f.pauseCh
	f.Ctl.Close(f.name())
	return
}

func (f *Fetcher) Continue() (err error) {
	if base.DownloadStatusStart == f.status || base.DownloadStatusDone == f.status {
		return
	}
	f.status = base.DownloadStatusStart
	var name = f.name()
	_, err = f.Ctl.Open(name)
	if err != nil {
		return err
	}
	defer f.Ctl.Close(name)
	return f.fetch()
}

func (f *Fetcher) name() string {
	// 创建文件
	var filename = f.opts.Name
	if filename == "" {
		filename = f.res.Files[0].Name
	}
	return filepath.Join(f.opts.Path, filename)
}

func (f *Fetcher) fetch() (err error) {
	errCh := make(chan error, f.opts.Connections)
	defer close(errCh)

	for i := 0; i < f.opts.Connections; i++ {
		go func(i int) {
			errCh <- f.fetchChunk(i, f.name(), f.chunks[i])
		}(i)
	}

	stopFlag := false
	for i := 0; i < f.opts.Connections; i++ {
		fetchErr := <-errCh
		if fetchErr != nil && !stopFlag {
			// 确保如果有暂停操作返回暂停
			if err == nil || err != base.PauseErr {
				err = fetchErr
			}
			if fetchErr != base.PauseErr {
				// 有一个连接失败就立即终止下载
				stopFlag = true
				f.stop()
			}
		}
	}

	if err != nil {
		if err == base.PauseErr {
			f.pauseCh <- nil
			err = nil
		} else {
			f.status = base.DownloadStatusError
		}
	} else {
		f.status = base.DownloadStatusDone
	}
	f.doneCh <- err
	return
}

func (f *Fetcher) stop() {
	if len(f.clients) > 0 {
		for _, client := range f.clients {
			if client != nil {
				client.Body.Close()
			}
		}
	}
}

func (f *Fetcher) fetchChunk(index int, name string, chunk *model.Chunk) (err error) {
	httpReq, err := buildRequest(f.res.Req)
	if err != nil {
		return err
	}
	var (
		client = buildClient()
		buf    = make([]byte, 8192)
	)
	// 重试5次
	for i := 0; i < 5; i++ {
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
			if f.status == base.DownloadStatusPause {
				return base.PauseErr
			}
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
			if err == base.PauseErr {
				return err
			}
			// 请求失败重试
			continue
		}
		retry, err = func() (bool, error) {
			defer resp.Body.Close()
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, err := f.Ctl.Write(name, chunk.Begin+chunk.Downloaded, buf[:n])
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
		f.chunks[index].Status = base.DownloadStatusError
	} else {
		f.chunks[index].Status = base.DownloadStatusDone
	}
	return
}

func buildClient() *http.Client {
	// Cookie handle
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:     jar,
		Timeout: time.Second * 10,
	}
}

func buildRequest(req *base.Request) (*http.Request, error) {
	url, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}
	httpReq := &http.Request{
		URL:    url,
		Header: map[string][]string{},
	}

	if req.Extra != nil {
		if extra, ok := req.Extra.(model.Extra); ok {
			if extra.Method != "" {
				httpReq.Method = extra.Method
			} else {
				httpReq.Method = http.MethodGet
			}
			if len(extra.Header) > 0 {
				for k, v := range extra.Header {
					httpReq.Header[k] = []string{v}
				}
			}
			httpReq.Body = ioutil.NopCloser(bytes.NewBufferString(extra.Body))
		}
	}
	return httpReq, nil
}
