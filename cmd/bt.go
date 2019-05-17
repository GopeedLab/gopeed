package main

import (
	"gopeed/down/bt"
)

func main() {
	metaInfo, _ := bt.ParseFromFile("e:/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	// metaInfo, _ := down.ParseFromFile("e:/test.torrent")
	if len(metaInfo.AnnounceList) > 0 {
		for _, announceArr := range metaInfo.AnnounceList {
			if len(announceArr) > 0 {
				for _, announce := range announceArr {
					bt.DoTracker(announce, metaInfo)
				}
			}
		}
	} else {
		bt.DoTracker(metaInfo.Announce, metaInfo)
	}
}
