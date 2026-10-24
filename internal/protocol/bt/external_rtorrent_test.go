package bt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// startRealRtorrent runs the rtorrent binary named by GOPEED_TEST_RTORRENT
// with a config of its own and returns the scgi:// address of its socket.
func startRealRtorrent(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("GOPEED_TEST_RTORRENT")
	if bin == "" {
		t.Skip("set GOPEED_TEST_RTORRENT to an rtorrent binary to run this test")
	}
	dir, err := os.MkdirTemp("", "rt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "rpc.sock")
	if len(sock) > 100 {
		t.Skip("the temp folder path is too long for a Unix socket")
	}
	for _, sub := range []string{"session", "default"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rc := filepath.Join(dir, "rtorrent.rc")
	conf := fmt.Sprintf(`directory.default.set = %[1]s/default
session.path.set = %[1]s/session
network.scgi.open_local = %[2]s
network.listen.port.range.set = 46900-46950
network.listen.port.random.set = yes
dht.mode.set = auto
system.daemon.set = true
`, dir, sock)
	if err := os.WriteFile(rc, []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-n", "-o", "import="+rc)
	cmd.Dir = dir
	out, _ := os.Create(filepath.Join(dir, "out.log"))
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		out.Close()
	})
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			log, _ := os.ReadFile(filepath.Join(dir, "out.log"))
			t.Fatalf("rtorrent did not open its socket: %s", log)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "scgi://" + sock
}

// TestExternalRealRtorrentTorrentFile hands a local .torrent to a real
// rtorrent whose data is already in the task folder, so rtorrent finishes
// after its hash check without any peer.
func TestExternalRealRtorrentTorrentFile(t *testing.T) {
	endpoint := startRealRtorrent(t)
	tc := newExternalTestConfig(endpoint, "")
	// File sizes are whole pieces, so no piece is shared with the file that
	// is not selected and rtorrent can finish without a peer.
	payload := map[string][]byte{
		"a.bin":     []byte(strings.Repeat("a", 2*16384)),
		"sub/b.bin": []byte(strings.Repeat("b", 16384)),
		"c.bin":     []byte(strings.Repeat("c", 2*16384)),
	}
	torrentPath, _, hash := test.WriteTorrent(t, `real, "folder"`, payload)

	// The task folder holds a.bin and c.bin only; b.bin is not selected.
	dir := t.TempDir()
	for _, p := range []string{"a.bin", "c.bin"} {
		target := filepath.Join(dir, `real, "folder"`, p)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, payload[p], 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	// Torrent order: a.bin, c.bin, sub/b.bin.
	f.Meta().Opts = &base.Options{Path: dir, SelectFiles: []int{0, 1}}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	client, _, err := f.externalClient()
	if err != nil {
		t.Fatal(err)
	}
	st, err := client.status(f.torrentDropCtx, hash)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, `real, "folder"`); st.Directory != want {
		t.Fatalf("rtorrent directory %q, want %q", st.Directory, want)
	}
	if err := waitExternal(t, f); err != nil {
		t.Fatal(err)
	}
	if p := f.Progress(); len(p) != 2 || p[0] != 2*16384 || p[1] != 2*16384 {
		t.Fatalf("progress %v, want [32768 32768]", p)
	}
	assertNoBuiltInClient(t)

	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if st, _ := client.status(f.torrentDropCtx, hash); st.Started {
		t.Fatal("pause must stop the download in rtorrent")
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if st, _ := client.status(f.torrentDropCtx, hash); !st.Started {
		t.Fatal("continue must start the download in rtorrent")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if ok, err := client.has(t.Context(), hash); err != nil || ok {
		t.Fatalf("after delete rtorrent has the download: %v %v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(dir, `real, "folder"`, "a.bin")); err != nil {
		t.Fatalf("rtorrent must keep the data: %v", err)
	}
}

// TestExternalRealRtorrentMagnet needs the network: GOPEED_TEST_RTORRENT_MAGNET
// names a well-seeded public magnet. It checks that the folder survives the
// swap from the metadata download to the real one, then removes the download.
func TestExternalRealRtorrentMagnet(t *testing.T) {
	magnet := os.Getenv("GOPEED_TEST_RTORRENT_MAGNET")
	if magnet == "" {
		t.Skip("set GOPEED_TEST_RTORRENT_MAGNET to a well-seeded public magnet to run this test")
	}
	endpoint := startRealRtorrent(t)
	tc := newExternalTestConfig(endpoint, "")
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: magnet}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	f.Meta().Opts = &base.Options{Path: dir}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- f.Wait() }()
	deadline := time.Now().Add(3 * time.Minute)
	for {
		f.mu.Lock()
		files := 0
		if f.meta.Res != nil {
			files = len(f.meta.Res.Files)
		}
		f.mu.Unlock()
		if files > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no metadata within 3 minutes")
		}
		time.Sleep(time.Second)
	}
	client, _, _ := f.externalClient()
	var st *rtorrentStatus
	for i := 0; i < 20; i++ {
		var err error
		if st, err = client.status(t.Context(), f.data.External.Hash); err != nil {
			t.Fatal(err)
		}
		if st.Started {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !st.Started || (st.Directory != dir && filepath.Dir(st.Directory) != dir) {
		t.Fatalf("real download started=%v in %q, want started in %q", st.Started, st.Directory, dir)
	}
	t.Logf("metadata arrived: %q, %d bytes, in %q", st.Name, st.SizeBytes, st.Directory)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not end after delete")
	}
}
