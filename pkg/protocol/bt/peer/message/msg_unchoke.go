package message

// 	unchoke: <len=0001><id=1>
type Unchoke struct {
	*base
}

func NewUnchoke() *Unchoke {
	return &Unchoke{
		base: &base{
			id: IdUnchoke,
		},
	}
}
