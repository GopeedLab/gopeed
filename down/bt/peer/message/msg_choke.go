package message

// choke: <len=0001><id=0>
type Choke struct {
	Message
}

func NewChoke() *Choke {
	return &Choke{
		Message{
			Length: 1,
			ID:     IdChoke,
		},
	}
}

func (c *Choke) Encode() []byte {
	return c.Message.Encode()
}

func (c *Choke) Decode(buf []byte) Serialize {
	c.Message.Decode(buf)
	return c
}
