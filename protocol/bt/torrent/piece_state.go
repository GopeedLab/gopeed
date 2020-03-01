package torrent

import (
	"sync"

	"github.com/RoaringBitmap/roaring"
)

type state int

const (
	stateReady = iota
	stateDownloading
	stateFinished
)

// 所有piece的下载状态
type pieces struct {
	rw  *sync.RWMutex
	arr []piece
}

type piece struct {
	// piece的下载状态
	state state
	// piece中block的下载状态
	blocks *roaring.Bitmap
}

func newPiecesState(size int) *pieces {
	ps := &pieces{
		rw:  &sync.RWMutex{},
		arr: make([]piece, size),
	}
	for i := 0; i < size; i++ {
		ps.arr[i].state = stateReady
		ps.arr[i].blocks = roaring.NewBitmap()
	}
	return ps
}

func (ps *pieces) getState(index int) state {
	ps.rw.RLock()
	defer ps.rw.RUnlock()
	return ps.arr[index].state
}

func (ps *pieces) setState(index int, state state) {
	ps.rw.Lock()
	defer ps.rw.Unlock()
	ps.arr[index].state = state
}

func (ps *pieces) getReady() int {
	ps.rw.Lock()
	defer ps.rw.Unlock()
	for i, s := range ps.arr {
		if s.state == stateReady {
			ps.arr[i].state = stateDownloading
			return i
		}
	}
	return -1
}

func (ps *pieces) getLeft() int {
	ps.rw.RLock()
	defer ps.rw.RUnlock()
	count := 0
	for _, s := range ps.arr {
		if s.state != stateFinished {
			count++
		}
	}
	return count
}

func (ps *pieces) size() int {
	return len(ps.arr)
}

func (ps *pieces) isDone() bool {
	ps.rw.RLock()
	defer ps.rw.RUnlock()
	for _, s := range ps.arr {
		if s.state != stateFinished {
			return false
		}
	}
	return true
}

// 指定piece下的某个block是否下载完成
func (ps *pieces) isBlockDownloaded(pieceIndex int, blockIndex int) bool {
	return ps.arr[pieceIndex].blocks.ContainsInt(blockIndex)
}

// 指定piece下的某个block下载完成
func (ps *pieces) setBlockDownloaded(pieceIndex int, blockIndex int) {
	ps.arr[pieceIndex].blocks.AddInt(blockIndex)
}

// 清空指定piece下所有的block下载记录
func (ps *pieces) clearBlocks(index int) {
	ps.arr[index].blocks.Clear()
}

func (ps *pieces) blockSize(index int) int {
	return int(ps.arr[index].blocks.GetCardinality())
}
