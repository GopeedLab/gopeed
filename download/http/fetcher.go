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
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		DefaultFetcher: new(base.DefaultFetcher),
		pauseCh:        make(chan interface{}),
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
	// Get file filename by URL
	if file.Name == "" {
		file.Name = path.Base(req.URL)
	}
	// unknown file filename
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
	name := f.filename()
	_, err = f.Ctl.Touch(name, f.res.Size)
	if err != nil {
		return err
	}
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
				end = f.res.Size - 1
			} else {
				end = begin + chunkSize - 1
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
	f.fetch()
	return
}

func (f *Fetcher) Pause() (err error) {
	if base.DownloadStatusStart != f.status {
		return
	}
	f.status = base.DownloadStatusPause
	f.stop()
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

func (f *Fetcher) filename() string {
	// 创建文件
	var filename = f.opts.Name
	if filename == "" {
		filename = f.res.Files[0].Name
	}
	return filepath.Join(f.opts.Path, filename)
}

func (f *Fetcher) fetch() {
	errCh := make(chan error, f.opts.Connections)

	for i := 0; i < f.opts.Connections; i++ {
		go func(i int) {
			errCh <- f.fetchChunk(i, f.filename(), f.chunks[i])
		}(i)
	}

	go func() {
		var err error
		stopFlag := false
		for i := 0; i < f.opts.Connections; i++ {
			fetchErr := <-errCh
			if fetchErr != nil && !stopFlag {
				// 有一个连接失败就立即终止下载
				err = fetchErr
				stopFlag = true
				f.stop()
			}
		}

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
