package bt

import (
	"testing"
)

func TestClient_AddTorrent(t *testing.T) {
	client := NewClient()
	client.AddTorrent("testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
}
