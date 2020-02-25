package message

// choke: <len=0001><id=0>
type NotInterested struct {
	Message
}

func NewNotInterested() *NotInterested {
	return &NotInterested{
		Message{
			Length: 1,
			ID:     IdNotinterested,
		},
	}
}

func (ni *NotInterested) Encode() []byte {
	return ni.Message.Encode()
}

func (ni *NotInterested) Decode(buf []byte) Serialize {
	ni.Message.Decode(buf)
	return ni
}
