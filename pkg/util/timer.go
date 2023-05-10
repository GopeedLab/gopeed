package util

import "time"

type Timer struct {
	t    int64
	used int64
}

func NewTimer(used int64) *Timer {
	return &Timer{
		used: used,
	}
}

func (t *Timer) Start() {
	t.t = time.Now().UnixNano()
}

func (t *Timer) Pause() {
	t.used += time.Now().UnixNano() - t.t
}

func (t *Timer) Used() int64 {
	return (time.Now().UnixNano() - t.t) + t.used
}
