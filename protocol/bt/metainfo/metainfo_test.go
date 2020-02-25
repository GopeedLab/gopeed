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
	bitmap := roaring.NewBitmap()
	bitmap.AddMany([]uint32{1, 2, 3, 4, 5, 6, 7, 10, 20, 9, 9, 1})
	fmt.Println(bitmap.Contains(1))
	fmt.Println(bitmap.Contains(2))
	fmt.Println(bitmap.Contains(8))
	fmt.Println(bitmap.GetCardinality())
}
