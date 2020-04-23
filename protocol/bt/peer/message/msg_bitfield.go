package message

import (
	"github.com/RoaringBitmap/roaring"
	"math"
)

// bitfield: <len=0001+X><id=5><bitfield>
type Bitfield struct {
	*base
	pieceCount int
	bitmap     *roaring.Bitmap
}

func NewBitfield() *Bitfield {
	return &Bitfield{
		base: &base{
			id: IdBitfield,
		},
	}
}

func BuildBitfield(pieceCount int, bitmap *roaring.Bitmap) *Bitfield {
	bitfield := NewBitfield()
	bitfield.pieceCount = pieceCount
	bitfield.bitmap = bitmap
	return bitfield
}

func (b *Bitfield) Encode() []byte {
	buf := make([]byte, int(math.Ceil(float64(b.pieceCount)/8)))
	for i := 0; i < len(buf); i++ {
		temp := byte(0)
		for j := 0; j < 8; j++ {
			// 如果piece存在,位数标识为1
			if b.bitmap.ContainsInt(i*8 + j) {
				temp = temp | (1 << (7 - j))
			}
		}
		buf[i] = temp
	}
	return encode(b.base, buf)
}

func (b *Bitfield) Decode(body []byte) {
	b.bitmap = roaring.New()
	// 初始化bitmap
	for i := 0; i < len(body); i++ {
		for j := 0; j < 8; j++ {
			if body[i]&(1<<(7-j)) > 0 {
				b.bitmap.AddInt(i*8 + j)
			}
		}
	}
}

// 某个分片是否下载完成
func (b *Bitfield) IsComplete(i int) bool {
	return b.bitmap.ContainsInt(i)
}

// 给定一组分片下载状态，计算出当前peer能提供下载的分片下标，如果已经拥有则不用提供
// had = 0b10010000,has = 0b01111111
// had & has = 0b10010000 & 0b01111111 = 0b00010000
// had & has ^ has = 0b00010000 ^ 0b01111111 = 0b01101111
func (b *Bitfield) Provide(had *roaring.Bitmap) []uint32 {
	xor := roaring.Xor(roaring.And(had, b.bitmap), b.bitmap)
	return xor.ToArray()
}
