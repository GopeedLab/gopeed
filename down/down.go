package down

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Resolve return the file response to be downloaded
func Resolve(request *Request) (*Response, error) {
	// Build request
	httpRequest, err := http.NewRequest(strings.ToUpper(request.Method), request.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range request.Header {
		httpRequest.Header.Add(k, v)
	}
	// Use "Range" header to resolve
	httpRequest.Header.Add("Range", "bytes=0-0")
	// Cookie handle
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 && response.StatusCode != 206 {
		return nil, fmt.Errorf("Response status error:%d", response.StatusCode)
	}
	ret := &Response{}
	// Get file name by "Content-Disposition"
	contentDisposition := response.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := params["filename"]
		if filename != "" {
			ret.Name = filename
		}
	}
	// Get file name by URL
	if ret.Name == "" {
		parse, err := url.Parse(request.URL)
		if err == nil {
			// e.g. /files/test.txt => test.txt
			ret.Name = subLastSlash(parse.Path)
		}
	}
	// Unknow file name
	if ret.Name == "" {
		ret.Name = "unknow"
	}
	// Is support range
	ret.Range = response.StatusCode == 206
	// Get file size
	if ret.Range {
		contentRange := response.Header.Get("Content-Range")
		if contentRange != "" {
			// e.g. bytes 0-1000/1001 => 1001
			total := subLastSlash(contentRange)
			if total != "" && total != "*" {
				parse, err := strconv.ParseInt(total, 10, 64)
				if err != nil {
					return nil, err
				}
				ret.Size = parse
			}
		}
	} else {
		contentLength := response.Header.Get("Content-Length")
		if contentLength != "" {
			ret.Size, _ = strconv.ParseInt(contentLength, 10, 64)
		}
	}
	return ret, nil
}

func subLastSlash(str string) string {
	index := strings.LastIndex(str, "/")
	if index != -1 {
		return str[index+1:]
	}
	return ""
}

// Down 下载
func Down(request *Request) {
	httpRequest, err := http.NewRequest(request.Method, request.URL, bytes.NewReader(request.content))
	if err != nil {
		fmt.Println("create http request error")
		return
	}
	for k, v := range request.Header {
		httpRequest.Header.Add(k, v)
	}
	httpClient := &http.Client{}
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		fmt.Printf("do http request error:%s\n", err)
		return
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		_, params, _ := mime.ParseMediaType(response.Header.Get("Content-Disposition"))
		filename := params["filename"]
		if len(filename) > 0 {
			fmt.Printf("filename:%s\n", filename)
		}
	}

	fmt.Printf("response status:%d %s\n", response.StatusCode, response.Status)
	fmt.Printf("response heads:%v\n", response.Header)
	file, err := os.Create("test.txt")
	if err != nil {
		fmt.Println("create file error")
		return
	}
	defer file.Close()
	fmt.Println("file close")
	io.Copy(file, response.Body)
}
