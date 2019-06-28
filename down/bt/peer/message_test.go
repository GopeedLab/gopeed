package peer

import (
	"strconv"
	"testing"
)

func TestMsgBitfield_get(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name string
		mb   MsgBitfield
		args args
		want bool
	}{
		{
			"case-0",
			MsgBitfield([]byte{convertByte("10000000")}),
			args{
				0,
			},
			true,
		},
		{
			"case-1",
			MsgBitfield([]byte{convertByte("01000000")}),
			args{
				1,
			},
			true,
		},
		{
			"case-2",
			MsgBitfield([]byte{convertByte("00000000")}),
			args{
				2,
			},
			false,
		},
		{
			"case-10",
			MsgBitfield([]byte{convertByte("00000000"), convertByte("00100000")}),
			args{
				10,
			},
			true,
		},
		{
			"case-11",
			MsgBitfield([]byte{convertByte("00000000"), convertByte("00100000")}),
			args{
				11,
			},
			false,
		},
		{
			"case-20",
			MsgBitfield([]byte{convertByte("00000000"), convertByte("00100000")}),
			args{
				20,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mb.get(tt.args.i); got != tt.want {
				t.Errorf("MsgBitfield.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func convertByte(bit string) byte {
	i, _ := strconv.ParseUint(bit, 2, 8)
	return byte(i)
}
