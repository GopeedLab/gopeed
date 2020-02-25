package message

import "encoding/binary"

// piece: <len=0009+X><id=7><index><begin><block>
type Piece struct {
	*Message
	Index uint32
	Begin uint32
	Block []byte
}

func NewPiece(index uint32, begin uint32, block []byte) *Piece {
	return &Piece{
		Message: &Message{
			Length: 9 + uint32(len(block)),
			ID:     IdPiece,
		},
		Index: index,
		Begin: begin,
		// Block: io.LimitReader(reader, int64(length)),
		Block: block,
	}
}

func (p *Piece) Encode() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[:4], p.Index)
	binary.BigEndian.PutUint32(buf[4:8], p.Begin)
	buf = append(buf, p.Block...)
	return append(p.Message.Encode(), buf...)
}

func (p *Piece) Decode(buf []byte) Serialize {
	p.Message = &Message{}
	p.Message.Decode(buf)
	p.Index = binary.BigEndian.Uint32(buf[5:9])
	p.Begin = binary.BigEndian.Uint32(buf[9:13])
	p.Block = buf[13:]
	return p
}
