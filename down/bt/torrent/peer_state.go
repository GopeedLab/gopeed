package torrent

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/cenkalti/mse"
	"gopeed/down/bt/peer"
	"io"
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

	bitfield peer.MsgBitfield
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
	scanner := bufio.NewScanner(ps.conn)
	scanner.Split(peer.SplitMessage)
	for scanner.Scan() {
		buf := scanner.Bytes()
		message := &peer.Message{}
		message.Decode(buf)
		switch message.ID {
		case peer.Keepalive:
			break
		case peer.Choke:
			ps.peerChoking = true
			break
		case peer.Unchoke:
			ps.handleUnchoke()
			break
		case peer.Interested:
			ps.peerInterested = true
			break
		case peer.NotInterested:
			ps.peerInterested = false
			break
		case peer.Have:
			break
		case peer.Bitfield:
			ps.handleBitfield(message.Payload)
			break
		case peer.Request:
			break
		case peer.Piece:
			break
		case peer.Cancel:
			break
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
			for i := range have {
				buf := make([]byte, 12)
				binary.BigEndian.PutUint32(buf[:4], uint32(have[i]))
				binary.BigEndian.PutUint32(buf[4:8], 0)
				binary.BigEndian.PutUint32(buf[8:], 2<<13)
				_, err := ps.conn.Write(peer.NewMessage(peer.Request, buf).Encode())
				if err != nil {
					ps.conn.Close()
					return
				}
			}
			return
		}
	}
	ps.conn.Close()
}

func (ps *peerState) handleBitfield(payload []byte) {
	ps.bitfield = payload
	have := ps.getHavePieces(ps.bitfield)
	if len(have) > 0 {
		// 表示对该peer感兴趣，并且不choked该peer
		_, err := ps.conn.Write(peer.NewMessage(peer.Interested, []byte{}).Encode())
		if err != nil {
			ps.conn.Close()
			return
		}
		ps.amInterested = true
		_, err = ps.conn.Write(peer.NewMessage(peer.Unchoke, []byte{}).Encode())
		if err != nil {
			ps.conn.Close()
			return
		}
		ps.amChoking = false
	} else {
		ps.conn.Close()
	}
}

// 获取peer能提供需要下载的文件分片
func (ps *peerState) getHavePieces(bitfield peer.MsgBitfield) []int {
	states := make([]bool, len(ps.torrent.PieceState))
	for i := range states {
		states[i] = ps.torrent.PieceState[i].complete
	}
	return bitfield.Have(states)
}
