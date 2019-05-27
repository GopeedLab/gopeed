package bt

import (
	"encoding/binary"
	"fmt"
	"log"
	"testing"
)

func TestDoDownload(t *testing.T) {
	metaInfo, err := ParseFromFile("testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		log.Fatal(err)
	}

	peers, err := metaInfo.Tracker()
	if err != nil {
		log.Fatal(err)
	}

	testPeer := peers[0]
	fmt.Println(testPeer.Address())

	peerId, err := GenPeerID()
	if err != nil {
		log.Fatal(err)
	}

	type fields struct {
		IP   uint32
		Port uint16
	}
	type args struct {
		metaInfo *MetaInfo
		peerId   [20]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"GOTS08E05",
			fields{
				testPeer.IP,
				testPeer.Port,
			},
			args{
				metaInfo,
				peerId,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer := &Peer{
				IP:   tt.fields.IP,
				Port: tt.fields.Port,
			}
			if err := peer.DoDownload(tt.args.metaInfo, tt.args.peerId); (err != nil) != tt.wantErr {
				t.Errorf("Peer.DoDownload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFindPeer(t *testing.T) {
	metaInfo, err := ParseFromFile("testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		log.Fatal(err)
	}

	peers, err := metaInfo.Tracker()
	if err != nil {
		log.Fatal(err)
	}

	// peerId, err := GenPeerID()
	peerId, err := [20]byte{0x2d, 0x55, 0x54, 0x33, 0x35, 0x35, 0x53, 0x2d, 0xaf, 0xb0, 0xec, 0xdc, 0xbf, 0xd1, 0x02, 0x93, 0xd6, 0xf4, 0x9b, 0x24}, nil
	if err != nil {
		log.Fatal(err)
	}
	ch := make(chan Peer)
	for i, peer := range peers {
		go func(i int, peer Peer) {
			fmt.Println(peer.Address())
			err := peer.DoDownload(metaInfo, peerId)
			if err == nil {
				ch <- peer
			} else {
				fmt.Printf("%d:%v\n", i, err)
			}
		}(i, peer)
	}
	peer := <-ch
	fmt.Println(peer.Address())
}

func TestDoDownloadSpecialPeer(t *testing.T) {
	metaInfo, err := ParseFromFile("testdata/Game.of.Thrones.S08E05.720p.WEB.H264-MEMENTO.torrent")
	if err != nil {
		log.Fatal(err)
	}

	// 101.188.129.156:14475
	testPeer := Peer{
		binary.BigEndian.Uint32([]byte{101, 188, 129, 156}),
		14475,
	}
	fmt.Println(testPeer.Address())

	peerId, err := GenPeerID()
	if err != nil {
		log.Fatal(err)
	}

	type fields struct {
		IP   uint32
		Port uint16
	}
	type args struct {
		metaInfo *MetaInfo
		peerId   [20]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"GOTS08E05",
			fields{
				testPeer.IP,
				testPeer.Port,
			},
			args{
				metaInfo,
				peerId,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer := &Peer{
				IP:   tt.fields.IP,
				Port: tt.fields.Port,
			}
			if err := peer.DoDownload(tt.args.metaInfo, tt.args.peerId); (err != nil) != tt.wantErr {
				t.Errorf("Peer.DoDownload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
