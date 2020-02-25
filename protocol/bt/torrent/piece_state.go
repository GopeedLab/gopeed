package torrent

import "sync"

type state int

const (
	stateReady = iota
	stateDownloading
	stateFinished
)

// 所有piece的下载状态
type piecesState struct {
	rw     *sync.RWMutex
	states []state
}

func NewPiecesState(size int) *piecesState {
	ps := &piecesState{
		rw:     &sync.RWMutex{},
		states: make([]state, size),
	}
	for i := 0; i < size; i++ {
		ps.states[i] = stateReady
	}
	return ps
}

func (ps *piecesState) getState(index int) state {
	ps.rw.RLock()
	defer ps.rw.RUnlock()
	return ps.states[index]
}

func (ps *piecesState) setState(index int, state state) {
	ps.rw.Lock()
	defer ps.rw.Unlock()
	ps.states[index] = state
}

func (ps *piecesState) getReadyAndDownload() int {
	ps.rw.Lock()
	defer ps.rw.Unlock()
	for i, s := range ps.states {
		if s == stateReady {
			ps.states[i] = stateDownloading
			return i
		}
	}
	return -1
}

func (ps *piecesState) Size() int {
	return len(ps.states)
}
