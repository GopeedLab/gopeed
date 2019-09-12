package message

import "io"

// piece: <len=0009+X><id=7><index><begin><block>
type Piece struct {
	Message
	Index uint32
	Begin uint32
	Block io.Reader
}

func newPiece(index uint32, begin uint32, reader io.Reader, length uint32) *Piece {
	return &Piece{
		Message: Message{
			Length: 9 + length,
			ID:     IdPiece,
		},
		Index: index,
		Begin: begin,
		Block: io.LimitReader(reader, int64(length)),
	}
}
