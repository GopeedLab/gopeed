package message

// 	keep-alive: <len=0000>
type Keepalive struct{}

func (k *Keepalive) Encode() []byte {
	return []byte{0, 0, 0, 0}
}

func (k *Keepalive) Decode(buf []byte) Serialize {
	return &Keepalive{}
}
