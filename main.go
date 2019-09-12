package main

import (
	"fmt"
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"github.com/monkeyWie/gopeed/down/bt/torrent"
	"github.com/monkeyWie/gopeed/down/bt/tracker"
	"github.com/monkeyWie/gopeed/down/http"
)

func main() {
	request := &http.Request{
		Method: "get",
		URL:    "http://github.com/proxyee-down-org/proxyee-down/releases/download/3.4/proxyee-down-main.jar",
		Header: map[string]string{
			"Host":            "github.com",
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
			"Referer":         "http://github.com/proxyee-down-org/proxyee-down/releases",
			"Accept-Encoding": "gzip, deflate, br",
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		},
	}
	got, _ := http.Resolve(request)
	fmt.Println(got)
	getUsablePeer()
	// webview.Open("Minimal webview example", "https://www.baidu.com", 800, 600, true)
}

func buildTorrent() *torrent.Torrent {
	metaInfo, err := metainfo.ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		panic(err)
	}
	metaInfo.Announce = "udp://tracker.opentrackr.org:1337/announce"
	metaInfo.AnnounceList = [][]string{}
	return torrent.NewTorrent(peer.GenPeerID(), metaInfo)
}

// 获取一个能连接上的peer
func getUsablePeer() {
	torrent := buildTorrent()
	tracker := &tracker.Tracker{
		PeerID:   torrent.PeerID,
		MetaInfo: torrent.MetaInfo,
	}
	tracker.Tracker()
}
