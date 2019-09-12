package message

import (
	"reflect"
	"testing"
)

func TestBitfield_IsComplete(t *testing.T) {
	type fields struct {
		Message Message
		payload []byte
	}
	type args struct {
		i int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"case-0",
			fields{
				payload: []byte{0b10000000},
			},
			args{
				0,
			},
			true,
		},
		{
			"case-1",
			fields{
				payload: []byte{0b01000000},
			},
			args{
				1,
			},
			true,
		},
		{
			"case-2",
			fields{
				payload: []byte{0b00000000, 0b00100000},
			},
			args{
				2,
			},
			false,
		},
		{
			"case-10",
			fields{
				payload: []byte{0b00000000, 0b00100000},
			},
			args{
				10,
			},
			true,
		},
		{
			"case-11",
			fields{
				payload: []byte{0b00000000, 0b00100000},
			},
			args{
				11,
			},
			false,
		},
		{
			"case-20",
			fields{
				payload: []byte{0b00000000, 0b00100000},
			},
			args{
				20,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bitfield{
				Message: tt.fields.Message,
				payload: tt.fields.payload,
			}
			if got := b.IsComplete(tt.args.i); got != tt.want {
				t.Errorf("Bitfield.IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBitfield_Have(t *testing.T) {
	type fields struct {
		Message Message
		payload []byte
	}
	type args struct {
		pieces []bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []int
	}{
		{
			"case-1",
			fields{
				payload: []byte{0b11111111, 0b00000000},
			},
			args{
				[]bool{false, false, false, false, false, false, false, false},
			},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
		},
		{
			"case-2",
			fields{
				payload: []byte{0b01010101, 0b01000000},
			},
			args{
				[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
			[]int{1, 3, 5, 7, 9},
		},
		{
			"case-3",
			fields{
				payload: []byte{0b11010101, 0b01000000},
			},
			args{
				[]bool{false, true, false},
			},
			[]int{0},
		},
		{
			"case-4",
			fields{
				payload: []byte{0b11010101},
			},
			args{
				[]bool{false, true, false, false, true, false, false, true, false},
			},
			[]int{0, 3, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bitfield{
				Message: tt.fields.Message,
				payload: tt.fields.payload,
			}
			if got := b.Have(tt.args.pieces); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bitfield.Have() = %v, want %v", got, tt.want)
			}
		})
	}
}
