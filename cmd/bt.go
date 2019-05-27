package main

import (
	"gopeed/down/bt"
)

func main() {
	metaInfo, _ := bt.ParseFromFile("e:/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	metaInfo.Tracker()
}
