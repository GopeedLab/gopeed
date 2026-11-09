package download

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/ftp"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// TestTorrentNameIsFile checks the default manager order: an FTP URL whose
// name looks like a torrent or a playlist is still an FTP download.
func TestTorrentNameIsFile(t *testing.T) {
	d := NewDownloader(nil)
	for u, want := range map[string]string{
		"ftp://h/x.torrent":    "ftp",
		"ftp://h/v.m3u8":       "ftp",
		"ftps://h/x.torrent":   "ftp",
		"ftpes://h/a.bin":      "ftp",
		"https://h/v.m3u8":     "hls",
		"https://h/x.torrent":  "http",
		"magnet:?xt=urn:btih:": "bt",
	} {
		fm, err := d.parseFm(u)
		if err != nil {
			t.Fatalf("%s: %v", u, err)
		}
		if fm.Name() != want {
			t.Fatalf("%s goes to %q, want %q", u, fm.Name(), want)
		}
	}
}

func TestPostDownloadExtraAcceptsFTP(t *testing.T) {
	yes := true
	file := &base.Resource{Files: []*base.FileInfo{{Name: "a.zip"}}}
	folder := &base.Resource{Name: "dir", Files: []*base.FileInfo{{Name: "a.zip"}}}

	httpExtra := &http.OptsExtra{AutoExtract: &yes}
	if got := postDownloadExtra(&fetcher.FetcherMeta{Res: file, Opts: &base.Options{Extra: httpExtra}}); got != httpExtra {
		t.Fatalf("HTTP options must pass through unchanged, got %+v", got)
	}

	ftpExtra := &ftp.OptsExtra{Connections: 4, AutoTorrent: &yes, DeleteTorrentAfterDownload: &yes, AutoExtract: &yes, ArchivePassword: "pw", DeleteAfterExtract: true}
	got := postDownloadExtra(&fetcher.FetcherMeta{Res: file, Opts: &base.Options{Extra: ftpExtra}})
	if got == nil || got.AutoTorrent != &yes || got.DeleteTorrentAfterDownload != &yes || got.AutoExtract != &yes ||
		got.ArchivePassword != "pw" || !got.DeleteAfterExtract {
		t.Fatalf("FTP options were not carried over: %+v", got)
	}
	if got := postDownloadExtra(&fetcher.FetcherMeta{Res: folder, Opts: &base.Options{Extra: ftpExtra}}); got != nil {
		t.Fatalf("an FTP folder has no single file to extract, got %+v", got)
	}
	if got := postDownloadExtra(&fetcher.FetcherMeta{Res: file, Opts: &base.Options{}}); got != nil {
		t.Fatalf("no options, got %+v", got)
	}
}
