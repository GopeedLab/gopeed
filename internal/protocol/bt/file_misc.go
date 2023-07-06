// modify from github.com/anacrolix/torrent/storage/file_misc.go

package bt

import "github.com/anacrolix/torrent/metainfo"

type requiredLength struct {
	fileIndex int
	length    int64
}

func extentCompleteRequiredLengths(info *metainfo.Info, off, n int64) (ret []requiredLength) {
	if n == 0 {
		return
	}
	for i, fi := range info.UpvertedFiles() {
		if off >= fi.Length {
			off -= fi.Length
			continue
		}
		n1 := n
		if off+n1 > fi.Length {
			n1 = fi.Length - off
		}
		ret = append(ret, requiredLength{
			fileIndex: i,
			length:    off + n1,
		})
		n -= n1
		if n == 0 {
			return
		}
		off = 0
	}
	return
}
