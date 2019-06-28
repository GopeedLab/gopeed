package tracker

import (
	"fmt"
	"gopeed/down/bt/metainfo"
	"testing"
)

func TestTracker_Tracker(t *testing.T) {
	metaInfo, err := metainfo.ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		panic(err)
	}
	tracker := &Tracker{
		PeerID:   [20]byte{},
		MetaInfo: metaInfo,
	}
	tracker.MetaInfo.Announce = "udp://tracker.opentrackr.org:1337/announce"
	tracker.MetaInfo.AnnounceList = [][]string{}
	peers, err := tracker.Tracker()
	if err != nil {
		panic(err)
		return
	}
	fmt.Println(len(peers))
}
