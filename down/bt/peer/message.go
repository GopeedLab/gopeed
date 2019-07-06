package peer

import (
	"encoding/binary"
)

type MessageID byte

const (
	Keepalive     MessageID = -1
	Choke         MessageID = 0
	Unchoke       MessageID = 1
	Interested    MessageID = 2
	NotInterested MessageID = 3
	Have          MessageID = 4
	Bitfield      MessageID = 5
	Request       MessageID = 6
	Piece         MessageID = 7
	Cancel        MessageID = 8
)

type Message struct {
	Length  uint32
	ID      MessageID
	Payload []byte
}

func NewMessage(id MessageID, payload []byte) *Message {
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
		buf = append(buf, msg.Payload[:]...)
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
}

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

type MsgBitfield []byte

// 某个分片是否下载完成
func (mb MsgBitfield) IsComplete(i int) bool {
	bts := []byte(mb)
	index := i / 8
	if index >= len(bts) {
		return false
	}
	return bts[index]&(1<<uint(7-i%8)) > 0
}

// 给定一组分片下载状态，计算出当前peer能提供下载的分片下标
func (mb MsgBitfield) Have(pieces []bool) []int {
	arr := make([]int, 0)
	bts := []byte(mb)
	length := len(pieces) / 8
	if len(pieces)%8 != 0 {
		length++
	}

	for i := 0; i < length; i++ {
		for j := 0; j < 8; j++ {
			index := i*8 + j
			// 如果此分片在本地还未下载，检查peer是否能提供该分片下载
			if index < len(pieces) &&
				!pieces[index] &&
				i < len(bts) &&
				uint(bts[i])&(1<<uint(7-j)) > 0 {
				arr = append(arr, index)
			}
		}
	}
	return arr
}
