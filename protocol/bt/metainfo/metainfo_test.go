package metainfo

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestParseFromFile(t *testing.T) {
	// metaInfo, err := ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	metaInfo, err := ParseFromFile("E:\\bt\\win10.torrent")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", metaInfo.AnnounceList)
	for _, file := range metaInfo.Info.Files {
		fmt.Printf("%+v\n", file)
	}
	fmt.Printf("%+v\n", metaInfo.Info.PieceLength)
	fmt.Printf("%+v\n", len(metaInfo.Info.Pieces))
	fmt.Printf("%+v\n", hex.EncodeToString(metaInfo.infoHash[:]))
}
