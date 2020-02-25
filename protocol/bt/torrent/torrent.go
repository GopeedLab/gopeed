package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/protocol/bt/metainfo"
	"github.com/monkeyWie/gopeed/protocol/bt/tracker"
	"sync"
	"time"
)

type Torrent struct {
	PeerID   [20]byte
	MetaInfo *metainfo.MetaInfo

	PiecesState *piecesState
	peerPool    *peerPool

	Path string
}

func NewTorrent(peerID [20]byte, metaInfo *metainfo.MetaInfo) *Torrent {
	torrent := &Torrent{
		PeerID:      peerID,
		MetaInfo:    metaInfo,
		PiecesState: NewPiecesState(len(metaInfo.Info.Pieces)),
	}
	return torrent
}

func (t *Torrent) Download(path string) {
	t.Path = path
	t.peerPool = newPeerPool()
	t.fetchPeers()

	taskCh := make(chan interface{}, 16)
	defer close(taskCh)
	for {
		index := t.PiecesState.getReadyAndDownload()
		if index == -1 {
			return
		}
		taskCh <- nil
		go func(index int) {
			peer := t.peerPool.get()
			if peer == nil {
				time.Sleep(time.Second * 10)
				t.PiecesState.setState(index, stateReady)
				<-taskCh
				return
			}
			pc := NewPeerConn(t, peer)
			err := pc.ready()
			if err != nil {
				t.PiecesState.setState(index, stateReady)
				t.peerPool.remove(peer)
			} else {
				fmt.Printf("peer ready:%s,download %d\n", peer.Address(), index)
				err = pc.downloadPiece(index)
				if err != nil {
					fmt.Println(err)
					// 下载失败
					t.PiecesState.setState(index, stateReady)
				} else {
					// 下载完成
					t.PiecesState.setState(index, stateFinished)
				}
				t.peerPool.release(peer)
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
	var wg sync.WaitGroup
	ch := make(chan interface{})
	wg.Add(len(urls))
	for _, url := range urls {
		go func(url string) {
			defer wg.Done()
			peers, err := tracker.DoTracker(url)
			if err == nil {
				fmt.Printf("Tracker end,url:%s,count:%d\n", url, len(peers))
				ch <- nil
				t.peerPool.put(peers)
			}
		}(url)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	<-ch
}
