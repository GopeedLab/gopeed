package torrent

import (
	"bufio"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/cenkalti/mse"
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"github.com/monkeyWie/gopeed/down/bt/peer/message"
	"io"
	"math"
	"net"
	"os"
	"path"
	"sync"
	"time"
)

type peerConn struct {
	torrent *Torrent
	peer    *peer.Peer
	conn    net.Conn
	// this client is choking the peer
	amChoking bool
	// this client is interested in the peer
	amInterested bool
	// peer is choking this client
	peerChoking bool
	// peer is interested in this client
	peerInterested bool

	bitfield          *message.Bitfield
	readyComplete     bool
	pieceDownloadedCh chan bool
	// block下载队列，官方推荐为5
	pieceBlockQueue chan interface{}
	// piece上所有的block下载状态
	pieceBlockState sync.Map
}

func NewPeerConn(torrent *Torrent, peer *peer.Peer) *peerConn {
	return &peerConn{
		torrent: torrent,
		peer:    peer,
	}
}

// 使用MSE加密来避免运营商对bt流量的封锁，基本上现在市面上BT客户端都默认开启了，不用MSE的话很多Peer拒绝连接
// see http://wiki.vuze.com/w/Message_Stream_Encryption
func (pc *peerConn) dialMse() error {
	conn, err := net.DialTimeout("tcp", pc.peer.Address(), time.Second*time.Duration(30))
	if err != nil {
		return err
	}
	mseConn := mse.WrapConn(conn)
	infoHash := pc.torrent.MetaInfo.GetInfoHash()
	_, err = mseConn.HandshakeOutgoing(infoHash[:], mse.PlainText, nil)
	if err != nil {
		mseConn.Close()
		return err
	}
	pc.conn = mseConn
	return nil
}

// Handshake of Peer wire protocol
// see https://wiki.theory.org/index.php/BitTorrentSpecification#Handshake
func (pc *peerConn) handshake() (*peer.Handshake, error) {
	handshakeRes, err := func() (*peer.Handshake, error) {
		handshakeReq := peer.NewHandshake([8]byte{}, pc.torrent.MetaInfo.GetInfoHash(), pc.torrent.PeerID)
		_, err := pc.conn.Write(handshakeReq.Encode())
		if err != nil {
			return nil, err
		}
		var read [68]byte
		_, err = io.ReadFull(pc.conn, read[:])
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
		pc.conn.Close()
		return nil, err
	}
	// init state
	pc.amChoking = true
	pc.amInterested = false
	pc.peerChoking = true
	pc.peerInterested = false
	return handshakeRes, nil
}

// 准备下载
func (pc *peerConn) ready() error {
	if err := pc.dialMse(); err != nil {
		return err
	}
	if _, err := pc.handshake(); err != nil {
		return err
	}
	pc.readyComplete = false
	readyCh := make(chan bool)
	defer close(readyCh)
	go func() {
		scanner := bufio.NewScanner(pc.conn)
		scanner.Split(message.SplitMessage)
		for scanner.Scan() {
			buf := scanner.Bytes()
			length := binary.BigEndian.Uint32(buf[:4])
			if length == 0 {
				// 	keepalive
			} else {
				messageID := message.ID(buf[4])
				switch messageID {
				case message.IdChoke:
					break
				case message.IdUnchoke:
					pc.handleUnchoke(readyCh)
					break
				case message.IdInterested:
					break
				case message.IdNotinterested:
					break
				case message.IdHave:
					break
				case message.IdBitfield:
					pc.handleBitfield(buf)
					break
				case message.IdRequest:
					break
				case message.IdPiece:
					pc.handlePiece(buf)
					break
				case message.IdCancel:
					break
				}
			}
		}
	}()
	err := func() error {
		select {
		case status := <-readyCh:
			if status {
				return nil
			} else {
				return errors.New("ready fail")
			}
		case <-time.After(time.Second * 30):
			// 30秒之后超时
			return errors.New("ready time out")
		}
	}()
	pc.readyComplete = true
	if err != nil {
		pc.conn.Close()
		return err
	}
	pc.pieceBlockQueue = make(chan interface{}, 5)
	return nil
}

// 下载指定piece
func (pc *peerConn) downloadPiece(index int) error {
	pc.pieceDownloadedCh = make(chan bool)
	defer close(pc.pieceDownloadedCh)
	pc.pieceBlockState = sync.Map{}
	pieceLength := pc.torrent.MetaInfo.GetPieceSize(index)
	lastOffset := uint64(0)

	waitDuration := time.Second * 30
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	// 按块下载分片
	for lastOffset < pieceLength {
		blockLength := math.Min(2<<13, float64(pieceLength-lastOffset))
		// 标记block开始下载
		lastOffset32 := uint32(lastOffset)
		pc.pieceBlockState.Store(lastOffset32, false)
		// block下载排队，并做超时处理
		timer.Reset(waitDuration)
		select {
		case pc.pieceBlockQueue <- nil:
		case <-timer.C:
			return errors.New("download timeout")
		}
		_, err := pc.conn.Write(message.NewRequest(uint32(index), lastOffset32, uint32(blockLength)).Encode())
		if err != nil {
			fmt.Println(err)
		}
		lastOffset += uint64(blockLength)
	}
	// 监听是否下载完成
	<-pc.pieceDownloadedCh
	return nil
}

func (pc *peerConn) handleUnchoke(readyCh chan<- bool) {
	pc.peerChoking = false
	// 已经处理过Unchoke信号
	if pc.readyComplete {
		return
	}
	// 如果客户端对peer感兴趣并且peer没有choked客户端，就可以开始下载了
	if pc.amInterested {
		readyCh <- true
	} else {
		readyCh <- false
	}
}

