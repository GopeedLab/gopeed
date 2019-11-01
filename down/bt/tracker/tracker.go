package tracker

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/peer"
	"io"
	"io/ioutil"
	"math"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	udpConnectRequestMagic = 0x41727101980
	udpConnectTimeout      = 15
	udpConnectRetries      = 8
	udpActionConnect       = 0
	udpActionAnnounce      = 1
	udpActionScrape        = 2
	udpActionError         = 3
)

/*
	Offset  Size            Name            Value
	0       64-bit integer  protocol_id     0x41727101980 // magic constant
	8       32-bit integer  action          0 // connect
	12      32-bit integer  transaction_id
	16
	Per https://www.libtorrent.org/udp_tracker_protocol.html#connecting
*/
type udpConnectRequest struct {
	protocolId    uint64
	action        uint32
	transactionId uint32
}

func newUdpConnectRequest() *udpConnectRequest {
	return &udpConnectRequest{
		protocolId:    udpConnectRequestMagic,
		action:        udpActionConnect,
		transactionId: mrand.Uint32(),
	}
}

func (req *udpConnectRequest) encode() []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], req.protocolId)
	binary.BigEndian.PutUint32(buf[8:12], req.action)
	binary.BigEndian.PutUint32(buf[12:], req.transactionId)
	return buf
}

/*
	Offset  Size            Name            Value
	0       32-bit integer  action          0 // connect
	4       32-bit integer  transaction_id
	8       64-bit integer  connection_id
	16
*/
type udpConnectResponse struct {
	action        uint32
	transactionId uint32
	connectionId  uint64
}

func newUdpConnectResponse(buf []byte) *udpConnectResponse {
	return &udpConnectResponse{
		action:        binary.BigEndian.Uint32(buf[:4]),
		transactionId: binary.BigEndian.Uint32(buf[4:8]),
		connectionId:  binary.BigEndian.Uint64(buf[8:]),
	}
}

/*
	Offset  Size    Name    Value
	0       64-bit integer  connection_id
	8       32-bit integer  action          1 // announce
	12      32-bit integer  transaction_id
	16      20-byte string  info_hash
	36      20-byte string  peer_id
	56      64-bit integer  downloaded
	64      64-bit integer  left
	72      64-bit integer  uploaded
	80      32-bit integer  event           0 // 0: none; 1: completed; 2: started; 3: stopped
	84      32-bit integer  ip address      0 // default
	88      32-bit integer  key
	92      32-bit integer  num_want        -1 // default
	96      16-bit integer  port
	98
	Per https://www.libtorrent.org/udp_tracker_protocol.html#announcing
*/
type udpAnnounceRequest struct {
	connectionId  uint64
	action        uint32
	transactionId uint32
	infoHash      [20]byte
	peerID        [20]byte
	downloaded    uint64
	left          uint64
	uploaded      uint64
	event         uint32
	ip            uint32
	key           uint32
	numWant       int32
	port          uint16
}

func newUdpAnnounceRequest(connectionId uint64) *udpAnnounceRequest {
	return &udpAnnounceRequest{
		connectionId:  connectionId,
		action:        udpActionAnnounce,
		transactionId: mrand.Uint32(),
	}
}

func (req *udpAnnounceRequest) encode() []byte {
	buf := make([]byte, 98, 98)
	binary.BigEndian.PutUint64(buf[:8], req.connectionId)
	binary.BigEndian.PutUint32(buf[8:12], req.action)
	binary.BigEndian.PutUint32(buf[12:16], req.transactionId)
	copy(buf[16:36], req.infoHash[:])
	copy(buf[36:56], req.peerID[:])
	binary.BigEndian.PutUint64(buf[56:64], req.downloaded)
	binary.BigEndian.PutUint64(buf[64:72], req.left)
	binary.BigEndian.PutUint64(buf[72:80], req.uploaded)
	binary.BigEndian.PutUint32(buf[80:84], req.event)
	binary.BigEndian.PutUint32(buf[84:88], req.ip)
	binary.BigEndian.PutUint32(buf[88:92], req.key)
	binary.BigEndian.PutUint32(buf[92:96], uint32(req.numWant))
	binary.BigEndian.PutUint16(buf[96:98], req.port)
	return buf
}

