package test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/GopeedLab/gopeed/internal/xmlrpc"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

// FakeRtorrent answers the XML-RPC calls the hand-off makes, the way
// rtorrent 0.16 does: unknown hashes are faults, a magnet is a metadata
// download until FinishMetadata replaces it with the real one, stopped and in
// the default folder.
type FakeRtorrent struct {
	t          testing.TB
	mu         sync.Mutex
	defaultDir string
	downloads  map[string]*FakeDownload
	calls      []string
}

type FakeDownload struct {
	Name      string
	Meta      bool
	Started   bool
	Directory string // as set; multi-file downloads report <directory>/<name>
	Multi     bool
	Files     []*FakeFile
	Chunk     int64
	UpTotal   int64
}

type FakeFile struct {
	Path      string
	Size      int64
	Priority  int64
	Completed int64 // chunks
	Chunks    int64
}

// NewFakeRtorrent returns an empty fake.
func NewFakeRtorrent(t testing.TB) *FakeRtorrent {
	return &FakeRtorrent{
		t:          t,
		defaultDir: "/rtorrent/default",
		downloads:  map[string]*FakeDownload{},
	}
}

// ServeSCGI listens on a TCP port and returns the scgi:// address.
func (r *FakeRtorrent) ServeSCGI(t testing.TB) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go r.acceptSCGI(l)
	return "scgi://" + l.Addr().String()
}

// ServeSCGIUnix listens on a Unix socket and returns the scgi:// address.
func (r *FakeRtorrent) ServeSCGIUnix(t testing.TB, path string) string {
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go r.acceptSCGI(l)
	return "scgi://" + path
}

func (r *FakeRtorrent) acceptSCGI(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			br := bufio.NewReader(conn)
			lenText, err := br.ReadString(':')
			if err != nil {
				return
			}
			n, _ := strconv.Atoi(strings.TrimSuffix(lenText, ":"))
			headers := make([]byte, n+1)
			if _, err := io.ReadFull(br, headers); err != nil {
				return
			}
			fields := strings.Split(string(headers[:n]), "\x00")
			length := 0
			for i := 0; i+1 < len(fields); i += 2 {
				if fields[i] == "CONTENT_LENGTH" {
					length, _ = strconv.Atoi(fields[i+1])
				}
			}
			body := make([]byte, length)
			if _, err := io.ReadFull(br, body); err != nil {
				return
			}
			out := r.handle(body)
			fmt.Fprintf(conn, "Status: 200 OK\r\nContent-Type: text/xml\r\nContent-Length: %d\r\n\r\n", len(out))
			conn.Write(out)
		}()
	}
}

