package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/protocol/bt/metainfo"
	"github.com/monkeyWie/gopeed/protocol/bt/tracker"
	"time"
)

type Torrent struct {
	PeerID   [20]byte
	MetaInfo *metainfo.MetaInfo

	pieces   *pieces
	peerPool *peerPool

	Path string
}

func NewTorrent(peerID [20]byte, metaInfo *metainfo.MetaInfo) *Torrent {
	torrent := &Torrent{
		PeerID:   peerID,
		MetaInfo: metaInfo,
		pieces:   newPiecesState(len(metaInfo.Info.Pieces)),
	}
	return torrent
}

func (t *Torrent) Download(path string) {
	t.Path = path
	t.peerPool = newPeerPool()
	t.fetchPeers()

	taskCh := make(chan interface{}, 128)
	defer close(taskCh)
	for {
		// 获取一个待下载的piece
		index := t.pieces.getReady()
		if index == -1 {
			return
		}
		taskCh <- nil
		go func(index int) {
			peer := t.peerPool.get()
			if peer == nil {
				fmt.Println("no peer")
				time.Sleep(time.Second * 10)
				t.pieces.setState(index, stateReady)
				<-taskCh
				return
			}
			pc := NewPeerConn(t, peer)
			err := pc.ready()
			if err != nil {
				t.pieces.setState(index, stateReady)
				t.peerPool.unavailable(peer)
			} else {
				fmt.Printf("peer ready:%s,download %d\n", peer.Address(), index)
				err = pc.downloadPiece(index)
				if err != nil {
					fmt.Println("down piece error:" + err.Error())
					// 下载失败
					t.pieces.setState(index, stateReady)
					t.peerPool.unavailable(peer)
				} else {
					// 下载完成
					t.pieces.setState(index, stateFinished)
					t.peerPool.release(peer)
				}
			}
			<-taskCh
		}(index)
	}
}

// 获取可用的peer
func (t *Torrent) fetchPeers() {
	tracker := &tracker.Tracker{
		PeerID:   t.PeerID,
		MetaInfo: t.MetaInfo,
	}
	urls := tracker.MetaInfo.AnnounceList[0]
	for _, url := range urls {
		go func(url string) {
			peers, err := tracker.DoTracker(url)
			if err == nil {
				t.peerPool.put(peers)
			}
		}(url)
	}
}
