package torrent

import (
	"fmt"
	"gopeed/down/bt/metainfo"
	"gopeed/down/bt/peer"
	"gopeed/down/bt/tracker"
	"testing"
)

func Test_peerState_handshake(t *testing.T) {
	torrent := buildTorrent()
	tracker := &tracker.Tracker{
		PeerID:   torrent.PeerID,
		MetaInfo: torrent.MetaInfo,
	}
	peers, err := tracker.Tracker()
	if err != nil {
		panic(err)
	}
	fmt.Println("Tracker end")
	ps := getUsablePeer(torrent, peers)

	// 发出握手请求
	handshake, err := ps.handshake()
	fmt.Println("Handshake end")
	if err != nil {
		fmt.Println(ps.Address())
		panic(err)
	}
	fmt.Println(handshake)
}

func buildTorrent() *Torrent {
	metaInfo, err := metainfo.ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		panic(err)
	}
	metaInfo.Announce = "udp://tracker.opentrackr.org:1337/announce"
	metaInfo.AnnounceList = [][]string{}
	return NewTorrent(peer.GenPeerID(), metaInfo)
}

func getUsablePeer(torrent *Torrent, peers []peer.Peer) *peerState {
	ch := make(chan *peerState, 1)
	for i := range peers {
		go peerTest(torrent, &peers[i], ch)
	}
	return <-ch
}

func peerTest(torrent *Torrent, peer *peer.Peer, ch chan *peerState) {
	ps := &peerState{
		torrent: torrent,
		Peer:    peer,
	}
	// 连接至peer
	err := ps.dialMse()
	if err != nil {
		return
	}
	ch <- ps
}
