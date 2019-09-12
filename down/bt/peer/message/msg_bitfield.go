package message

// bitfield: <len=0001+X><id=5><bitfield>
type Bitfield struct {
	Message
	payload []byte
}

func NewBitfield(payload []byte) *Bitfield {
	return &Bitfield{
		Message: Message{
			Length: uint32(len(payload) + 1),
			ID:     IdBitfield,
		},
		payload: payload,
	}
}

func (b *Bitfield) Encode() []byte {
	return append(b.Message.Encode(), b.payload...)
}

func (b *Bitfield) Decode(buf []byte) Serialize {
	b.Message.Decode(buf)
	copy(b.payload, buf[5:])
	return b
}

// 某个分片是否下载完成
func (b *Bitfield) IsComplete(i int) bool {
	index := i / 8
	if index >= len(b.payload) {
		return false
	}
	return b.payload[index]&(1<<(7-i%8)) > 0
}

// 给定一组分片下载状态，计算出当前peer能提供下载的分片下标
func (b *Bitfield) Have(pieces []bool) []int {
	arr := make([]int, 0)
	length := len(pieces) / 8
	if len(pieces)%8 != 0 {
		length++
	}

	for i := 0; i < length; i++ {
		for j := 0; j < 8; j++ {
			index := i*8 + j
			// 如果此分片在本地还未下载，检查peer是否能提供该分片下载
			if index < len(pieces) &&
				!pieces[index] &&
				i < len(b.payload) &&
				b.payload[i]&(1<<(7-j)) > 0 {
				arr = append(arr, index)
			}
		}
	}
	return arr
}
