package down

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Resolve 解析下载请求
func Resolve(request *Request) (*Response, error) {
	httpRequest, err := http.NewRequest(strings.ToUpper(request.Method), request.URL, nil)
	if err != nil {
		return nil, err
	}
	/* for k, v := range request.Header {
		httpRequest.Header.Add(k, v)
	} */
	httpClient := &http.Client{}
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 && response.StatusCode != 206 {
		fmt.Println(response.Header)
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(body))
		return nil, fmt.Errorf("Response status error:%d", response.StatusCode)
	}

	ret := &Response{}

	//解析文件名
	contentDisposition := response.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := params["filename"]
		if filename != "" {
			ret.Name = filename
		}
	}
	if ret.Name == "" {

	}
	//解析文件大小
	contentLength := response.Header.Get("Content-Length")
	if contentLength != "" {
		ret.size, _ = strconv.ParseInt(contentLength, 10, 64)
	}
	//判断是否支持分段下载
	ret.Partial = ret.size > 0 && response.StatusCode == 206
	return ret, nil
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
