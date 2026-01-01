package util

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func TestParseSchema(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "http",
			args: args{
				url: "http://www.google.com",
			},
			want: "HTTP",
		},
		{
			name: "https",
			args: args{
				url: "https://www.google.com",
			},
			want: "HTTPS",
		},
		{
			name: "file",
			args: args{
				url: "file:///home/bt.torrent",
			},
			want: "FILE",
		},
		{
			name: "file-no-scheme",
			args: args{
				url: "./url.go",
			},
			want: "",
		},
		{
			name: "data-uri",
			args: args{
				url: "data:application/x-bittorrent;base64,test",
			},
			want: "DATA",
		},
		{
			name: "windows-path",
			args: args{
				url: "D:\\bt.torrent",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseSchema(tt.args.url); got != tt.want {
				t.Errorf("ParseSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDataUri(t *testing.T) {
	type args struct {
		uri string
	}
	type result struct {
		mime string
		data []byte
	}

	testData := []byte("test")
	testData64 := base64.StdEncoding.EncodeToString(testData)

	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "success",
			args: args{
				uri: "data:application/x-bittorrent;base64," + testData64,
			},
			want: result{
				mime: "application/x-bittorrent",
				data: testData,
			},
		},
		{
			name: "fail-dirty-data",
			args: args{
				uri: "data::application/x-bittorrent;base64,!@$",
			},
			want: result{
				mime: "",
				data: nil,
			},
		},
		{
			name: "fail-miss-data",
			args: args{
				uri: ":application/x-bittorrent;base64," + testData64,
			},
			want: result{
				mime: "",
				data: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime, data := ParseDataUri(tt.args.uri)
			got := result{
				mime: mime,
				data: data,
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDataUri() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryURLDecode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"%E7%8F%80%E5%B0%94%E8%AF%BA.zip", "珀尔诺.zip"},
		{"hello%20world.txt", "hello world.txt"},
		{"bad%2-text", "bad%2-text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TryUrlQueryUnescape(tt.input)
			if got != tt.expected {
				t.Errorf("TryUrlQueryUnescape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTryUrlPathUnescape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"%E7%8F%80%E5%B0%94%E8%AF%BA.zip", "珀尔诺.zip"},
		{"hello%20world.txt", "hello world.txt"},
		{"bad%2-text", "bad%2-text"},
		// The key difference: %2B should decode to + (not space)
		{"C%2B%2B%20Primer.txt", "C++ Primer.txt"},
		{"test%2Bfile.txt", "test+file.txt"},
		// Plus sign in path should remain as-is (not decoded to space)
		{"test+plus.txt", "test+plus.txt"},
		// Mixed encoding
		{"C%2B%2B%20%20Primer%20%20Plus.mobi", "C++  Primer  Plus.mobi"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TryUrlPathUnescape(tt.input)
			if got != tt.expected {
				t.Errorf("TryUrlPathUnescape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
