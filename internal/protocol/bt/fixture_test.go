package bt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/merkle"
	"github.com/anacrolix/torrent/metainfo"
	"golang.org/x/time/rate"
)

// btFixture serves generated, trackerless content over a real loopback TCP peer.
// Its HTTP tracker exercises .torrent downloads; magnet links also carry the
// explicit x.pe address.
type btFixture struct {
	MetaInfo    *metainfo.MetaInfo
	Magnet      string
	TorrentFile string
	DataURI     string
	TrackerURL  string
	Files       map[string][]byte // Paths relative to the selected download directory.
	Seeder      *torrent.Client
	Torrent     *torrent.Torrent
	Limiter     *rate.Limiter
}

func newBTFixture(t testing.TB, files map[string][]byte, bytesPerSecond int) *btFixture {
	return makeBTFixture(t, files, bytesPerSecond, false, true)
}

func newBTHybridFixture(t testing.TB, payload []byte) *btFixture {
	return makeBTFixture(t, map[string][]byte{"": payload}, 0, true, true)
}

func makeBTFixture(t testing.TB, files map[string][]byte, bytesPerSecond int, v2, private bool) *btFixture {
	t.Helper()
	source := t.TempDir()
	name := "payload.bin"
	if _, single := files[""]; !single {
		name = "bundle"
	}
	root := filepath.Join(source, name)
	expected := make(map[string][]byte, len(files))
	for path, data := range files {
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
		expected[filepath.Join(name, filepath.FromSlash(path))] = append([]byte(nil), data...)
	}
	info := metainfo.Info{PieceLength: 16 * 1024}
	if err := info.BuildFromFilePath(root); err != nil {
		t.Fatal(err)
	}
	var layers map[string]string
	if v2 {
		data := files[""]
		h := merkle.NewHash()
		h.Write(data)
		root := h.Sum(nil)
		var pieceHashes []byte
		for off := 0; off < len(data); off += int(info.PieceLength) {
			end := min(off+int(info.PieceLength), len(data))
			ph := merkle.NewHash()
			ph.Write(data[off:end])
			pieceHashes = append(pieceHashes, ph.SumMinLength(nil, int(info.PieceLength))...)
		}
		info = metainfo.Info{Name: name, PieceLength: info.PieceLength, Length: int64(len(data)), Pieces: info.Pieces, MetaVersion: 2, FileTree: metainfo.FileTree{File: metainfo.FileTreeFile{Length: int64(len(data)), PiecesRoot: string(root)}}}
		layers = map[string]string{string(root): string(pieceHashes)}
	}
	// Private torrents avoid public DHT discovery once metadata is available.
	info.Private = &private
	raw, err := bencode.Marshal(&info)
	if err != nil {
		t.Fatal(err)
	}
	mi := &metainfo.MetaInfo{InfoBytes: raw, PieceLayers: layers}
	cfg := torrent.TestingConfig(t)
	cfg.DataDir = source
	cfg.Seed = true
	cfg.DisableUTP = true
	cfg.DisableIPv6 = true
	cfg.DisableWebtorrent = true
	cfg.KeepAliveTimeout = time.Minute
	cfg.MaxAllocPeerRequestDataPerConn = 1 << 20
	limiter := rate.NewLimiter(rate.Inf, 16*1024)
	if bytesPerSecond > 0 {
		limiter.SetLimit(rate.Limit(bytesPerSecond))
	}
	cfg.UploadRateLimiter = limiter
	seed, err := torrent.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { seed.Close() })
	seeded, _, err := seed.AddTorrentSpec(torrent.TorrentSpecFromMetaInfo(mi))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := seeded.VerifyDataContext(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-seeded.Complete().On():
	case <-ctx.Done():
		t.Fatal("local seeder did not finish verification")
	}
	addr := seed.ListenAddrs()[0].String()
	host, portString, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := net.ResolveTCPAddr("tcp4", net.JoinHostPort(host, portString))
	if err != nil {
		t.Fatal(err)
	}
	compact := make([]byte, 6)
	copy(compact, resolved.IP.To4())
	binary.BigEndian.PutUint16(compact[4:], uint16(resolved.Port))
	tracker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, err := bencode.Marshal(map[string]any{"interval": 3600, "peers": string(compact)})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(response)
	}))
	t.Cleanup(tracker.Close)
	// The pinned library has a hybrid tracker-dispatch assertion bug.
	// Hybrid fixtures use their explicit loopback peer rather than a tracker.
	if !v2 {
		mi.Announce = tracker.URL
	}
	var encoded bytes.Buffer
	if err := mi.Write(&encoded); err != nil {
		t.Fatal(err)
	}
	torrentFile := filepath.Join(t.TempDir(), "fixture.torrent")
	if err := os.WriteFile(torrentFile, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := mi.MagnetV2()
	if err != nil {
		t.Fatal(err)
	}
	magnet := m.String()
	u, err := url.Parse(magnet)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Add("x.pe", addr)
	u.RawQuery = q.Encode()
	return &btFixture{MetaInfo: mi, Magnet: u.String(), TorrentFile: torrentFile, DataURI: "data:application/x-bittorrent;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes()), TrackerURL: tracker.URL, Files: expected, Seeder: seed, Torrent: seeded, Limiter: limiter}
}

func btPayload(size int, label string) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i*31 + i/251) % 251)
	}
	copy(data, label)
	return data
}

func (f *btFixture) AssertFiles(t testing.TB, dir string, paths ...string) {
	t.Helper()
	if len(paths) == 0 {
		for path := range f.Files {
			paths = append(paths, path)
		}
	}
	for _, path := range paths {
		got, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
		want, ok := f.Files[path]
		if !ok {
			t.Fatalf("unknown fixture file %q", path)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("BT file %s differs: received %d bytes, want %d", path, len(got), len(want))
		}
	}
}

func (f *btFixture) FileURI() string {
	path := filepath.ToSlash(f.TorrentFile)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
