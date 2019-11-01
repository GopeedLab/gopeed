package torrent

import (
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"sync"
)

type peerPool struct {
	peers []peerState
	lock  *sync.Mutex
}

type peerState struct {
	peer  *peer.Peer
	using bool
	ready bool
}

func newPeerPool() *peerPool {
	return &peerPool{
		lock: &sync.Mutex{},
	}
}

func (pp *peerPool) put(peers []peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	for i := range peers {
		pp.peers = append(pp.peers, peerState{
			peer:  &peers[i],
			using: false,
		})
	}
}

func (pp *peerPool) get() *peer.Peer {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	for i, p := range pp.peers {
		if !p.using {
			pp.peers[i].using = true
			return pp.peers[i].peer
		}
	}
	return nil
}

func (pp *peerPool) ready(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	i := findPeer(pp, peer)
	if i != -1 {
		pp.peers[i].ready = true
	}
}

func (pp *peerPool) release(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	i := findPeer(pp, peer)
	if i != -1 {
		pp.peers[i].using = false
	}
}

func (pp *peerPool) remove(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	i := findPeer(pp, peer)
	if i != -1 {
		pp.peers = append(pp.peers[0:i], pp.peers[i+1:]...)
	}
}

func findPeer(pp *peerPool, peer *peer.Peer) int {
	for i := range pp.peers {
		if pp.peers[i].peer == peer {
			return i
		}
	}
	return -1
}
