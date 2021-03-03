package http

import "github.com/monkeyWie/gopeed-core/download/base"

type Event struct {
	status base.Status
}

func (e *Event) Status() base.Status {
	return e.status
}
