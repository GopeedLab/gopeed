package torrent

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/cenkalti/mse"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"github.com/monkeyWie/gopeed/down/bt/peer/message"
	"io"
	"math"
	"net"
)

type peerState struct {
	torrent *Torrent
	*peer.Peer
	conn net.Conn
	// this client is choking the peer
	amChoking bool
	// this client is interested in the peer
	amInterested bool
	// peer is choking this client
	peerChoking bool
	// peer is interested in this client
	peerInterested bool

	writeMsgCh chan []byte
	bitfield   *message.Bitfield
}

// 使用MSE加密来避免运营商对bt流量的封锁，基本上现在市面上BT客户端都默认开启了，不用MSE的话很多Peer拒绝连接
// see http://wiki.vuze.com/w/Message_Stream_Encryption
func (ps *peerState) dialMse() error {
	conn, err := net.Dial("tcp", ps.Address())
	if err != nil {
		return err
	}
	mseConn := mse.WrapConn(conn)
	_, err = mseConn.HandshakeOutgoing(ps.torrent.MetaInfo.InfoHash[:], mse.PlainText, nil)
	if err != nil {
		mseConn.Close()
		return err
	}
	ps.conn = mseConn
	return nil
}

// Handshake of Peer wire protocol
// see https://wiki.theory.org/index.php/BitTorrentSpecification#Handshake
func (ps *peerState) handshake() (*peer.Handshake, error) {
	handshakeRes, err := func() (*peer.Handshake, error) {
		handshakeReq := peer.NewHandshake([8]byte{}, ps.torrent.MetaInfo.InfoHash, ps.torrent.PeerID)
		_, err := ps.conn.Write(handshakeReq.Encode())
		if err != nil {
			return nil, err
		}
		var read [68]byte
		_, err = io.ReadFull(ps.conn, read[:])
		if err != nil {
			return nil, err
		}
		handshakeRes := &peer.Handshake{}
		err = handshakeRes.Decode(read[:])
		if err != nil {
			return nil, err
		}
		// InfoHash不匹配
		if handshakeRes.InfoHash != handshakeReq.InfoHash {
			return nil, fmt.Errorf("info_hash not currently serving")
		}
		return handshakeRes, nil
	}()
	if err != nil {
		ps.conn.Close()
		return nil, err
	}
	// init state
	ps.amChoking = true
	ps.amInterested = false
	ps.peerChoking = true
	ps.peerInterested = false
	return handshakeRes, nil
}

func (ps *peerState) download() error {
	// 异步写，防止读取阻塞
	ps.writeMsgCh = make(chan []byte, 64)
	go func() {
		for buf := range ps.writeMsgCh {
			_, err := ps.conn.Write(buf)
			if err != nil {
				fmt.Println(err)
				ps.conn.Close()
			}
		}
	}()
	scanner := bufio.NewScanner(ps.conn)
	scanner.Split(message.SplitMessage)
	for scanner.Scan() {
		buf := scanner.Bytes()
		length := binary.BigEndian.Uint32(buf[:4])
		if length == 0 {
			// 	keepalive
		} else {
			messageID := message.ID(buf[4])
			payload := buf[5:]
			switch messageID {
			case message.IdChoke:
				break
			case message.IdUnchoke:
				ps.handleUnchoke()
				break
			case message.IdInterested:
				break
			case message.IdNotinterested:
				break
			case message.IdHave:
				break
			case message.IdBitfield:
				ps.handleBitfield(payload)
				break
			case message.IdRequest:
				break
			case message.IdPiece:
				ps.handlePiece()
				break
			case message.IdCancel:
				break
			}
		}
	}
	return nil
}

func (ps *peerState) handleUnchoke() {
	ps.peerChoking = false
	// 如果客户端对peer感兴趣并且peer没有choked客户端，就可以开始下载了
	if ps.amInterested {
		have := ps.getHavePieces(ps.bitfield)
		if len(have) > 0 {
			// 获取分片的长度
			for _, index := range have {
				pieceLength := ps.torrent.MetaInfo.GetPieceSize(index)
				lastOffset := uint64(0)
				// 按块下载分片
				for lastOffset < pieceLength {
					blockLength := math.Min(2<<13, float64(pieceLength-lastOffset))
					ps.write(message.NewRequest(uint32(index), uint32(lastOffset), uint32(blockLength)).Encode())
					lastOffset += uint64(blockLength)
					fmt.Println("request piece")
				}
				// TODO 测试代码
				break
			}
			return
		}
	}
	ps.conn.Close()
}

func (ps *peerState) handleBitfield(payload []byte) {
	ps.bitfield = message.NewBitfield(payload)
	have := ps.getHavePieces(ps.bitfield)
	if len(have) > 0 {
		// 表示对该peer感兴趣，并且不choked该peer
		ps.write(message.NewInterested().Encode())
		ps.amInterested = true

		ps.write(message.NewUnchoke().Encode())
		ps.amChoking = false
	} else {
		ps.conn.Close()
	}
}

// 处理分片下载响应
func (ps *peerState) handlePiece() {
	fmt.Println("one piece")
}

// 获取peer能提供需要下载的文件分片
func (ps *peerState) getHavePieces(bitfield *message.Bitfield) []int {
	states := make([]bool, len(ps.torrent.PieceState))
	for i := range states {
		states[i] = ps.torrent.PieceState[i].complete
	}
	return bitfield.Have(states)
}

// 异步写入数据
func (ps *peerState) write(buf []byte) {
	ps.writeMsgCh <- buf
}

// 获取要写入到的文件
/*func (ps *peerState) getWriteFile(request *message.Request) string {
	info := ps.torrent.MetaInfo.Info
	// 单文件
	if len(info.Files) == 0 {
		return info.Name
	} else {
		request.Index * info.PieceLength + be
		for i := 0; i < len(info.Files); i++ {

		}
	}
}
*/
