package message

// choke: <len=0001><id=2>
type Interested struct {
	*base
}

func NewInterested() *Interested {
	return &Interested{
		base: &base{
			id: IdInterested,
		},
	}
}
