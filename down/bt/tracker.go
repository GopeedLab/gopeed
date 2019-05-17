package bt

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
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
	Detail see https://www.libtorrent.org/udp_tracker_protocol.html#connecting
*/
type UdpConnectRequest struct {
	ProtocolId    uint64
	Action        uint32
	TransactionId uint32
}

func NewUdpConnectRequest() *UdpConnectRequest {
	return &UdpConnectRequest{
		ProtocolId:    udpConnectRequestMagic,
		Action:        udpActionConnect,
		TransactionId: mrand.Uint32(),
	}
}

func (req *UdpConnectRequest) Encode() []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], req.ProtocolId)
	binary.BigEndian.PutUint32(buf[8:12], req.Action)
	binary.BigEndian.PutUint32(buf[12:], req.TransactionId)
	return buf
}

/*
	Offset  Size            Name            Value
	0       32-bit integer  action          0 // connect
	4       32-bit integer  transaction_id
	8       64-bit integer  connection_id
	16
*/
type UdpConnectResponse struct {
	Action        uint32
	TransactionId uint32
	ConnectionId  uint64
}

func NewUdpConnectResponse(buf []byte) *UdpConnectResponse {
	return &UdpConnectResponse{
		Action:        binary.BigEndian.Uint32(buf[:4]),
		TransactionId: binary.BigEndian.Uint32(buf[4:8]),
		ConnectionId:  binary.BigEndian.Uint64(buf[8:]),
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
	84      32-bit integer  IP address      0 // default
	88      32-bit integer  key
	92      32-bit integer  num_want        -1 // default
	96      16-bit integer  port
	98
	Detail see https://www.libtorrent.org/udp_tracker_protocol.html#announcing
*/
type UdpAnnounceRequest struct {
	ConnectionId  uint64
	Action        uint32
	TransactionId uint32
	InfoHash      [20]byte
	PeerID        [20]byte
	Downloaded    uint64
	Left          uint64
	Uploaded      uint64
	Event         uint32
	IP            uint32
	Key           uint32
	NumWant       int32
	Port          uint16
}

func NewUdpAnnounceRequest(connectionId uint64) *UdpAnnounceRequest {
	return &UdpAnnounceRequest{
		ConnectionId:  connectionId,
		Action:        udpActionAnnounce,
		TransactionId: mrand.Uint32(),
	}
}

func (req *UdpAnnounceRequest) encode() []byte {
	buf := make([]byte, 98, 98)
	binary.BigEndian.PutUint64(buf[:8], req.ConnectionId)
	binary.BigEndian.PutUint32(buf[8:12], req.Action)
	binary.BigEndian.PutUint32(buf[12:16], req.TransactionId)
	copy(buf[16:36], req.InfoHash[:])
	copy(buf[36:56], req.PeerID[:])
	binary.BigEndian.PutUint64(buf[56:64], req.Downloaded)
	binary.BigEndian.PutUint64(buf[64:72], req.Left)
	binary.BigEndian.PutUint64(buf[72:80], req.Uploaded)
	binary.BigEndian.PutUint32(buf[80:84], req.Event)
	binary.BigEndian.PutUint32(buf[84:88], req.IP)
	binary.BigEndian.PutUint32(buf[88:92], req.Key)
	binary.BigEndian.PutUint32(buf[92:96], uint32(req.NumWant))
	binary.BigEndian.PutUint16(buf[96:98], req.Port)
	return buf
}

/*
	Offset      Size            Name            Value
	0           32-bit integer  action          1 // announce
	4           32-bit integer  transaction_id
	8           32-bit integer  interval
	12          32-bit integer  leechers
	16          32-bit integer  seeders
	20 + 6 * n  32-bit integer  IP address
	24 + 6 * n  16-bit integer  TCP port
	20 + 6 * N
*/
type UdpAnnounceResponse struct {
	Action        uint32
	TransactionId uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	IP            []uint32
	Port          []uint16
}

func NewUdpAnnounceResponse(buf []byte) *UdpAnnounceResponse {
	response := &UdpAnnounceResponse{
		Action:        binary.BigEndian.Uint32(buf[:4]),
		TransactionId: binary.BigEndian.Uint32(buf[4:8]),
		Interval:      binary.BigEndian.Uint32(buf[8:12]),
		Leechers:      binary.BigEndian.Uint32(buf[12:16]),
		Seeders:       binary.BigEndian.Uint32(buf[16:20]),
	}
	count := (len(buf) - 20) / 6
	response.IP = make([]uint32, count, count)
	response.Port = make([]uint16, count, count)
	for i := 0; i < count; i++ {
		begin := 20 + 6*i
		response.IP[i] = binary.BigEndian.Uint32(buf[begin : begin+4])
		response.Port[i] = binary.BigEndian.Uint16(buf[begin+4 : begin+6])
	}
	return response
}

type UdpTrackerContent struct {
	URL              *url.URL
	MetaInfo         *MetaInfo
	Conn             *net.UDPConn
	ConnectRequest   *UdpConnectRequest
	ConnectResponse  *UdpConnectResponse
	AnnounceRequest  *UdpAnnounceRequest
	AnnounceResponse *UdpAnnounceResponse
}

func NewUdpTrackerContent(url *url.URL, metaInfo *MetaInfo) *UdpTrackerContent {
	return &UdpTrackerContent{
		URL:      url,
		MetaInfo: metaInfo,
	}
}

func (content *UdpTrackerContent) connect() error {
	host, portStr, err := net.SplitHostPort(content.URL.Host)
	if err != nil {
		fmt.Println(err)
		return err
	}
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		fmt.Println(err)
		return err
	}
	ip, err := lookupIP(host, content.URL.Scheme)
	if err != nil {
		fmt.Println(err)
		return err
	}

	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}
	conn, err := net.DialUDP(content.URL.Scheme, srcAddr, dstAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// Connect
	request := NewUdpConnectRequest()
	var response *UdpConnectResponse
	for n := 0; n < udpConnectRetries; n++ {
		buf := request.Encode()
		conn.Write(buf)
		// 15 * 2 ^ n seconds
		timeout := udpConnectTimeout * math.Pow(2, float64(n))
		conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
		len, err := conn.Read(buf)
		if len == 16 && err == nil {
			response = NewUdpConnectResponse(buf)
			break
		}
	}
	if response == nil {
		conn.Close()
		return errors.New("UDP tracker connect error")
	}
	if response.Action != udpActionConnect || request.TransactionId != response.TransactionId {
		conn.Close()
		return errors.New("UDP tracker connect response error")
	}
	content.Conn = conn
	content.ConnectRequest = request
	content.ConnectResponse = response
	return nil
}

func (content *UdpTrackerContent) announce() error {
	request := NewUdpAnnounceRequest(content.ConnectResponse.ConnectionId)
	request.InfoHash = content.MetaInfo.InfoHash
	peerID, err := genPeerID()
	if err != nil {
		fmt.Println(err)
		return err
	}
	request.PeerID = peerID
	request.Downloaded = 0
	request.Uploaded = 0
	request.Left = content.MetaInfo.GetTotalSize() - request.Downloaded
	request.Event = 0
	request.IP = 0
	request.Key = 0
	request.NumWant = 50
	request.Port = 6882
	encode := request.encode()
	content.Conn.Write(encode)
	buf := make([]byte, 512)
	len, err := content.Conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return err
	}
	response := NewUdpAnnounceResponse(buf[:len])
	fmt.Println(response)
	return nil
}

