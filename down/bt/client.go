package bt

import (
	"crypto/rand"
	"github.com/monkeyWie/gopeed/down/bt/metainfo"
	"github.com/monkeyWie/gopeed/down/bt/torrent"
	"net/url"
)

type Client struct {
	PeerID [20]byte

	// torrents []torrent.Torrent
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
	var metaInfo *metainfo.MetaInfo
	switch parse.Scheme {
	case "magnet":
	// TODO 磁力链接解析
	default:
		metaInfo, err = metainfo.ParseFromFile(rawurl)
		break
	}
	if err != nil {
		return err
	}

	torrent := torrent.Torrent{
		client:   client,
		MetaInfo: metaInfo,
	}
	client.torrents = append(client.torrents, torrent)

	torrent.Download()

	return nil
}