/*
	Offset      Size            Name            Value
	0           32-bit integer  action          1 // announce
	4           32-bit integer  transaction_id
	8           32-bit integer  interval
	12          32-bit integer  leechers
	16          32-bit integer  seeders
	20 + 6 * n  32-bit integer  ip address
	24 + 6 * n  16-bit integer  TCP port
	20 + 6 * N
	Per https://www.libtorrent.org/udp_tracker_protocol.html#announcing
*/
type udpAnnounceResponse struct {
	action        uint32
	transactionId uint32
	interval      uint32
	leechers      uint32
	seeders       uint32
	peers         []peer.Peer
}

func newUdpAnnounceResponse(buf []byte) *udpAnnounceResponse {
	response := &udpAnnounceResponse{
		action:        binary.BigEndian.Uint32(buf[:4]),
		transactionId: binary.BigEndian.Uint32(buf[4:8]),
		interval:      binary.BigEndian.Uint32(buf[8:12]),
		leechers:      binary.BigEndian.Uint32(buf[12:16]),
		seeders:       binary.BigEndian.Uint32(buf[16:20]),
	}
	count := (len(buf) - 20) / 6
	response.peers = make([]peer.Peer, count, count)
	for i := 0; i < count; i++ {
		ipBegin := 20 + 6*i
		portBegin := ipBegin + 4
		response.peers[i] = peer.Peer{
			IP:   binary.BigEndian.Uint32(buf[ipBegin:portBegin]),
			Port: binary.BigEndian.Uint16(buf[portBegin : portBegin+2]),
		}
	}
	return response
}

type Tracker struct {
	PeerID   [20]byte
	MetaInfo *metainfo.MetaInfo
}

func (tracker *Tracker) connect(conn *net.UDPConn, timeout int64) (response *udpConnectResponse, err error) {
	request := newUdpConnectRequest()
	buf := request.encode()
	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
	conn.Write(buf)
	len, err := conn.Read(buf)
	if err != nil {
		return
	}
	if len != 16 {
		err = NewTrackerError(ErrResponse, errors.New("invalid response"))
		return
	}
	response = newUdpConnectResponse(buf)
	return
}

func (tracker *Tracker) announce(conn *net.UDPConn, timeout int64, connectionId uint64) (response *udpAnnounceResponse, err error) {
	request := newUdpAnnounceRequest(connectionId)
	request.infoHash = tracker.MetaInfo.GetInfoHash()
	request.peerID = tracker.PeerID
	request.downloaded = 0
	request.uploaded = 0
	request.left = tracker.MetaInfo.GetTotalSize() - request.downloaded
	request.event = 0
	request.ip = 0
	request.key = 0
	request.numWant = 50
	request.port = 6882
	encode := request.encode()
	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
	conn.Write(encode)
	buf := make([]byte, 512)
	len, err := conn.Read(buf)
	if err != nil {
		return
	}
	response = newUdpAnnounceResponse(buf[:len])
	return
}

// 通过tracker服务器获取peer信息
func (tracker *Tracker) Tracker() (peers []peer.Peer, err error) {
	metaInfo := tracker.MetaInfo

	checkAnnounceList := func(announceList [][]string) bool {
		if len(announceList) > 0 {
			for _, l := range announceList {
				if len(l) > 0 {
					return true
				}
			}
		}
		return false
	}

	// 只要获取到了peers就直接返回
	if checkAnnounceList(metaInfo.AnnounceList) {
		for _, announceArr := range metaInfo.AnnounceList {
			if len(announceArr) > 0 {
				for _, announce := range announceArr {
					peers, err = tracker.DoTracker(announce)
					if err == nil {
						return
					}
				}
			}
		}
	} else {
		peers, err = tracker.DoTracker(metaInfo.Announce)
	}
	return
}

