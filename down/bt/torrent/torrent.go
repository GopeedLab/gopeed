package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/tracker"
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
				fmt.Println("no peer")
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
					t.PiecesState.setState(index, stateDownloading)
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
	urls := []string{
		"udp://exodus.desync.com:6969/announce",
		"udp://tracker.openbittorrent.com:80/announce",
		"udp://tracker.leechers-paradise.org:6969",
	}
	var wg sync.WaitGroup
	ch := make(chan interface{})
	wg.Add(len(urls))
	for _, url := range urls {
		go func(url string) {
			defer wg.Done()
			peers, err := tracker.DoTracker(url)
			if err == nil {
				fmt.Println("tracker end:" + url)
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
