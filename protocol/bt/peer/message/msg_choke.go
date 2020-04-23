package message

// choke: <len=0001><id=0>
type Choke struct {
	*base
}

func NewChoke() *Choke {
	return &Choke{
		base: &base{
			id: IdChoke,
		},
	}
}
