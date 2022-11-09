package base

import "errors"

var (
	NotFound  = errors.New("not found")
	BadParams = errors.New("bad params")
)
