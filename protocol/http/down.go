package http

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Download interface {
	Resolve(request *http.Request) (*http.Response, error)
	Down(request *http.Request) error
}

// Resolve return the file response to be downloaded
func Resolve(request *Request) (*Response, error) {
	httpRequest, err := BuildHTTPRequest(request)
	if err != nil {
		return nil, err
	}
	// Use "Range" header to resolve
	httpRequest.Header.Add("Range", "bytes=0-0")
	httpClient := BuildHTTPClient()
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 && response.StatusCode != 206 {
		return nil, fmt.Errorf("response status error:%d", response.StatusCode)
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
		parse, err := url.Parse(httpRequest.URL.String())
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

// Down
func Down(request *Request) error {
	response, err := Resolve(request)
	if err != nil {
		return err
	}
	// allocate file
	file, err := os.Create(response.Name)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(response.Size); err != nil {
		return err
	}
	// support range
	if response.Range {
		cons := 16
		chunkSize := response.Size / int64(cons)
		waitGroup := &sync.WaitGroup{}
		waitGroup.Add(cons)
		for i := 0; i < cons; i++ {
			start := int64(i) * chunkSize
			end := start + chunkSize
			if i == cons-1 {
				end = response.Size
			}
			go downChunk(request, file, start, end-1, waitGroup)
		}
		waitGroup.Wait()
	} else {
		downChunk(request, file, 0, response.Size, nil)
	}
	return nil
}

func subLastSlash(str string) string {
	index := strings.LastIndex(str, "/")
	if index != -1 {
		return str[index+1:]
	}
	return ""
}

func BuildHTTPRequest(request *Request) (*http.Request, error) {
	// Build request
	httpRequest, err := http.NewRequest(strings.ToUpper(request.Method), request.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range request.Header {
		httpRequest.Header.Add(k, v)
	}
	return httpRequest, nil
}

func BuildHTTPClient() *http.Client {
	// Cookie handle
	jar, _ := cookiejar.New(nil)

	return &http.Client{Jar: jar}
}

func downChunk(request *Request, file *os.File, start int64, end int64, waitGroup *sync.WaitGroup) {
	if waitGroup != nil {
		defer waitGroup.Done()
	}
	httpRequest, _ := BuildHTTPRequest(request)
	httpRequest.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	fmt.Printf("down %d-%d\n", start, end)
	httpClient := BuildHTTPClient()
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer httpResponse.Body.Close()
	buf := make([]byte, 8192)
	writeIndex := start
	for {
		n, err := httpResponse.Body.Read(buf)
		if n > 0 {
			writeSize, err := file.WriteAt(buf[0:n], writeIndex)
			if err != nil {
				fmt.Println(err)
				return
			}
			writeIndex += int64(writeSize)
		}
		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
				return
			}
			break
		}
	}
}
