package torrent

import (
	"fmt"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"testing"
)

func Test_PeerPool(t *testing.T) {
	pool := newPeerPool()
	pool.put([]peer.Peer{
		{
			IP:   0,
			Port: 0,
		},
		{
			IP:   1,
			Port: 0,
		},
		{
			IP:   2,
			Port: 0,
		},
		{
			IP:   3,
			Port: 0,
		},
		{
			IP:   4,
			Port: 0,
		},
	})
	get := pool.get()
	pool.release(get)
	pool.remove(get)
	for i := 0; i < 5; i++ {
		fmt.Println(pool.get())
	}
	fmt.Println(pool.get())
}
