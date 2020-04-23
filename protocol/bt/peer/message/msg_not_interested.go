package message

// choke: <len=0001><id=3>
type NotInterested struct {
	*base
}

func NewNotInterested() *NotInterested {
	return &NotInterested{
		base: &base{
			id: IdNotInterested,
		},
	}
}