func DoTracker(announce string, metaInfo *MetaInfo) {
	if announce != "" {
		url, _ := url.Parse(announce)
		switch url.Scheme {
		case "http", "https":
			// TODO UDP test
			break
			httpTracker(url, metaInfo)
		case "udp", "udp4", "udp6":
			udpTracker(url, metaInfo)
		}
	}
}

func genPeerID() ([20]byte, error) {
	peerId := [20]byte{'G', 'P', '0', '0', '1'}
	_, err := rand.Read(peerId[5:])
	if err != nil {
		return peerId, err
	}
	return peerId, nil
}

// http://bittorrent.org/beps/bep_0003.html#trackers
func httpTracker(url *url.URL, metaInfo *MetaInfo) {
	query := url.Query()
	query.Add("info_hash", string(metaInfo.InfoHash[:]))
	// Generate peer ID
	peerID, err := genPeerID()
	if err != nil {
		fmt.Println(err)
		return
	}
	query.Add("peer_id", string(peerID[:]))
	query.Add("port", "6882")
	query.Add("uploaded", "0")
	query.Add("downloaded", "0")
	query.Add("left", strconv.FormatInt(int64(metaInfo.GetTotalSize()), 10))
	url.RawQuery = query.Encode()
	request, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	var httpClient = &http.Client{
		Timeout: time.Second * time.Duration(15),
	}
	request.Header.Set("User-Agent", "gopeed/0.0.1")
	fmt.Println("request:" + url.String())
	response, err := httpClient.Do(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer response.Body.Close()
	resp, err := ioutil.ReadAll(response.Body)
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
			return
		}
	}
	fmt.Println(string(resp))
}

// http://bittorrent.org/beps/bep_0015.html
func udpTracker(url *url.URL, metaInfo *MetaInfo) {
	content := NewUdpTrackerContent(url, metaInfo)
	err := content.connect()
	if err != nil {
		fmt.Println(err)
		return
	}
	content.announce()
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
