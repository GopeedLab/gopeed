package http

import (
	"bytes"
	"fmt"
	"github.com/monkeyWie/gopeed-core/download/common"
	"github.com/monkeyWie/gopeed-core/download/http/model"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	*common.BaseFetcher
}

func NewFetcher() *Fetcher {
	return &Fetcher{BaseFetcher: &common.BaseFetcher{}}
}

func (h *Fetcher) Protocols() []string {
	return []string{"HTTP", "HTTPS"}
}

func (h *Fetcher) Resolve(req *common.Request) (*common.Resource, error) {
	httpReq, err := BuildRequest(req)
	if err != nil {
		return nil, err
	}
	client := BuildClient()
	// 只访问一个字节，测试资源是否支持Range请求
	httpReq.Header.Set(common.HttpHeaderRange, fmt.Sprintf(common.HttpHeaderRangeFormat, 0, 0))
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	// 拿到响应头就关闭，不用加defer
	httpResp.Body.Close()
	res := &common.Resource{
		Req:   req,
		Range: false,
		Files: []*common.FileInfo{},
	}
	if common.HttpCodePartialContent == httpResp.StatusCode {
		// 返回206响应码表示支持断点下载
		res.Range = true
		// 解析资源大小: bytes 0-1000/1001 => 1001
		contentTotal := subLastSlash(httpResp.Header.Get(common.HttpHeaderContentRange))
		if contentTotal != "" {
			parse, err := strconv.ParseInt(contentTotal, 10, 64)
			if err != nil {
				return nil, err
			}
			res.Size = parse
		}
	} else if common.HttpCodeOK == httpResp.StatusCode {
		// 返回200响应码，不支持断点下载，通过Content-Length头获取文件大小，获取不到的话可能是chunked编码
		contentLength := httpResp.Header.Get(common.HttpHeaderContentLength)
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
	file := &common.FileInfo{
		Size: res.Size,
	}
	contentDisposition := httpResp.Header.Get(common.HttpHeaderContentDisposition)
	if contentDisposition != "" {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := params["filename"]
		if filename != "" {
			file.Name = filename
		}
	}
	// Get file name by URL
	if file.Name == "" {
		parse, err := url.Parse(req.URL)
		if err == nil {
			// e.g. /files/test.txt => test.txt
			file.Name = subLastSlash(parse.Path)
		}
	}
	// unknown file name
	if file.Name == "" {
		file.Name = "unknown"
	}
	res.Files = append(res.Files, file)
	return res, nil
}

func (h *Fetcher) Create(res *common.Resource, opts *common.Options) (common.Process, error) {
	if opts.Connections != 1 && !res.Range {
		opts.Connections = 1
	}
	return NewProcess(h, res, opts), nil
}

func subLastSlash(str string) string {
	if str == "" {
		return ""
	}
	index := strings.LastIndex(str, "/")
	if index != -1 {
		return str[index+1:]
	}
	return ""
}

func BuildClient() *http.Client {
	// Cookie handle
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar:     jar,
		Timeout: time.Second * 10,
	}
}

func BuildRequest(req *common.Request) (*http.Request, error) {
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
