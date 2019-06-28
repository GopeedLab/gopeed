package tracker

import (
	"fmt"
)

type ErrorCode int

const (
	ErrTimeout  ErrorCode = -1
	ErrResponse ErrorCode = -2
)

type BtError interface {
	error
}

type TrackerError struct {
	Code ErrorCode
	Err  error
}

func (te *TrackerError) Error() string {
	return fmt.Sprintf("%s:%d", te.Err.Error(), te.Code)
}

func NewTrackerError(code ErrorCode, err error) *TrackerError {
	return &TrackerError{code, err}
}
