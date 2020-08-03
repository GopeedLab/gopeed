package dht

import (
	"github.com/marksamman/bencode"
	"net"
)

func ping() string {
	data := map[string]interface{}{
		"t": "aa",
		"y": "q",
		"q": "ping",
		"a": map[string]interface{}{"id": "abcdefghij0123456789"},
	}
	req := bencode.Encode(data)
	// session.add_dht_router("router.utorrent.com", 6881)
	// session.add_dht_router("router.bittorrent.com", 6881)
	// session.add_dht_router("dht.transmissionbt.com", 6881)
	// session.add_dht_router("router.bitcomet.com", 6881)
	// session.add_dht_router("dht.aelitis.com", 6881)
	dial, err := net.Dial("udp", "router.utorrent.com:6881")
	if err != nil {
		panic(err)
	}
	dial.Write(req)

	reps := make([]byte, 65535)

	n, err := dial.Read(reps)
	if err != nil {
		panic(err)
	}
	return string(reps[:n])
}
