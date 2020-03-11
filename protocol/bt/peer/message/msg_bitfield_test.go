package message

import (
	"github.com/RoaringBitmap/roaring"
	"reflect"
	"testing"
)

func TestBitfield_IsComplete(t *testing.T) {
	type fields struct {
		Message Message
		payload *roaring.Bitmap
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
			"case-1",
			fields{
				payload: roaring.BitmapOf(1, 2, 3, 4),
			},
			args{
				1,
			},
			true,
		},
		{
			"case-2",
			fields{
				payload: roaring.BitmapOf(1, 2, 3, 4),
			},
			args{
				5,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bitfield{
				Message: &tt.fields.Message,
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
		payload *roaring.Bitmap
	}
	type args struct {
		had *roaring.Bitmap
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []uint32
	}{
		{
			"case-1",
			fields{
				payload: roaring.BitmapOf(0, 1, 2, 3, 4),
			},
			args{
				roaring.BitmapOf(0),
			},
			[]uint32{1, 2, 3, 4},
		},
		{
			"case-2",
			fields{
				payload: roaring.BitmapOf(0, 1, 2, 3, 4, 5, 6, 7, 8),
			},
			args{
				roaring.BitmapOf(0, 1, 3, 5, 6, 8),
			},
			[]uint32{2, 4, 7},
		},
		{
			"case-3",
			fields{
				payload: roaring.BitmapOf(),
			},
			args{
				roaring.BitmapOf(0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
			},
			[]uint32{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bitfield{
				Message: &tt.fields.Message,
				payload: tt.fields.payload,
			}
			if got := b.Provide(tt.args.had); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bitfield.Provide() = %v, want %v", got, tt.want)
			}
		})
	}
}
