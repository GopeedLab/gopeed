package message

import (
	"encoding/binary"
)

// message ID
type ID byte

const (
	IdChoke ID = iota
	IdUnchoke
	IdInterested
	IdNotInterested
	IdHave
	IdBitfield
	IdRequest
	IdPiece
	IdCancel
)

/*type Message struct {
	Length  uint32
	ID      MessageID
	Payload []byte
}*/

type base struct {
	id ID
}

func (base *base) ID() ID {
	return base.id
}

func (base *base) Encode() []byte {
	return encode(base, nil)
}

func (base *base) Decode(body []byte) {

}

type Message interface {
	ID() ID
	Encode() []byte
	Decode(body []byte)
}

type TestMsg struct {
	*base
}

func (base *TestMsg) Encode() []byte {
	return []byte{1, 2, 3}
}

func (base *TestMsg) Decode(body []byte) {
}

func encode(base *base, body []byte) []byte {
	buf := make([]byte, 5+len(body))
	binary.BigEndian.PutUint32(buf, uint32(len(buf)))
	buf[4] = byte(base.id)
	copy(buf[4:], body)
	return buf
}

/*func Decode(buf []byte) Message {
	head := make([]byte, 5)
	body := message.Encode()
	binary.BigEndian.PutUint32(head, uint32(len(head)+len(body)))
	head[4] = byte(message.ID())
	return append(head, body...)
}

func (msg *Message) ToBytes() []byte {
	buf := make([]byte, 5)
	binary.BigEndian.PutUint32(buf, msg.Length)
	buf[4] = byte(msg.ID)
	if msg.Payload != nil {
		return append(buf, msg.Payload.Encode()...)
	} else {
		return buf
	}
}

func (msg *Message) FormBytes(buf []byte) {
	msg.Length = binary.BigEndian.Uint32(buf)
	msg.ID = ID(buf[4])
	if msg.Payload != nil {
		msg.Payload.Decode(buf[4:])
	}
}*/

/*func NewMessage(id MessageID, Payload []byte) *Message {
	message := &Message{ID: id, Payload: Payload}
	if id == Keepalive {
		message.Length = 0
	} else {
		message.Length = uint32(len(Payload) + 1)
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
