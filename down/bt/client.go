package bt

import (
	"crypto/rand"
	"net/url"
)

type Client struct {
	PeerID [20]byte

	torrents []Torrent
}

func NewClient() *Client {
	client := &Client{}

	// 生成Peer ID，规则为前三位固定字母(-GP)+SemVer(xyz)+End(-),后面随机生成
	// 参考：https://wiki.theory.org/index.php/BitTorrentSpecification#peer_id
	peerID := [20]byte{'-', 'G', 'P', '0', '0', '1', '-'}
	_, err := rand.Read(peerID[7:])
	if err != nil {
		panic(err)
	}
	client.PeerID = peerID

	return client
}

func (client *Client) AddTorrent(rawurl string) error {
	parse, err := url.Parse(rawurl)
	if err != nil {
		return err
	}
	var metaInfo *MetaInfo
	switch parse.Scheme {
	case "magnet":
	// TODO 磁力链接解析
	default:
		metaInfo, err = ParseFromFile(rawurl)
		break
	}
	if err != nil {
		return err
	}

	torrent := Torrent{
		client:   client,
		MetaInfo: metaInfo,
	}
	client.torrents = append(client.torrents, torrent)

	torrent.Download()

	return nil
}
