package util

import "testing"

func TestByteFmt(t *testing.T) {
	type args struct {
		size int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "unknown",
			args: args{size: int64(0)},
			want: "unknown",
		},
		{
			name: "100B",
			args: args{size: int64(100)},
			want: "100B",
		},
		{
			name: "1KB",
			args: args{size: int64(1024)},
			want: "1KB",
		},
		{
			name: "1.9KB",
			args: args{size: int64(1024*2 - 1)},
			want: "1.9KB",
		},
		{
			name: "2KB",
			args: args{size: int64(1024 * 2)},
			want: "2KB",
		},
		{
			name: "1MB",
			args: args{size: int64(1024 * 1024)},
			want: "1MB",
		},
		{
			name: "1.9MB",
			args: args{size: int64(1024*1024*2 - 1)},
			want: "1.9MB",
		},
		{
			name: "2MB",
			args: args{size: int64(1024 * 1024 * 2)},
			want: "2MB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByteFmt(tt.args.size); got != tt.want {
				t.Errorf("ByteFmt() = %v, want %v", got, tt.want)
			}
		})
	}
}
