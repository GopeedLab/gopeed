package down

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestDown(t *testing.T) {
	type args struct {
		request *Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"readme",
			args{
				&Request{
					"get",
					"https://raw.githubusercontent.com/proxyee-down-org/proxyee-down/master/README.md",
					map[string]string{
						"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
						"Host":            "raw.githubusercontent.com",
						"Accept-Encoding": "gzip, deflate, br",
						"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
						"Cache-Control":   "no-cache",
						"Connection":      "keep-alive",
						"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
					},
					nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Down(tt.args.request)
		})
	}
}

func TestResolve(t *testing.T) {
	// os.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	type args struct {
		request *Request
	}
	tests := []struct {
		name    string
		args    args
		want    *Response
		wantErr bool
	}{
		{
			"proxyee-down",
			args{
				&Request{
					"get",
					"http://github.com/proxyee-down-org/proxyee-down/releases/download/3.4/proxyee-down-main.jar",
					map[string]string{
						"Host":            "github.com",
						"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
						"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
						"Referer":         "http://github.com/proxyee-down-org/proxyee-down/releases",
						"Accept-Encoding": "gzip, deflate, br",
						"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
					},
					nil,
				},
			},
			&Response{"proxyee-down-main.jar", 30159703, true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemp(t *testing.T) {
	parse, _ := url.Parse("http://www.baidu.com/asda/text.txt?a=11&b=333")
	arr := strings.Split(parse.Path, "/")
	for i := range arr {
		fmt.Println(arr[i])
	}
	// var begin, end, total int64
	//fmt.Sscanf("bytes 0-1000/1001", "%s %d-%d/%d", _, &begin, &end, &total)
	//fmt.Printf("%d-%d/%d", begin, end, total)
}
