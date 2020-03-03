package torrent

import (
	"github.com/monkeyWie/gopeed/protocol/bt/metainfo"
	"math"
	"sync"

	"github.com/RoaringBitmap/roaring"
)

type state int

const (
	stateReady = iota
	stateDownloading
	stateFinish
)

// 所有piece的下载状态
type pieceStates struct {
	lock *sync.RWMutex
	// 所有piece的block下载进度
	states map[int]*pieceState
}

type pieceState struct {
	state state
	// piece中block的个数
	blockCount int
	// piece中block的下载记录
	blockRecord *roaring.Bitmap
}

func newPiecesState(meta *metainfo.MetaInfo) *pieceStates {
	pieceCount := len(meta.Info.Pieces)
	ps := &pieceStates{
		lock:   &sync.RWMutex{},
		states: make(map[int]*pieceState, pieceCount),
	}
	for i := 0; i < pieceCount; i++ {
		ps.states[i] = &pieceState{
			state:       stateReady,
			blockCount:  int(math.Ceil(float64(meta.GetPieceLength(i)) / blockSize)),
			blockRecord: roaring.New(),
		}
	}
	return ps
}

func (ps *pieceStates) getState(index int) state {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.states[index].state
}

func (ps *pieceStates) setState(index int, state state) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.states[index].state = state
}

func (ps *pieceStates) size() int {
	return len(ps.states)
}

func (ps *pieceStates) getLeft() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	count := 0
	for _, v := range ps.states {
		if v.state == stateReady || v.state == stateDownloading {
			count++
		}
	}
	return count
}

func (ps *pieceStates) isDone() bool {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	for _, v := range ps.states {
		if v.state == stateReady || v.state == stateDownloading {
			return false
		}
	}
	return true
}

func (ps *pieceStates) isPieceDone(index int) bool {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	state := ps.states[index]
	return uint64(state.blockCount) == state.blockRecord.GetCardinality()
}

// 指定piece下的某个block是否下载完成
func (ps *pieceStates) isBlockDone(pieceIndex int, blockIndex int) bool {
	return ps.states[pieceIndex].blockRecord.ContainsInt(blockIndex)
}

// 指定piece下的某个block下载完成
func (ps *pieceStates) setBlockDone(pieceIndex int, blockIndex int) {
	ps.states[pieceIndex].blockRecord.AddInt(blockIndex)
}

// 清空指定piece下所有的block下载记录
func (ps *pieceStates) clearBlocks(index int) {
	ps.states[index].blockRecord.Clear()
}