func (tracker *Tracker) DoTracker(announce string) (peers []peer.Peer, err error) {
	if announce != "" {
		url, _ := url.Parse(announce)
		switch url.Scheme {
		case "http", "https":
			return tracker.httpTracker(url)
		case "udp", "udp4", "udp6":
			return tracker.udpTracker(url)
		default:
			return nil, errors.New("unsupported protocol")
		}
	}
	return nil, errors.New("empty announce")
}

// http://bittorrent.org/beps/bep_0003.html#trackers
func (tracker *Tracker) httpTracker(url *url.URL) (peers []peer.Peer, err error) {
	metaInfo := tracker.MetaInfo
	peerID := tracker.PeerID

	query := url.Query()
	infoHash := metaInfo.GetInfoHash()
	query.Add("info_hash", string(infoHash[:]))
	query.Add("peer_id", string(peerID[:]))
	query.Add("port", "6882")
	query.Add("uploaded", "0")
	query.Add("downloaded", "0")
	query.Add("left", strconv.FormatInt(int64(metaInfo.GetTotalSize()), 10))
	url.RawQuery = query.Encode()
	request, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return
	}
	var httpClient = &http.Client{
		Timeout: time.Second * time.Duration(15),
	}
	request.Header.Set("User-Agent", "gopeed/0.0.1")
	response, err := httpClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	resp, err := ioutil.ReadAll(response.Body)
	if err != nil {
		if err != io.EOF {
			return
		}
	}
	// TODO Parse HTTP tracker response
	fmt.Println(resp)
	return nil, nil
}

// http://bittorrent.org/beps/bep_0015.html
func (tracker *Tracker) udpTracker(url *url.URL) (peers []peer.Peer, err error) {
	conn, err := dial(url)
	if err != nil {
		return
	}
	defer conn.Close()

	announceResponse, err := func() (announceResponse *udpAnnounceResponse, err error) {
		var connectResponse *udpConnectResponse
		// 0:connect 1:announce
		state := 0
		// 	访问超时最多重试8次，当有请求成功则将重试次数重置为0
		for n := 0; n < udpConnectRetries; n++ {
			// 15 * 2 ^ n seconds
			timeout := int64(udpConnectTimeout * math.Pow(2, float64(n)))
			switch state {
			case 0:
				connectResponse, err = tracker.connect(conn, timeout)
				if err != nil {
					if oe, ok := err.(*net.OpError); ok && oe.Timeout() {
						break
					} else {
						return
					}
				}
				n = -1
				state = 1
				break
			case 1:
				announceResponse, err = tracker.announce(conn, timeout, connectResponse.connectionId)
				if err != nil {
					if oe, ok := err.(*net.OpError); ok && oe.Timeout() {
						break
					} else {
						return
					}
				}
				return
			}
		}
		return
	}()

	if err != nil {
		return
	}

	return announceResponse.peers, nil
}

func dial(trackerUrl *url.URL) (*net.UDPConn, error) {
	host, portStr, err := net.SplitHostPort(trackerUrl.Host)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return nil, err
	}
	ip, err := lookupIP(host, trackerUrl.Scheme)
	if err != nil {
		return nil, err
	}

	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}
	return net.DialUDP(trackerUrl.Scheme, srcAddr, dstAddr)
}

func lookupIP(host string, scheme string) (ip net.IP, err error) {
	ips, _ := net.LookupIP(host)
	if len(ips) == 0 {
		err = errors.New("no ips")
		return
	}
	for _, ip = range ips {
		switch scheme {
		case "udp", "udp4":
			if ip.To4() != nil {
				return
			}
		case "udp6":
			if ip.To4() == nil {
				return
			}
		}
	}
	err = errors.New("no available ips ")
	return
}
