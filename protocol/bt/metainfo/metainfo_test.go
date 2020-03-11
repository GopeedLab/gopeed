package metainfo

import (
	"encoding/hex"
	"fmt"
	"github.com/RoaringBitmap/roaring"
	"testing"
)

func TestParseFromFile(t *testing.T) {
	// metaInfo, err := ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	metaInfo, err := ParseFromFile("../testdata/office.torrent")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", metaInfo.AnnounceList)
	fmt.Printf("%+v\n", metaInfo.Info.Files)
	fmt.Printf("%+v\n", metaInfo.Info.PieceLength)
	fmt.Printf("%+v\n", hex.EncodeToString(metaInfo.infoHash[:]))
}

func TestParseFromFile2(t *testing.T) {
	a := 0b10010000
	b := 0b01111111
	fmt.Printf("%08b\n", (a&b)^b)

	have := roaring.BitmapOf(0, 3)
	peer := roaring.BitmapOf(1, 2, 3, 4, 5, 6, 7)

	need := roaring.Xor(roaring.And(have, peer), peer)
	fmt.Println(need.ToArray())
}
