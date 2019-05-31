package bt

import (
	"fmt"
)

type Torrent struct {
	client *Client

	MetaInfo *MetaInfo
	Peers    []Peer
}

func (torrent *Torrent) Download() {
	t := NewTracker(torrent)
	peers, err := t.Tracker()
	if err != nil {
		fmt.Println(err)
		return
	}

	peers[0].Torrent = torrent
	peers[0].DoDownload()
}
