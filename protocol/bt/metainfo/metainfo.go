package metainfo

import (
	"crypto/sha1"
	"encoding/json"
	"os"

	"github.com/marksamman/bencode"
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

	infoHash    [20]byte
	totalSize   uint64
	fileDetails []FileDetail
}

type Info struct {
	Name        string `json:"name"`
	PieceLength int    `json:"piece length"`
	Pieces      [][20]byte
	// 单个文件时，length为该文件的长度
	Length uint64 `json:"length"`
	// 多个文件
	Files []File `json:"files"`
}

type File struct {
	Length uint64   `json:"length"`
	Path   []string `json:"path"`
}

type FileDetail struct {
	Length uint64
	Path   []string
	Begin  int64
	End    int64
}

type PieceMapping struct {
	file  int
	begin int64
	end   int64
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
		metaInfo.infoHash = sha1.Sum(infoBtEncode[:])
		// Split pieces hash
		pieces := info["pieces"].(string)
		if pieces != "" {
			metaInfo.Info.Pieces = make([][20]byte, len(pieces)/20)
			// pieces parse
			for i := range metaInfo.Info.Pieces {
				begin := i * 20
				metaInfo.Info.Pieces[i] = [20]byte{}
				copy(metaInfo.Info.Pieces[i][:], pieces[begin:begin+20])
			}
		}
	}
	calcFileSize(&metaInfo)
	return &metaInfo, nil
}

// 获取某个分片的大小
func (metaInfo *MetaInfo) GetPieceLength(index int) int {
	// 是否为最后一个分片
	if index == len(metaInfo.Info.Pieces)-1 {
		size := metaInfo.totalSize % uint64(metaInfo.Info.PieceLength)
		if size > 0 {
			return int(size)
		}
	}
	return metaInfo.Info.PieceLength
}

// 获取所有文件的大小
func (metaInfo *MetaInfo) GetTotalSize() uint64 {
	return metaInfo.totalSize
}

// 获取info hash
func (metaInfo *MetaInfo) GetInfoHash() [20]byte {
	return metaInfo.infoHash
}

// 获取所有文件的开始偏移字节
func (metaInfo *MetaInfo) GetFileDetails() []FileDetail {
	return metaInfo.fileDetails
}

// 计算所有文件的大小
func calcFileSize(metaInfo *MetaInfo) {
	if metaInfo.Info != nil {
		fileCount := len(metaInfo.Info.Files)
		if fileCount > 0 {
			metaInfo.fileDetails = make([]FileDetail, fileCount)
			var length uint64
			for i, f := range metaInfo.Info.Files {
				metaInfo.fileDetails[i] = FileDetail{
					Length: f.Length,
					Path:   f.Path,
					Begin:  int64(length),
					End:    int64(length + f.Length),
				}
				length += f.Length
			}
			metaInfo.totalSize = length
		} else {
			metaInfo.totalSize = metaInfo.Info.Length
		}
	}
}
