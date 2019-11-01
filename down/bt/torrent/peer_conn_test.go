package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"github.com/monkeyWie/gopeed/down/bt/tracker"
	"sync"
	"testing"
	"time"
)

func Test_peerState_ready(t *testing.T) {

	ps := getUsablePeer()
	fmt.Println("get a peer")

	// 下载前准备
	err := ps.ready()
	fmt.Println("ready end")
	if err != nil {
		fmt.Println(ps.peer.Address())
		t.Fatal(err)
	}

}

func Test_peerState_download(t *testing.T) {

	psCh := getUsablePeerMore()

	lock := sync.Mutex{}
	state := map[int]bool{}

	for {
		select {
		case ps := <-psCh:
			go func() {
				// 下载前准备
				err := ps.ready()
				if err != nil {
					fmt.Println("ready error")
					return
				}
				have := ps.getHavePieces(ps.bitfield)
				fmt.Printf("download is ready,have %d\n", len(have))
				if len(have) > 0 {
					// 获取分片的长度
					for _, index := range have {
						lock.Lock()
						if state[index] {
							lock.Unlock()
							continue
						} else {
							state[index] = true
							lock.Unlock()
						}
						fmt.Printf("download index %d\n", index)
						ps.downloadPiece(index)
					}
					return
				}
			}()
		}
	}
}

func Test_peerState_downloadPiece(t *testing.T) {

	ps := getUsablePeer()
	fmt.Println("get a peer")

	// 下载前准备
	err := ps.ready()
	if err != nil {
		t.Fatal(err)
	}
	have := ps.getHavePieces(ps.bitfield)
	fmt.Printf("download is ready,have %d\n", len(have))
	if len(have) > 0 {
		// 获取分片的长度
		for _, index := range have {
			fmt.Printf("download index %d\n", index)
			ps.downloadPiece(index)
		}
		return
	}
}

// 获取一个能连接上的peer
func getUsablePeer() *peerConn {
	ch := getUsablePeerMore()
	ps := <-ch
	return ps
}

// 获取多个能连接上的peer
func getUsablePeerMore() chan *peerConn {
	torrent := buildTorrent()
	tracker := &tracker.Tracker{
		PeerID:   torrent.PeerID,
		MetaInfo: torrent.MetaInfo,
	}
	peers, err := tracker.Tracker()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Tracker end,peer count:%d\n", len(peers))
	ch := make(chan *peerConn)
	for i := range peers {
		go peerTest(torrent, &peers[i], ch)
	}
	return ch
}

func peerTest(torrent *Torrent, peer *peer.Peer, ch chan *peerConn) {
	ps := &peerConn{
		torrent: torrent,
		peer:    peer,
	}
	// 连接至peer
	err := ps.ready()
	if err != nil {
		return
	}
	ps.conn.Close()
	ch <- ps
}

func TestSome(t *testing.T) {
	ch := make(chan interface{})

	go func() {
		time.Sleep(time.Second)
		select {
		case ch <- nil:
			fmt.Println("write")
		case <-time.After(time.Second * 3):
			fmt.Println("timeout")
		}
	}()

	time.Sleep(time.Second * 5)
	r := <-ch
	fmt.Println(r)
	fmt.Println("111")

}
