package torrent

import (
	"github.com/monkeyWie/gopeed/protocol/bt/peer"
	"sync"
)

type peerPool struct {
	states map[peer.Peer]*peerState
	lock   *sync.Mutex
}

type peerState struct {
	// 是否在被使用
	using bool
	// 下载失败次数
	errors int
}

func newPeerPool() *peerPool {
	return &peerPool{
		lock:   &sync.Mutex{},
		states: map[peer.Peer]*peerState{},
	}
}

func (pp *peerPool) put(peers []peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	for _, peer := range peers {
		if _, ok := pp.states[peer]; !ok {
			pp.states[peer] = &peerState{
				using:  false,
				errors: 0,
			}
		}

	}
}

func (pp *peerPool) get() *peer.Peer {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	for peer, state := range pp.states {
		if !state.using {
			state.using = true
			return &peer
		}
	}
	return nil
}

func (pp *peerPool) release(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	pp.states[*peer].using = false
	pp.states[*peer].errors = 0
}

func (pp *peerPool) unavailable(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	pp.states[*peer].errors++
	if pp.states[*peer].errors > 3 {
		delete(pp.states, *peer)
	}

}
