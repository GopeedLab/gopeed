package peer

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func TestMsgBitfield_IsComplete(t *testing.T) {
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
			if got := tt.mb.IsComplete(tt.args.i); got != tt.want {
				t.Errorf("MsgBitfield.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMsgBitfield_Have(t *testing.T) {
	type args struct {
		pieces []bool
	}
	tests := []struct {
		name string
		mb   MsgBitfield
		args args
		want []int
	}{
		{
			"case-1",
			MsgBitfield([]byte{convertByte("11111111"), convertByte("00000000")}),
			args{
				[]bool{false, false, false, false, false, false, false, false},
			},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
		},
		{
			"case-2",
			MsgBitfield([]byte{convertByte("01010101"), convertByte("01000000")}),
			args{
				[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
			[]int{1, 3, 5, 7, 9},
		},
		{
			"case-3",
			MsgBitfield([]byte{convertByte("11010101"), convertByte("01000000")}),
			args{
				[]bool{false, true, false},
			},
			[]int{0},
		},
		{
			"case-4",
			MsgBitfield([]byte{convertByte("11010101")}),
			args{
				[]bool{false, true, false, false, true, false, false, true, false},
			},
			[]int{0, 3, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mb.Have(tt.args.pieces); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MsgBitfield.Have() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMsg(t *testing.T) {
	fmt.Println(Choke)
	fmt.Println(Bitfield)
}

func convertByte(bit string) byte {
	i, _ := strconv.ParseUint(bit, 2, 8)
	return byte(i)
}
