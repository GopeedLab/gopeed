package message

import (
	"encoding/binary"
)

// request: <len=0013><id=6><index><begin><length>
type Request struct {
	Message
	Index  uint32
	Begin  uint32
	Length uint32
}

func NewRequest(index uint32, begin uint32, length uint32) *Request {
	return &Request{
		Message: Message{
			Length: 13,
			ID:     IdRequest,
		},
		Index:  index,
		Begin:  begin,
		Length: length,
	}
}

func (r *Request) Encode() []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], r.Index)
	binary.BigEndian.PutUint32(buf[4:8], r.Begin)
	binary.BigEndian.PutUint32(buf[8:12], r.Length)
	return append(r.Message.Encode(), buf...)
}

func (r *Request) Decode(buf []byte) Serialize {
	r.Message.Decode(buf)
	r.Length = binary.BigEndian.Uint32(buf[5:9])
	r.Begin = binary.BigEndian.Uint32(buf[9:13])
	r.Index = binary.BigEndian.Uint32(buf[13:17])
	return r
}
