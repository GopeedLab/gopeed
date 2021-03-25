package peer

import (
	"bytes"
	"encoding/binary"
)

const (
	ProtocolIdentifier       = "BitTorrent protocol"
	ProtocolIdentifierLength = 0x13
)

// Handshake of Peer wire protocol
// Per https://wiki.theory.org/index.php/BitTorrentSpecification#Handshake
type Handshake struct {
	Pstrlen  byte
	Pstr     [19]byte
	Reserved [8]byte
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(reserved [8]byte, infoHash [20]byte, peerID [20]byte) *Handshake {
	var arr [ProtocolIdentifierLength]byte
	copy(arr[:], ProtocolIdentifier)
	return &Handshake{
		Pstrlen:  ProtocolIdentifierLength,
		Pstr:     arr,
		Reserved: reserved,
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (handshake *Handshake) Encode() []byte {
	writer := new(bytes.Buffer)
	err := binary.Write(writer, binary.BigEndian, handshake)
	if err != nil {
		panic(err)
	}
	return writer.Bytes()
}

func (handshake *Handshake) Decode(buf []byte) error {
	reader := bytes.NewReader(buf)
	err := binary.Read(reader, binary.BigEndian, handshake)
	if err != nil {
		return err
	}
	return nil
}
