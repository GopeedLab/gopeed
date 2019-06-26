package bt

import (
	"fmt"
	"io"
)

type Torrent struct {
	client *Client

	MetaInfo *MetaInfo
	Peers    []Peer
}

// Handshake of Peer wire protocol
// Per https://wiki.theory.org/index.php/BitTorrentSpecification#Handshake
func (torrent *Torrent) doHandshake(peer *Peer) error {
	conn, err := dialMse(peer.Address(), peer.Torrent.MetaInfo.InfoHash)
	if err != nil {
		return err
	}
	defer conn.Close()

	metaInfo := peer.Torrent.MetaInfo
	peerID := peer.Torrent.client.PeerID

	reserved := [8]byte{}
	/*	reserved[5] = 0x10
		reserved[6] = 0x0
		reserved[7] = 0x5*/
	handshakeReq := newHandshake(reserved, metaInfo.InfoHash, peerID)
	buf, err := handshakeReq.encode()
	if err != nil {
		return err
	}
	conn.Write(buf)

	var read [68]byte
	_, err = io.ReadFull(conn, read[:])
	if err != nil {
		return err
	}
	handshakeRes := new(Handshake)
	err = handshakeRes.decode(read[:])
	if err != nil {
		return err
	}
	if handshakeRes.InfoHash != handshakeReq.InfoHash {
		return err
	}
	return nil
}

func (torrent *Torrent) Download() {
	err := torrent.Tracker()
	if err != nil {
		fmt.Println(err)
		return
	}
}
