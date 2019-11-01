package metainfo

import (
	"fmt"
	"testing"
)

func TestParseFromFile(t *testing.T) {
	metaInfo, err := ParseFromFile("../testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", metaInfo.AnnounceList)
	fmt.Printf("%+v\n", metaInfo.Info.Files)
}
