package down

import (
	"reflect"
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
					"https://github.com/proxyee-down-org/proxyee-down/releases/download/3.4/proxyee-down-main.jar",
					map[string]string{
						"Host":            "github.com",
						"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
						"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
						"Referer":         "https://github.com/proxyee-down-org/proxyee-down/releases",
						"Accept-Encoding": "gzip, deflate, br",
						"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
					},
					nil,
				},
			},
			&Response{"proxyee-down-main.jar", 30159703, false},
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
