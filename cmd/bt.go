package main

import (
	"gopeed/down/bt/metainfo"
)

func main() {
	metaInfo, _ := metainfo.ParseFromFile("e:/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	metaInfo.Tracker()
}
