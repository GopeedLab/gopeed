package torrent

import (
	"errors"
	"github.com/monkeyWie/gopeed/protocol/bt/metainfo"
	log "github.com/sirupsen/logrus"
	"time"
)

type Torrent struct {
	PeerID   [20]byte
	MetaInfo *metainfo.MetaInfo

	pieces     *pieces
	peerPool   *peerPool
	completeCh chan bool

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
	t.peerPool = newPeerPool(t)
	t.peerPool.fetch()

	pieceQueueCh := make(chan interface{}, 128)
	defer close(pieceQueueCh)
	t.completeCh = make(chan bool)
	defer close(t.completeCh)
	for {
		pieceQueueCh <- nil
		// 获取一个待下载的piece
		index := t.pieces.getReady()
		// 没有待下载的piece了
		if index == -1 {
			if <-t.completeCh {
				// 下载完成
				break
			} else {
				// 继续下载
				continue
			}
		}
		go func(index int) {
			peer := t.peerPool.get()
			if peer == nil {
				time.Sleep(time.Second * 10)
				t.pieces.setState(index, stateReady)
				<-pieceQueueCh
				return
			}
			pc := newPeerConn(t, peer)
			log.Debugf("peer ready start:%s,download %d", peer.Address(), index)
			err := pc.ready()
			if err != nil {
				log.Debugf("peer ready error:%s,download %d,error:%s", peer.Address(), index, err.Error())
				t.pieces.setState(index, stateReady)
				t.peerPool.unavailable(peer)
			} else {
				log.Debugf("peer ready success:%s,download %d", peer.Address(), index)
				err = pc.downloadPiece(index)
				if err != nil {
					// 下载失败
					t.pieces.setState(index, stateReady)
					t.peerPool.unavailable(peer)
					if errors.Is(err, errPieceCheckFailed) {
						// piece校验失败，需要重新下载过一遍，清空block记录
						t.pieces.clearBlocks(index)
					}
					t.completeCh <- false
				} else {
					// 下载完成
					t.pieces.setState(index, stateFinished)
					t.peerPool.release(peer)
					// 检查是否所有piece下载完成
					if t.pieces.isDone() {
						t.completeCh <- true
					}
				}
			}
			<-pieceQueueCh
		}(index)
	}
}
