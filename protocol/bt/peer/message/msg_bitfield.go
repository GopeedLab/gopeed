package message

import "github.com/RoaringBitmap/roaring"

// bitfield: <len=0001+X><id=5><bitfield>
type Bitfield struct {
	*Message
	payload *roaring.Bitmap
}

func NewBitfield(payload *roaring.Bitmap) *Bitfield {
	return &Bitfield{
		Message: &Message{
			// TODO 长度处理
			Length: 1,
			ID:     IdBitfield,
		},
		payload: payload,
	}
}

func (b *Bitfield) Encode() []byte {
	buf := b.Message.Encode()
	for i := 0; i < int(b.Length)-len(buf); i++ {
		temp := byte(0)
		for j := 0; j < 8; j++ {
			// 如果piece存在,位数标识为1
			if b.payload.ContainsInt(i*8 + j) {
				temp = temp | (1 << (7 - j))
			}
		}
		buf = append(buf, temp)
	}
	return buf
}

func (b *Bitfield) Decode(buf []byte) Serialize {
	b.Message = &Message{}
	b.Message.Decode(buf)
	temp := buf[5:]
	b.payload = roaring.New()
	// 初始化bitmap
	for i := 0; i < len(temp); i++ {
		for j := 0; j < 8; j++ {
			if temp[i]&(1<<(7-j)) > 0 {
				b.payload.AddInt(i*8 + j)
			}
		}
	}
	return b
}

// 某个分片是否下载完成
func (b *Bitfield) IsComplete(i int) bool {
	return b.payload.ContainsInt(i)
}

// 给定一组分片下载状态，计算出当前peer能提供下载的分片下标，如果已经拥有则不用提供
// had = 0b10010000,has = 0b01111111
// had & has = 0b10010000 & 0b01111111 = 0b00010000
// had & has ^ has = 0b00010000 ^ 0b01111111 = 0b01101111
func (b *Bitfield) Provide(had *roaring.Bitmap) []uint32 {
	xor := roaring.Xor(roaring.And(had, b.payload), b.payload)
	return xor.ToArray()
}
