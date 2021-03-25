package util

import "time"

// 计时器
type Timer struct {
	t    int64
	used int64
}

func (t *Timer) Start() {
	t.t = time.Now().UnixNano()
}

func (t *Timer) Pause() {
	t.used += time.Now().UnixNano() - t.t
}

func (t *Timer) Continue() {
	t.t = time.Now().UnixNano()
}

func (t *Timer) Used() int64 {
	return (time.Now().UnixNano() - t.t) + t.used
}