func (pc *peerConn) handleBitfield(buf []byte) {
	pc.bitfield = &message.Bitfield{}
	pc.bitfield.Decode(buf)
	have := pc.getHavePieces(pc.bitfield)
	if len(have) > 0 {
		// 表示对该peer感兴趣，并且不choked该peer
		pc.conn.Write(message.NewInterested().Encode())
		pc.amInterested = true

		pc.conn.Write(message.NewUnchoke().Encode())
		pc.amChoking = false
	} else {
		pc.conn.Close()
	}
}

// 处理分片下载响应
func (pc *peerConn) handlePiece(buf []byte) {
	piece := &message.Piece{}
	piece.Decode(buf)
	info := pc.torrent.MetaInfo.Info
	fds := pc.torrent.MetaInfo.GetFileDetails()
	blockLength := int64(len(piece.Block))
	pieceBegin := int64(piece.Index) * int64(info.PieceLength)
	blockBegin := pieceBegin + int64(piece.Begin)
	// 获取对应要的文件
	var fileBlocks []fileBlock
	if len(info.Files) == 0 {
		// 单文件
		fileBlocks = append(fileBlocks, fileBlock{
			filepath:   info.Name,
			fileSeek:   blockBegin,
			blockBegin: 0,
			blockEnd:   blockLength,
		})
	} else {
		// 获取要写入的第一个文件和偏移
		writeIndex := getWriteFile(blockBegin, fds)
		// block对应种子的偏移
		blockFileBegin := blockBegin
		// block对应文件的偏移
		var blockSeek int64 = 0
		for _, f := range fds[writeIndex:] {
			// 计算文件可剩余写入字节数
			fileWritable := f.End - blockFileBegin
			// 计算block剩余写入字节数
			blockWritable := blockLength - blockSeek
			fb := fileBlock{
				filepath:   f.Path[0],
				fileSeek:   blockFileBegin - f.Begin,
				blockBegin: blockSeek,
			}
			if fileWritable >= blockWritable {
				// 若够block写入直接跳出循环
				fb.blockEnd = blockSeek + blockWritable
				fileBlocks = append(fileBlocks, fb)
				break
			} else {
				// 否则计算剩余可写字节数，写入到下一个文件中
				fb.blockEnd = blockSeek + fileWritable
				fileBlocks = append(fileBlocks, fb)
				blockFileBegin = f.End
				blockSeek += fileWritable
			}
		}
	}
	// fmt.Println(fileBlocks)
	for _, f := range fileBlocks {
		func() {
			name := path.Join(pc.torrent.Path, f.filepath)
			file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				panic(err)
			}
			defer file.Close()
			_, err = file.WriteAt(piece.Block[f.blockBegin:f.blockEnd], f.fileSeek)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}
	pc.pieceBlockState.Store(piece.Begin, true)
	// 出队
	<-pc.pieceBlockQueue
	flag := true
	// fmt.Printf("index %d: %v\n", piece.Index, pc.pieceBlockState)
	pc.pieceBlockState.Range(func(key, value interface{}) bool {
		// 如果有还没下载完的block
		if !value.(bool) {
			flag = false
			return false
		}
		return true
	})
	// piece全部下载完
	if flag {
		// 计算piece对应的文件偏移
		sha1 := sha1.New()
		pieceLength := pc.torrent.MetaInfo.GetPieceSize(int(piece.Index))
		writeIndex := getWriteFile(pieceBegin, fds)
		fileBegin := pieceBegin - fds[writeIndex].Begin
		for _, fd := range fds[writeIndex:] {
			func() {
				file, err := os.Open(path.Join(pc.torrent.Path, fd.Path[0]))
				if err != nil {
					panic(err)
				}
				defer file.Close()
				_, err = file.Seek(fileBegin, 0)
				if err != nil {
					panic(err)
				}
				written, err := io.CopyN(sha1, file, int64(pieceLength))
				if err != nil {
					if err != io.EOF {
						panic(err)
					}
				}
				pieceLength -= uint64(written)
			}()
			if pieceLength > 0 {
				// 	继续读下个文件
				fileBegin = 0
			} else {
				break
			}
		}
		// 校验piece SHA-1 hash
		downHash := [20]byte{}
		copy(downHash[:], sha1.Sum(nil))
		if downHash == pc.torrent.MetaInfo.Info.Pieces[piece.Index] {
			fmt.Printf("piece %d 校验通过\n", piece.Index)
		} else {
			fmt.Printf("piece %d 校验失败\n", piece.Index)
		}
		// piece下载完成
		pc.pieceDownloadedCh <- true
	}
}

// 获取piece对应要写入的文件
func getWriteFile(pieceBegin int64, fds []metainfo.FileDetail) int {
	for i, f := range fds {
		if f.Begin <= pieceBegin && f.End > pieceBegin {
			return i
		}
	}
	return -1
}

type fileBlock struct {
	filepath   string
	fileSeek   int64
	blockBegin int64
	blockEnd   int64
}

// 获取peer能提供需要下载的文件分片
func (pc *peerConn) getHavePieces(bitfield *message.Bitfield) []int {
	states := make([]bool, pc.torrent.PiecesState.Size())
	for i := range states {
		states[i] = pc.torrent.PiecesState.getState(i) == stateFinished
	}
	return bitfield.Have(states)
}

// 获取要写入到的文件
/*func (ps *peerConn) getWriteFile(request *message.Request) string {
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
