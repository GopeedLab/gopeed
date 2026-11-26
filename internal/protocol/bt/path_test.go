package bt

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

func traversalTorrentDataURI(t *testing.T, name string, path []string) string {
	t.Helper()
	info := metainfo.Info{
		Name:        name,
		PieceLength: 16 * 1024,
		Pieces:      make([]byte, 20),
		Files:       []metainfo.FileInfo{{Length: 4, Path: path}},
	}
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := (&metainfo.MetaInfo{InfoBytes: infoBytes}).Write(&buf); err != nil {
		t.Fatal(err)
	}
	return "data:application/x-bittorrent;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestFetcher_ResolveRejectsPathTraversal(t *testing.T) {
	cases := []struct {
		name string
		path []string
	}{
		{"bundle", []string{"..", "..", "pwned"}},
		{"bundle", []string{"a/../../../pwned"}},
		{"..", []string{"pwned"}},
	}
	for _, c := range cases {
		f := buildFetcher(t)
		err := f.Resolve(&base.Request{URL: traversalTorrentDataURI(t, c.name, c.path)})
		if err == nil || !strings.Contains(err.Error(), "invalid torrent file path") {
			t.Errorf("name=%q path=%q: Resolve() err = %v, want invalid torrent file path", c.name, c.path, err)
		}
	}
}

func TestCheckTorrentFilePathsAllowsNestedPaths(t *testing.T) {
	info := &metainfo.Info{
		Name:  "bundle",
		Files: []metainfo.FileInfo{{Length: 1, Path: []string{"dir", "file.bin"}}, {Length: 1, Path: []string{"top.txt"}}},
	}
	if err := checkTorrentFilePaths(info); err != nil {
		t.Fatal(err)
	}
}
