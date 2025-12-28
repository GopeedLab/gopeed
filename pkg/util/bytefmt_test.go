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
			want: unknownSize,
		},
		{
			name: "negative value",
			args: args{size: int64(-1)},
			want: unknownSize,
		},
		{
			name: "negative min int64",
			args: args{size: int64(-9223372036854775808)},
			want: unknownSize,
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
		{
			name: "large value",
			args: args{size: int64(9223372036854775807)}, // max int64
			want: "8EB",
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
