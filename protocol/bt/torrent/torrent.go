package torrent

import (
	"errors"
	"time"

	"github.com/monkeyWie/gopeed/protocol/bt/metainfo"
	log "github.com/sirupsen/logrus"
)

type Torrent struct {
	PeerID   [20]byte
	MetaInfo *metainfo.MetaInfo

	pieces     *pieces
	peerPool   *peerPool
	completeCh chan interface{}

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

	pieceQueueCh := make(chan interface{}, 256)
	defer close(pieceQueueCh)
	t.completeCh = make(chan interface{})
	defer close(t.completeCh)
	for {
		pieceQueueCh <- nil
		// 获取一个待下载的piece
		index := t.pieces.getReady()
		// 没有待下载的piece了
		if index == -1 {
			log.Debugf("no piece to download")
			<-t.completeCh
			// 下载完成
			break
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
			err := pc.ready()
			if err != nil {
				t.pieces.setState(index, stateReady)
				t.peerPool.unavailable(peer)
			} else {
				err = pc.downloadPiece(index)
				if err != nil {
					log.Debugf("piece %d download fail:%s", index, err.Error())
					// 下载失败
					t.pieces.setState(index, stateReady)
					t.peerPool.unavailable(peer)
					if errors.Is(err, errPieceCheckFailed) {
						// piece校验失败，需要重新下载过一遍，清空block记录
						t.pieces.clearBlocks(index)
					}
				} else {
					// 下载完成
					t.pieces.setState(index, stateFinished)
					t.peerPool.release(peer)
					log.Debugf("piece %d download success,total:%d,left:%d", index, t.pieces.size(), t.pieces.getLeft())
					// 检查是否所有piece下载完成
					if t.pieces.isDone() {
						t.completeCh <- nil
					}
				}
			}
			<-pieceQueueCh
		}(index)
	}
}
