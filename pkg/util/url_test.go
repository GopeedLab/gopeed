package util

import "testing"

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
				url: "/home/bt.torrent",
			},
			want: "FILE",
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
