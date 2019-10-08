package message

import (
	"encoding/binary"
)

// message ID
type ID byte

const (
	IdChoke         ID = 0
	IdUnchoke       ID = 1
	IdInterested    ID = 2
	IdNotinterested ID = 3
	IdHave          ID = 4
	IdBitfield      ID = 5
	IdRequest       ID = 6
	IdPiece         ID = 7
	IdCancel        ID = 8
)

/*type Message struct {
	Length  uint32
	ID      MessageID
	Payload []byte
}*/
type Message struct {
	Length uint32
	ID     ID
}

func (msg *Message) Encode() []byte {
	buf := make([]byte, 5)
	binary.BigEndian.PutUint32(buf, msg.Length)
	buf[4] = byte(msg.ID)
	return buf
}

func (msg *Message) Decode(buf []byte) {
	msg.Length = binary.BigEndian.Uint32(buf)
	msg.ID = ID(buf[4])
}

type Serialize interface {
	Encode() []byte
	Decode(buf []byte) Serialize
}

/*func NewMessage(id MessageID, payload []byte) *Message {
	message := &Message{ID: id, Payload: payload}
	if id == Keepalive {
		message.Length = 0
	} else {
		message.Length = uint32(len(payload) + 1)
	}
	return message
}

func (msg *Message) Encode() []byte {
	if msg.ID == Keepalive {
		return make([]byte, 4)
	} else {
		buf := make([]byte, 5+len(msg.Payload))
		binary.BigEndian.PutUint32(buf, msg.Length)
		buf[4] = byte(msg.ID)
		copy(buf[5:], msg.Payload[:])
		return buf
	}
}

func (msg *Message) Decode(buf []byte) {
	msg.Length = binary.BigEndian.Uint32(buf)
	if msg.Length == 0 {
		msg.ID = -1
	} else {
		msg.ID = MessageID(buf[4])
		if msg.Length > 4 {
			msg.Payload = buf[5:]
		}
	}
}*/

// Message protocol
// Length prefix| message ID | Payload
// 4-byte       | 1-byte     | remaining bytes
// 100          | 5          | 99-bytes
// https://wiki.theory.org/index.php/BitTorrentSpecification#Messages
func SplitMessage(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if !atEOF && len(data) > 4 {
		length := int(binary.BigEndian.Uint32(data))
		if len(data)-4 >= length {
			return length + 4, data[:length+4], nil
		}
	}
	return
}
