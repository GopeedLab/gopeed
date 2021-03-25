package message

// 	keep-alive: <len=0000>
type Keepalive struct {
	*base
}

var data = make([]byte, 4)

func (k *Keepalive) Encode() []byte {
	return data
}

func (k *Keepalive) Decode(body []byte) {

}