// ServeHTTP serves XML-RPC over HTTP with basic auth, the way a web server in
// front of rtorrent does.
func (r *FakeRtorrent) ServeHTTPEndpoint(t testing.TB, user, pass string) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		u, p, ok := req.BasicAuth()
		if !ok || u != user || p != pass {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(req.Body)
		w.Header().Set("Content-Type", "text/xml")
		w.Write(r.handle(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/RPC2"
}

func (r *FakeRtorrent) handle(body []byte) []byte {
	method, params, err := xmlrpc.DecodeCall(body)
	if err != nil {
		return xmlrpc.EncodeFault(-500, err.Error())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.dispatch(method, params)
	if err != nil {
		return xmlrpc.EncodeFault(-500, err.Error())
	}
	out, err := xmlrpc.EncodeResponse(v)
	if err != nil {
		return xmlrpc.EncodeFault(-500, err.Error())
	}
	return out
}

var errFakeNotFound = errors.New("invalid parameters: info-hash not found")

func (r *FakeRtorrent) dispatch(method string, params []any) (any, error) {
	if method != "system.multicall" {
		r.calls = append(r.calls, method)
	}
	str := func(i int) string {
		if i < len(params) {
			s, _ := params[i].(string)
			return s
		}
		return ""
	}
	switch method {
	case "system.multicall":
		calls, _ := params[0].([]any)
		out := make([]any, 0, len(calls))
		for _, c := range calls {
			m := c.(map[string]any)
			v, err := r.dispatch(m["methodName"].(string), m["params"].([]any))
			if err != nil {
				out = append(out, map[string]any{"faultCode": -500, "faultString": err.Error()})
			} else {
				out = append(out, []any{v})
			}
		}
		return out, nil
	case "load.raw":
		data, _ := params[1].([]byte)
		mi, err := metainfo.Load(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		r.add(mi, false)
		return int64(0), nil
	case "load.normal":
		m, err := metainfo.ParseMagnetUri(str(1))
		if err != nil {
			return nil, err
		}
		hash := strings.ToUpper(m.InfoHash.HexString())
		r.downloads[hash] = &FakeDownload{Name: hash + ".meta", Meta: true, Directory: "/rtorrent/session", Files: []*FakeFile{{Path: hash + ".meta", Size: 1, Priority: 1, Chunks: 1}}, Chunk: 1}
		return int64(0), nil
	case "f.priority.set":
		hash, index, _ := strings.Cut(str(0), ":f")
		d, ok := r.downloads[hash]
		if !ok {
			return nil, errFakeNotFound
		}
		i, _ := strconv.Atoi(index)
		if i < 0 || i >= len(d.Files) {
			return nil, errors.New("invalid file index")
		}
		d.Files[i].Priority = asInt(params[1])
		return int64(0), nil
	}
	d, ok := r.downloads[str(0)]
	if !ok {
		return nil, errFakeNotFound
	}
	switch method {
	case "d.hash":
		return str(0), nil
	case "d.directory.set":
		if d.Started {
			return nil, errors.New("cannot change the directory of an open download")
		}
		d.Directory = str(1)
		return int64(0), nil
	case "d.update_priorities":
		return int64(0), nil
	case "d.start":
		d.Started = true
		return int64(0), nil
	case "d.stop":
		d.Started = false
		return int64(0), nil
	case "d.erase":
		delete(r.downloads, str(0))
		return int64(0), nil
	case "d.name":
		return d.Name, nil
	case "d.is_meta":
		return boolInt(d.Meta), nil
	case "d.state":
		return boolInt(d.Started), nil
	case "d.is_active":
		return boolInt(d.Started), nil
	case "d.hashing":
		return int64(0), nil
	case "d.complete":
		return boolInt(d.chunksLeft(false) == 0), nil
	case "d.completed_bytes":
		var n int64
		for _, f := range d.Files {
			n += f.Size * f.Completed / max(f.Chunks, 1)
		}
		return n, nil
	case "d.size_bytes":
		var n int64
		for _, f := range d.Files {
			n += f.Size
		}
		return n, nil
	case "d.wanted_chunks":
		return d.chunksLeft(true), nil
	case "d.size_chunks":
		var n int64
		for _, f := range d.Files {
			n += f.Chunks
		}
		return n, nil
	case "d.down.rate", "d.up.rate", "d.ratio", "d.peers_connected", "d.peers_complete":
		return int64(0), nil
	case "d.up.total":
		return d.UpTotal, nil
	case "d.message":
		return "", nil
	case "d.directory":
		if d.Multi {
			return filepath.Join(d.Directory, d.Name), nil
		}
		return d.Directory, nil
	case "d.is_multi_file":
		return boolInt(d.Multi), nil
	case "f.multicall":
		rows := make([]any, 0, len(d.Files))
		for _, f := range d.Files {
			row := []any{}
			for _, field := range params[2:] {
				switch field.(string) {
				case "f.path=":
					row = append(row, f.Path)
				case "f.size_bytes=":
					row = append(row, f.Size)
				case "f.completed_chunks=":
					row = append(row, f.Completed)
				case "f.size_chunks=":
					row = append(row, f.Chunks)
				case "f.priority=":
					row = append(row, f.Priority)
				}
			}
			rows = append(rows, row)
		}
		return rows, nil
	}
	return nil, fmt.Errorf("fake rtorrent: method %s is not handled", method)
}

func (d *FakeDownload) chunksLeft(wantedOnly bool) int64 {
	var n int64
	for _, f := range d.Files {
		if wantedOnly && f.Priority == 0 {
			continue
		}
		n += f.Chunks - f.Completed
	}
	return n
}

// Add puts a torrent into the fake as if a user had added it in rtorrent.
func (r *FakeRtorrent) Add(mi *metainfo.MetaInfo, started bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.add(mi, started)
}

// Remove erases a download behind the client's back.
func (r *FakeRtorrent) Remove(hash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.downloads, hash)
}

// SetUpTotal sets the bytes a download has uploaded.
func (r *FakeRtorrent) SetUpTotal(hash string, n int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.downloads[hash].UpTotal = n
}

func (r *FakeRtorrent) add(mi *metainfo.MetaInfo, started bool) *FakeDownload {
	info, err := mi.UnmarshalInfo()
	if err != nil {
		r.t.Fatal(err)
	}
	hash := strings.ToUpper(mi.HashInfoBytes().HexString())
	d := &FakeDownload{Name: info.Name, Directory: r.defaultDir, Multi: info.IsDir(), Chunk: info.PieceLength, Started: started}
	for _, f := range info.UpvertedFiles() {
		p := strings.Join(f.Path, "/")
		if !d.Multi {
			p = info.Name
		}
		d.Files = append(d.Files, &FakeFile{Path: p, Size: f.Length, Priority: 1, Chunks: (f.Length + info.PieceLength - 1) / info.PieceLength})
	}
	r.downloads[hash] = d
	return d
}

// FinishMetadata delivers a magnet's metadata: the metadata download is
// replaced by the real one, stopped and in the default folder.
func (r *FakeRtorrent) FinishMetadata(hash string, mi *metainfo.MetaInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.add(mi, false)
}

// Complete marks every file of the download complete.
func (r *FakeRtorrent) Complete(hash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.downloads[hash].Files {
		f.Completed = f.Chunks
	}
}

// CompleteFile marks one file complete.
func (r *FakeRtorrent) CompleteFile(hash string, index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f := r.downloads[hash].Files[index]
	f.Completed = f.Chunks
}

// Download returns a copy of the download, or nil.
func (r *FakeRtorrent) Download(hash string) *FakeDownload {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.downloads[hash]
	if !ok {
		return nil
	}
	c := *d
	c.Files = make([]*FakeFile, len(d.Files))
	for i, f := range d.Files {
		file := *f
		c.Files[i] = &file
	}
	return &c
}

// Called counts the calls of a method, the calls inside system.multicall
// included.
func (r *FakeRtorrent) Called(method string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.calls {
		if c == method {
			n++
		}
	}
	return n
}

func asInt(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case bool:
		if t {
			return 1
		}
	}
	return 0
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// WriteTorrent writes files under a temp folder named name and returns
// the .torrent path, its metainfo and the info hash in rtorrent's form. A
// single file is passed with the empty path.
func WriteTorrent(t testing.TB, name string, files map[string][]byte) (string, *metainfo.MetaInfo, string) {
	t.Helper()
	src := t.TempDir()
	root := filepath.Join(src, name)
	for p, data := range files {
		target := root
		if p != "" {
			target = filepath.Join(root, filepath.FromSlash(p))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	info := metainfo.Info{PieceLength: 16 * 1024}
	if err := info.BuildFromFilePath(root); err != nil {
		t.Fatal(err)
	}
	mi := &metainfo.MetaInfo{}
	var err error
	if mi.InfoBytes, err = bencode.Marshal(info); err != nil {
		t.Fatal(err)
	}
	torrentPath := filepath.Join(src, name+".torrent")
	f, err := os.Create(torrentPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := mi.Write(f); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return torrentPath, mi, strings.ToUpper(mi.HashInfoBytes().HexString())
}
