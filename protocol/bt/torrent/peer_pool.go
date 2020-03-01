package torrent

import (
	"sync"
	"time"

	"github.com/monkeyWie/gopeed/protocol/bt/peer"
	"github.com/monkeyWie/gopeed/protocol/bt/tracker"
)

type peerPool struct {
	torrent *Torrent

	states map[peer.Peer]*peerState
	lock   *sync.Mutex
}

type peerState struct {
	// 是否在被使用
	using bool
	// 下载失败次数
	errors int
}

func newPeerPool(torrent *Torrent) *peerPool {
	return &peerPool{
		torrent: torrent,
		lock:    &sync.Mutex{},
		states:  map[peer.Peer]*peerState{},
	}
}

func (pp *peerPool) fetch() {
	tracker := &tracker.Tracker{
		PeerID:   pp.torrent.PeerID,
		MetaInfo: pp.torrent.MetaInfo,
	}

	go func() {
		for {
			// 当peer数量少于200个时重新发起一次tracker
			if len(pp.states) < 200 {
				go func() {
					for peers := range tracker.Tracker() {
						pp.put(peers)
					}
				}()
			}
			// 每2分钟检测一次
			time.Sleep(time.Minute * 2)
		}
	}()
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

// 将peer重新放回池中，等待使用
func (pp *peerPool) release(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	pp.states[*peer].using = false
	pp.states[*peer].errors = 0
}

// 标记peer为不可用，超过3次则剔除该peer
func (pp *peerPool) unavailable(peer *peer.Peer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	pp.states[*peer].using = false
	pp.states[*peer].errors++
	if pp.states[*peer].errors > 3 {
		delete(pp.states, *peer)
	}

}
