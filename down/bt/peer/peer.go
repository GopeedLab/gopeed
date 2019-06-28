package peer

import (
	"encoding/binary"
	"math/rand"
	"net"
	"strconv"
)

type Peer struct {
	IP   uint32
	Port uint16
}

func (peer *Peer) Address() string {
	bts := make([]byte, 4)
	binary.BigEndian.PutUint32(bts, peer.IP)
	return net.IP(bts).String() + ":" + strconv.Itoa(int(peer.Port))
}

// 生成Peer ID，规则为前三位固定字母(-GP)+SemVer(xyz)+End(-),后面随机生成
// 参考：https://wiki.theory.org/index.php/BitTorrentSpecification#peer_id
func GenPeerID() [20]byte {
	peerID := [20]byte{'-', 'G', 'P', '0', '0', '1', '-'}
	_, err := rand.Read(peerID[7:])
	if err != nil {
		panic(err)
	}
	return peerID
}
