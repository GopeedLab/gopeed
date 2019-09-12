package message

// 	unchoke: <len=0001><id=1>
type Unchoke struct {
	Message
}

func NewUnchoke() *Unchoke {
	return &Unchoke{
		Message{
			Length: 1,
			ID:     IdUnchoke,
		},
	}
}

func (u *Unchoke) Encode() []byte {
	return u.Message.Encode()
}

func (u *Unchoke) Decode(buf []byte) Serialize {
	u.Message.Decode(buf)
	return u
}
