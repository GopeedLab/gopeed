package http

import "github.com/monkeyWie/gopeed-core/download/common"

type Event struct {
	status common.Status
}

func (e *Event) Status() common.Status {
	return e.status
}
