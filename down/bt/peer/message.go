package peer

import (
	"encoding/binary"
)

type MessageType int

const (
	Keepalive MessageType = -1
	Choke     MessageType = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

// Message protocol
// length prefix| message ID | payload
// 4-byte       | 1-byte     | remaining bytes
// 100          | 5          | 99-bytes
// https://wiki.theory.org/index.php/BitTorrentSpecification#Messages
type Message struct {
	Length  uint32
	Type    int
	Payload []byte
}

func SplitMessage(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if !atEOF && len(data) > 4 {
		length := int(binary.BigEndian.Uint32(data))
		if len(data)-4 >= length {
			return length + 4, data[:length+4], nil
		}
	}
	return
}

func (msg *Message) Encode() []byte {
	if msg.Type == -1 {
		return make([]byte, 4)
	} else {
		buf := make([]byte, 5+len(msg.Payload))
		binary.BigEndian.PutUint32(buf, msg.Length)
		buf[4] = byte(msg.Type)
		buf = append(buf, msg.Payload[:]...)
		return buf
	}
}

func (msg *Message) Decode(buf []byte) {
	msg.Length = binary.BigEndian.Uint32(buf)
	if msg.Length == 0 {
		msg.Type = -1
	} else {
		msg.Type = int(buf[4])
		if msg.Length > 4 {
			msg.Payload = buf[5:]
		}
	}
}

type MsgBitfield []byte

// 某个分片是否下载完成
func (mb MsgBitfield) get(i int) bool {
	bts := []byte(mb)
	index := i / 8
	if index >= len(bts) {
		return false
	}
	return bts[index]&(1<<uint(7-i%8)) > 0
}
