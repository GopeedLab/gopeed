package metainfo

import (
	"crypto/sha1"
	"encoding/json"
	"github.com/marksamman/bencode"
	"os"
)

type MetaInfo struct {
	// tracker URL
	// http://bittorrent.org/beps/bep_0003.html#metainfo-files
	Announce string `json:"announce"`
	// tracker URL
	// http://bittorrent.org/beps/bep_0012.html
	AnnounceList [][]string `json:"announce-list"`
	Comment      string     `json:"comment"`
	CreatedBy    string     `json:"created by"`
	CreationDate int64      `json:"creation date"`
	Encoding     string     `json:"encoding"`
	// http://bittorrent.org/beps/bep_0019.html
	UrlList []string `json:"url-list"`
	Info    *Info    `json:"info"`

	ExtraInfo `json:"-"`
}

type Info struct {
	Name        string `json:"name"`
	PieceLength uint64 `json:"piece length"`
	Pieces      []string
	Length      uint64 `json:"length"`
	Files       []File `json:"files"`
}

type File struct {
	Length uint64 `json:"length"`
	Path   string `json:"path"`
}

type ExtraInfo struct {
	InfoHash [20]byte
	FileSize uint64
}

func ParseFromFile(path string) (*MetaInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// http://bittorrent.org/beps/bep_0003.html#bencoding
	btDecode, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}
	bytes, err := json.Marshal(btDecode)
	if err != nil {
		return nil, err
	}
	var metaInfo MetaInfo
	json.Unmarshal(bytes, &metaInfo)
	if metaInfo.Info != nil {
		// encode info hash
		info := btDecode["info"].(map[string]interface{})
		infoBtEncode := bencode.Encode(info)
		metaInfo.InfoHash = sha1.Sum(infoBtEncode[:])
		// Split pieces hash
		pieces := info["pieces"].(string)
		if pieces != "" {
			metaInfo.Info.Pieces = make([]string, len(pieces)/20)
			// pieces parse
			for i := range metaInfo.Info.Pieces {
				begin := i * 20
				metaInfo.Info.Pieces[i] = pieces[begin : begin+20]
			}
		}
	}
	metaInfo.FileSize = getTotalSize(&metaInfo)
	return &metaInfo, nil
}

func getTotalSize(metaInfo *MetaInfo) uint64 {
	if metaInfo.Info != nil {
		if len(metaInfo.Info.Files) > 0 {
			var length uint64
			for _, f := range metaInfo.Info.Files {
				length += f.Length
			}
			return length
		} else {
			return metaInfo.Info.Length
		}
	}
	return 0
}
