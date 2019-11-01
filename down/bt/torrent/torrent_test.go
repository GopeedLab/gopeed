package torrent

import (
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"testing"
)

func TestTorrent_Download(t *testing.T) {
	torrent := buildTorrent()
	torrent.Download("e:/testbt/download")
}

func buildTorrent() *Torrent {
	metaInfo, err := metainfo.ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		panic(err)
	}
	// metaInfo.Announce = "udp://exodus.desync.com:6969/announce"
	// metaInfo.Announce = "udp://tracker.openbittorrent.com:80/announce"
	// metaInfo.Announce = "udp://tracker.cyberia.is:6969/announce"
	// metaInfo.Announce = "udp://9.rarbg.me:2780/announce"
	// metaInfo.Announce = "udp://9.rarbg.to:2760/announce"
	// metaInfo.Announce = "udp://tracker.leechers-paradise.org:6969"
	metaInfo.AnnounceList = [][]string{}
	metaInfo.AnnounceList = append(metaInfo.AnnounceList, []string{
		"udp://exodus.desync.com:6969/announce",
		"udp://tracker.openbittorrent.com:80/announce",
		"udp://tracker.leechers-paradise.org:6969",
	})
	return NewTorrent(peer.GenPeerID(), metaInfo)
}
