package torrent

import (
	"gopeed/down/bt/metainfo"
	"gopeed/down/bt/peer"
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

// 检查一个peer能否提供需要下载的文件分片
func CheckPeer(bitfield peer.MsgBitfield) {
}
