package torrent

import (
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
)

type Torrent struct {
	PeerID     [20]byte
	MetaInfo   *metainfo.MetaInfo
	PieceState []pieceState

	Peers []peerState
}

func NewTorrent(peerID [20]byte, metaInfo *metainfo.MetaInfo) *Torrent {
	torrent := &Torrent{
		PeerID:     peerID,
		MetaInfo:   metaInfo,
		PieceState: make([]pieceState, len(metaInfo.Info.Pieces)),
	}
	return torrent
}
