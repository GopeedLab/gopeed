package message

import "encoding/binary"

// piece: <len=0009+X><id=7><index><begin><block>
type Piece struct {
	*base
	Index uint32
	Begin uint32
	Block []byte
}

func NewPiece() *Piece {
	return &Piece{
		base: &base{
			id: IdPiece,
		},
	}
}

func BuildPiece(index uint32, begin uint32, block []byte) *Piece {
	piece := NewPiece()
	piece.Index = index
	piece.Begin = begin
	piece.Block = block
	return piece

}

func (p *Piece) Encode() []byte {
	buf := make([]byte, 8+len(p.Block))
	binary.BigEndian.PutUint32(buf[0:4], p.Index)
	binary.BigEndian.PutUint32(buf[4:8], p.Begin)
	copy(buf[8:], p.Block)
	return encode(p.base, buf)
}

func (p *Piece) Decode(buf []byte) {
	p.Index = binary.BigEndian.Uint32(buf[0:4])
	p.Begin = binary.BigEndian.Uint32(buf[4:8])
	p.Block = buf[8:]
}
