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

	pieceStates *pieceStates
	peerPool    *peerPool
	Path        string
}

func NewTorrent(peerID [20]byte, metaInfo *metainfo.MetaInfo) *Torrent {
	torrent := &Torrent{
		PeerID:      peerID,
		MetaInfo:    metaInfo,
		pieceStates: newPiecesState(metaInfo),
	}
	return torrent
}

func (t *Torrent) Download(path string) {
	t.Path = path
	t.peerPool = newPeerPool(t)
	t.peerPool.fetch()

	pieceQueueCh := make(chan interface{}, 128)
	defer close(pieceQueueCh)
	downloadedCh := make(chan interface{})
	defer close(downloadedCh)

	for i := 0; i < len(t.MetaInfo.Info.Pieces); i++ {
		// 加入下载队列
		pieceQueueCh <- nil
		go func(index int) {
			t.pieceStates.setState(index, stateDownloading)
			// 一直下载到成功
			for {
				peer := t.peerPool.get()
				if peer == nil {
					time.Sleep(time.Second * 10)
					continue
				}
				pc := newPeerConn(t, peer)
				err := pc.ready()
				if err != nil {
					t.peerPool.unavailable(peer)
				} else {
					err = pc.downloadPiece(index)
					if err != nil {
						log.Debugf("piece %d download fail:%s", index, err.Error())
						// 下载失败
						t.peerPool.unavailable(peer)
						if errors.Is(err, errPieceCheckFailed) {
							// piece校验失败，需要重新下载过一遍，清空block记录
							t.pieceStates.clearBlocks(index)
						}
						continue
					} else {
						// 下载完成
						t.pieceStates.setState(index, stateFinish)
						t.peerPool.release(peer)
						log.Debugf("piece %d download success,total:%d,left:%d", index, t.pieceStates.size(), t.pieceStates.getLeft())
						if t.pieceStates.isDone() {
							downloadedCh <- nil
						}
						break
					}
				}
			}
			<-pieceQueueCh
		}(i)
	}
	<-downloadedCh
	log.Debug("download end")
}
