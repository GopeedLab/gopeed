package message

import (
	"encoding/binary"
)

// request: <len=0013><id=6><index><begin><length>
type Request struct {
	*base
	Index  uint32
	Begin  uint32
	Length uint32
}

func NewRequest() *Request {
	return &Request{
		base: &base{
			id: IdRequest,
		},
	}
}

func BuildRequest(index uint32, begin uint32, length uint32) *Request {
	request := NewRequest()
	request.Index = index
	request.Begin = begin
	request.Length = length
	return request
}

func (r *Request) Encode() []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], r.Index)
	binary.BigEndian.PutUint32(buf[4:8], r.Begin)
	binary.BigEndian.PutUint32(buf[8:12], r.Length)
	return encode(r.base, buf)
}

func (r *Request) Decode(buf []byte) {
	r.Length = binary.BigEndian.Uint32(buf[0:4])
	r.Begin = binary.BigEndian.Uint32(buf[4:8])
	r.Index = binary.BigEndian.Uint32(buf[8:12])
}
