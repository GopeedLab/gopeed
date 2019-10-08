package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"github.com/monkeyWie/gopeed/down/bt/tracker"
	"testing"
)

func Test_peerState_handshake(t *testing.T) {

	ps := getUsablePeer()

	// 发出握手请求
	handshake, err := ps.handshake()
	fmt.Println("Handshake end")
	if err != nil {
		fmt.Println(ps.Address())
		panic(err)
	}
	fmt.Println(handshake)
}

func Test_peerState_download(t *testing.T) {

	ps := getUsablePeer()

	// 发出握手请求
	_, err := ps.handshake()
	if err != nil {
		panic(err)
	}
	fmt.Println("handshake end")

	// 开始下载
	err = ps.download()

}

func buildTorrent() *Torrent {
	metaInfo, err := metainfo.ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		panic(err)
	}
	metaInfo.Announce = "udp://exodus.desync.com:6969/announce"
	metaInfo.AnnounceList = [][]string{}
	return NewTorrent(peer.GenPeerID(), metaInfo)
}

// 获取一个能连接上的peer
func getUsablePeer() *peerState {
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

func TestSome(t *testing.T) {
	fmt.Println(11)
}
