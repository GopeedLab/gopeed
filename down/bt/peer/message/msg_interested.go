package message

// choke: <len=0001><id=0>
type Interested struct {
	Message
}

func NewInterested() *Interested {
	return &Interested{
		Message{
			Length: 1,
			ID:     IdInterested,
		},
	}
}

func (c *Interested) Encode() []byte {
	return c.Message.Encode()
}

func (c *Interested) Decode(buf []byte) Serialize {
	c.Message.Decode(buf)
	return c
}
