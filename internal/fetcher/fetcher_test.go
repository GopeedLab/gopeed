package fetcher

import "testing"

func TestSchemeFilter_Match(t *testing.T) {
	type fields struct {
		Type    FilterType
		Pattern string
	}
	type args struct {
		uri string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "url match",
			fields: fields{
				Type:    FilterTypeUrl,
				Pattern: "https",
			},
			args: args{
				uri: "https://github.com",
			},
			want: true,
		},
		{
			name: "url not match",
			fields: fields{
				Type:    FilterTypeUrl,
				Pattern: "https",
			},
			args: args{
				uri: "ftp://github.com",
			},
			want: false,
		},
		{
			name: "file match",
			fields: fields{
				Type:    FilterTypeFile,
				Pattern: "torrent",
			},
			args: args{
				uri: "d:/temp/test.torrent",
			},
			want: true,
		},
		{
			name: "file not match",
			fields: fields{
				Type:    FilterTypeFile,
				Pattern: "torrent",
			},
			args: args{
				uri: "d:/temp/test.txt",
			},
			want: false,
		},
		{
			name: "base64 match",
			fields: fields{
				Type:    FilterTypeBase64,
				Pattern: "application/x-bittorrent",
			},
			args: args{
				uri: "data:application/x-bittorrent;base64,xxx",
			},
			want: true,
		},
		{
			name: "base64 not match",
			fields: fields{
				Type:    FilterTypeBase64,
				Pattern: "application/x-bittorrent",
			},
			args: args{
				uri: "data:application/javascript;base64,xxx",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SchemeFilter{
				Type:    tt.fields.Type,
				Pattern: tt.fields.Pattern,
			}
			if got := s.Match(tt.args.uri); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
