package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/protocol/bt/peer"
	"github.com/monkeyWie/gopeed/protocol/bt/tracker"
	"os"
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
			ps.downloadPiece(int(index))
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
	peers := <-tracker.Tracker()
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

func TestSome2(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(10)
	file, err := os.OpenFile("e:/testbt/test.data", os.O_RDWR|os.O_CREATE, 0644)
	for i := 0; i < 10; i++ {
		go func(i int) {
			for j := 0; j < 100; j++ {
				if err != nil {
					panic(err)
				}
				_, err = file.WriteAt([]byte{byte(i)}, int64(i*8))
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}
