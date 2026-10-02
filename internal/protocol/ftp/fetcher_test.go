package ftp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/armon/go-socks5"
	"github.com/rs/zerolog"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	pftp "github.com/GopeedLab/gopeed/pkg/protocol/ftp"
)

const mib = 1 << 20

// testManager trusts the test server's certificate, the way an embedding
// application could add private roots.
func testManager(srv *FTPServer) *FetcherManager {
	fm := &FetcherManager{}
	if srv != nil {
		fm.TLSConfig = &tls.Config{RootCAs: srv.CA}
	}
	return fm
}

func testController(fm *FetcherManager) *controller.Controller {
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		data, _ := json.Marshal(fm.DefaultConfig())
		_ = json.Unmarshal(data, v)
	}
	return ctl
}

func buildFetcher(fm *FetcherManager) *Fetcher {
	f := fm.Build().(*Fetcher)
	f.Setup(testController(fm))
	return f
}

func ftpURL(srv *FTPServer, scheme, userinfo, p string) string {
	if userinfo != "" {
		userinfo += "@"
	}
	return scheme + "://" + userinfo + srv.Addr + "/" + strings.TrimPrefix(p, "/")
}

func resolveTask(t *testing.T, f *Fetcher, rawURL string, extra *pftp.OptsExtra) {
	t.Helper()
	opts := &base.Options{Path: t.TempDir()}
	if extra != nil {
		opts.Extra = extra
	}
	if err := f.Resolve(&base.Request{URL: rawURL}, opts); err != nil {
		t.Fatalf("resolve %s: %v", rawURL, err)
	}
}

func waitFetcher(t *testing.T, f fetcher.Fetcher) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- f.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(60 * time.Second):
		t.Fatal("download did not finish in 60s")
		return nil
	}
}

func startAndWait(t *testing.T, f *Fetcher) {
	t.Helper()
	if err := f.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := waitFetcher(t, f); err != nil {
		t.Fatalf("download: %v", err)
	}
}

func checkFile(t *testing.T, name, want string) {
	t.Helper()
	if got := FileSHA256(t, name); got != want {
		t.Fatalf("%s: sha256 %s, want %s", name, got, want)
	}
}

func ftpStats(t *testing.T, f *Fetcher) *pftp.Stats {
	t.Helper()
	stats, ok := f.Stats().Snapshot.(*pftp.Stats)
	if !ok {
		t.Fatalf("stats snapshot is %T, want *ftp.Stats", f.Stats().Snapshot)
	}
	return stats
}

func storeData(t *testing.T, fm *FetcherManager, f *Fetcher) []byte {
	t.Helper()
	data, err := fm.Store(f)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

// restoreFetcher rebuilds a fetcher the way the downloader does after a
// restart: the meta and the save blob both come back from JSON.
func restoreFetcher(t *testing.T, fm *FetcherManager, meta *fetcher.FetcherMeta, blob []byte) *Fetcher {
	t.Helper()
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	var restoredMeta fetcher.FetcherMeta
	if err := json.Unmarshal(metaJSON, &restoredMeta); err != nil {
		t.Fatal(err)
	}
	v, build := fm.Restore()
	if err := json.Unmarshal(blob, v); err != nil {
		t.Fatal(err)
	}
	f := build(&restoredMeta, v).(*Fetcher)
	f.Setup(testController(fm))
	return f
}

func TestResolveFile(t *testing.T) {
	for _, noMLST := range []bool{false, true} {
		t.Run(fmt.Sprintf("noMLST=%v", noMLST), func(t *testing.T) {
			srv := StartFTPServer(t, ServerOptions{NoMLST: noMLST})
			data := RandomBytes(100*1024, 1)
			srv.WriteFile(t, "pub/file name.bin", data)

			f := buildFetcher(testManager(srv))
			resolveTask(t, f, ftpURL(srv, "ftp", "", "pub/file%20name.bin"), nil)
			res := f.Meta().Res
			if res.Name != "" {
				t.Fatalf("a file must not be a folder resource, got name %q", res.Name)
			}
			if len(res.Files) != 1 {
				t.Fatalf("got %d files, want 1", len(res.Files))
			}
			file := res.Files[0]
			if file.Name != "file name.bin" || file.Path != "" || file.Size != int64(len(data)) {
				t.Fatalf("got file %+v, want name %q, empty path, size %d", file, "file name.bin", len(data))
			}
			if res.Size != int64(len(data)) {
				t.Fatalf("resource size %d, want %d", res.Size, len(data))
			}
			if file.Ctime == nil || time.Since(*file.Ctime) > time.Hour {
				t.Fatalf("modification time %v is missing or wrong", file.Ctime)
			}
		})
	}
}

func writeTree(t *testing.T, srv *FTPServer) map[string]string {
	t.Helper()
	return map[string]string{
		"a.txt":            srv.WriteFile(t, "pub/tree/a.txt", []byte("hello")),
		"sub/b.bin":        srv.WriteFile(t, "pub/tree/sub/b.bin", RandomBytes(3*mib+17, 2)),
		"sub/deeper/c.dat": srv.WriteFile(t, "pub/tree/sub/deeper/c.dat", RandomBytes(2000, 3)),
		"sub/empty.txt":    srv.WriteFile(t, "pub/tree/sub/empty.txt", nil),
	}
}

func TestResolveDir(t *testing.T) {
	for _, noMLST := range []bool{false, true} {
		for _, p := range []string{"pub/tree", "pub/tree/"} {
			t.Run(fmt.Sprintf("noMLST=%v/%s", noMLST, p), func(t *testing.T) {
				srv := StartFTPServer(t, ServerOptions{NoMLST: noMLST})
				writeTree(t, srv)
				if err := os.MkdirAll(filepath.Join(srv.Root, "pub/tree/void"), 0o755); err != nil {
					t.Fatal(err)
				}

				f := buildFetcher(testManager(srv))
				resolveTask(t, f, ftpURL(srv, "ftp", "", p), nil)
				res := f.Meta().Res
				if res.Name != "tree" {
					t.Fatalf("folder name %q, want tree", res.Name)
				}
				var got []string
				for _, file := range res.Files {
					got = append(got, fmt.Sprintf("%s|%s|%d", file.Path, file.Name, file.Size))
				}
				want := []string{
					"|a.txt|5",
					"sub|b.bin|3145745",
					"sub|empty.txt|0",
					"sub/deeper|c.dat|2000",
				}
				if !slices.Equal(got, want) {
					t.Fatalf("files\n got %q\nwant %q", got, want)
				}
				if res.Size != 5+3145745+2000 {
					t.Fatalf("resource size %d", res.Size)
				}
			})
		}
	}
}

func TestResolveDirLimits(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	srv.WriteFile(t, "deep/1/2/3/4/5/6/7/8/f.txt", []byte("ok"))
	srv.WriteFile(t, "deep9/1/2/3/4/5/6/7/8/9/f.txt", []byte("too deep"))
	for i := 0; i < 4; i++ {
		srv.WriteFile(t, fmt.Sprintf("many/f%d", i), []byte("x"))
	}

	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "", "deep"), nil)
	if got := f.Meta().Res.Files[0].Path; got != "1/2/3/4/5/6/7/8" {
		t.Fatalf("depth 8 path %q", got)
	}

	f = buildFetcher(testManager(srv))
	err := f.Resolve(&base.Request{URL: ftpURL(srv, "ftp", "", "deep9")}, &base.Options{Path: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "8 levels") {
		t.Fatalf("a tree deeper than 8 levels must fail and name the limit, got %v", err)
	}

	saved := maxWalkFiles
	maxWalkFiles = 3
	defer func() { maxWalkFiles = saved }()
	f = buildFetcher(testManager(srv))
	err = f.Resolve(&base.Request{URL: ftpURL(srv, "ftp", "", "many")}, &base.Options{Path: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "3 files") {
		t.Fatalf("a tree over the file cap must fail and name the limit, got %v", err)
	}
}

func TestDownloadPlain(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	data := RandomBytes(3*mib+12345, 4)
	want := srv.WriteFile(t, "pub/plain.bin", data)

	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "", "pub/plain.bin"), &pftp.OptsExtra{Connections: 4})
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)
	if got := f.Progress().TotalDownloaded(); got != int64(len(data)) {
		t.Fatalf("progress %d, want %d", got, len(data))
	}
	stats := ftpStats(t, f)
	if len(stats.Connections) == 0 || len(stats.Connections) > 4 {
		t.Fatalf("got %d connections in stats", len(stats.Connections))
	}
	var sum int64
	for _, c := range stats.Connections {
		sum += c.Downloaded
		if !c.Completed || c.Failed {
			t.Fatalf("connection stats %+v after success", c)
		}
	}
	if sum != int64(len(data)) {
		t.Fatalf("connections downloaded %d bytes, want %d", sum, len(data))
	}
}

func TestDownloadDir(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	want := writeTree(t, srv)

	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "", "pub/tree"), &pftp.OptsExtra{Connections: 2})
	startAndWait(t, f)
	root := f.Meta().FolderPath()
	if root != filepath.Join(f.Meta().Opts.Path, "tree") {
		t.Fatalf("folder path %q", root)
	}
	for rel, sum := range want {
		checkFile(t, filepath.Join(root, filepath.FromSlash(rel)), sum)
	}
}

func testDownloadTLS(t *testing.T, mode string, schemes map[string]string) {
	srv := StartFTPServer(t, ServerOptions{TLS: mode})
	data := RandomBytes(2*mib+999, 5)
	want := srv.WriteFile(t, "secure.bin", data)
	for scheme, optTLS := range schemes {
		t.Run(scheme+"+"+optTLS, func(t *testing.T) {
			f := buildFetcher(testManager(srv))
			resolveTask(t, f, ftpURL(srv, scheme, "", "secure.bin"), &pftp.OptsExtra{Connections: 2, TLS: optTLS})
			startAndWait(t, f)
			checkFile(t, f.Meta().SingleFilepath(), want)
		})
	}
}

func TestDownloadExplicitTLS(t *testing.T) {
	testDownloadTLS(t, pftp.TLSExplicit, map[string]string{
		"ftpes": "",
		"ftp":   pftp.TLSExplicit,
	})
}

func TestDownloadImplicitTLS(t *testing.T) {
	testDownloadTLS(t, pftp.TLSImplicit, map[string]string{
		"ftps": "",
		"ftp":  pftp.TLSImplicit,
	})
}

func TestBadCertRejectedByDefault(t *testing.T) {
	for mode, scheme := range map[string]string{pftp.TLSExplicit: "ftpes", pftp.TLSImplicit: "ftps"} {
		t.Run(mode, func(t *testing.T) {
			srv := StartFTPServer(t, ServerOptions{TLS: mode})
			want := srv.WriteFile(t, "f.bin", RandomBytes(4096, 6))
			rawURL := ftpURL(srv, scheme, "", "f.bin")

			// No injected roots: the self-signed certificate is unknown.
			f := buildFetcher(testManager(nil))
			err := f.Resolve(&base.Request{URL: rawURL}, &base.Options{Path: t.TempDir()})
			var verifyErr *tls.CertificateVerificationError
			if !errors.As(err, &verifyErr) {
				t.Fatalf("want a certificate verification error, got %v", err)
			}

			// The request's existing skipVerifyCert switch accepts it.
			f = buildFetcher(testManager(nil))
			if err := f.Resolve(&base.Request{URL: rawURL, SkipVerifyCert: true}, &base.Options{Path: t.TempDir()}); err != nil {
				t.Fatalf("skipVerifyCert resolve: %v", err)
			}
			startAndWait(t, f)
			checkFile(t, f.Meta().SingleFilepath(), want)
		})
	}
}

func TestSegmentsUseRest(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	srv.ReadDelay.Store(int64(time.Millisecond))
	size := 8 * mib
	want := srv.WriteFile(t, "big.bin", RandomBytes(size, 7))

	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "", "big.bin"), &pftp.OptsExtra{Connections: 4})
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)

	rests := srv.RestOffsets()
	for _, off := range []int64{2 * mib, 4 * mib, 6 * mib} {
		if !slices.Contains(rests, off) {
			t.Fatalf("REST offsets %v miss the segment start %d", rests, off)
		}
	}
	for _, off := range rests {
		if off <= 0 || off >= int64(size) {
			t.Fatalf("REST offset %d out of range", off)
		}
	}
	if got := len(ftpStats(t, f).Connections); got != 4 {
		t.Fatalf("stats report %d connections, want 4", got)
	}
}

func TestLoginCapFallsBack(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{MaxLogins: 1})
	want := srv.WriteFile(t, "capped.bin", RandomBytes(4*mib+5, 8))

	fm := testManager(srv)
	f := buildFetcher(fm)
	resolveTask(t, f, ftpURL(srv, "ftp", "", "capped.bin"), &pftp.OptsExtra{Connections: 4})
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)

	if srv.Refused() == 0 {
		t.Fatal("the server never refused a login, so the cap was not exercised")
	}
	if got := len(ftpStats(t, f).Connections); got != 1 {
		t.Fatalf("stats report %d connections, want 1", got)
	}
	var saved fetcherData
	if err := json.Unmarshal(storeData(t, fm, f), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.MaxConns != 1 {
		t.Fatalf("saved login ceiling %d, want 1", saved.MaxConns)
	}
}

func TestNoRestSequential(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{NoRest: true})
	srv.ReadDelay.Store(int64(time.Millisecond))
	want := srv.WriteFile(t, "norest.bin", RandomBytes(4*mib+3, 9))

	fm := testManager(srv)
	f := buildFetcher(fm)
	resolveTask(t, f, ftpURL(srv, "ftp", "", "norest.bin"), &pftp.OptsExtra{Connections: 4})
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)

	if rests := srv.RestOffsets(); len(rests) != 0 {
		t.Fatalf("the server cannot have seen a REST, got %v", rests)
	}
	var saved fetcherData
	if err := json.Unmarshal(storeData(t, fm, f), &saved); err != nil {
		t.Fatal(err)
	}
	if !saved.NoRest {
		t.Fatal("the missing REST support was not recorded")
	}
}

// waitProgress waits until the fetcher has between lo and hi bytes.
func waitProgress(t *testing.T, f *Fetcher, lo, hi int64) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		got := f.Progress().TotalDownloaded()
		if got >= lo {
			if got > hi {
				t.Fatalf("progress %d passed %d before the pause; slow the server down", got, hi)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("progress stayed under %d", lo)
}

func TestResumeAfterKill(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	srv.ReadDelay.Store(int64(3 * time.Millisecond))
	size := int64(4*mib + 777)
	want := srv.WriteFile(t, "resume.bin", RandomBytes(int(size), 10))

	fm := testManager(srv)
	f := buildFetcher(fm)
	resolveTask(t, f, ftpURL(srv, "ftp", "", "resume.bin"), &pftp.OptsExtra{Connections: 2})
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	waitProgress(t, f, size/5, size*4/5)
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	blob := storeData(t, fm, f)

	var saved fetcherData
	if err := json.Unmarshal(blob, &saved); err != nil {
		t.Fatal(err)
	}
	var savedTotal int64
	resumeAt := map[int64]bool{}
	for _, seg := range saved.Segments {
		savedTotal += seg.Downloaded
		if seg.Begin+seg.Downloaded <= seg.End {
			resumeAt[seg.Begin+seg.Downloaded] = true
		}
	}
	if savedTotal == 0 || savedTotal >= size {
		t.Fatalf("saved progress %d of %d", savedTotal, size)
	}

	before := len(srv.RetrOffsets())
	srv.ReadDelay.Store(0)
	restored := restoreFetcher(t, fm, f.Meta(), blob)
	if got := restored.Progress().TotalDownloaded(); got != savedTotal {
		t.Fatalf("restored progress %d, want %d", got, savedTotal)
	}
	startAndWait(t, restored)
	checkFile(t, restored.Meta().SingleFilepath(), want)

	resumed := srv.RetrOffsets()[before:]
	if len(resumed) == 0 {
		t.Fatal("no transfer after resume")
	}
	for _, off := range resumed {
		if !resumeAt[off] {
			t.Fatalf("resumed RETR at %d, want one of the saved offsets %v", off, resumeAt)
		}
	}
}

func TestRetryAfterDroppedConnection(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	want := srv.WriteFile(t, "drop.bin", RandomBytes(3*mib, 11))
	srv.DropAfter.Store(mib + 100)

	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "", "drop.bin"), &pftp.OptsExtra{Connections: 1})
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)

	offsets := srv.RetrOffsets()
	if len(offsets) != 2 || offsets[0] != 0 || offsets[1] <= 0 || offsets[1] > mib+100 {
		t.Fatalf("RETR offsets %v, want a restart at the dropped offset", offsets)
	}
	if got := ftpStats(t, f).Connections[0].RetryTimes; got != 1 {
		t.Fatalf("retry count %d, want 1", got)
	}
}

func TestPercentEncodedPassword(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{User: "u", Pass: "p@ss"})
	want := srv.WriteFile(t, "secret.bin", RandomBytes(5000, 12))

	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "u:p%40ss", "secret.bin"), nil)
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)

	f = buildFetcher(testManager(srv))
	err := f.Resolve(&base.Request{URL: ftpURL(srv, "ftp", "u:wrong", "secret.bin")}, &base.Options{Path: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "530") {
		t.Fatalf("a wrong password must fail with 530, got %v", err)
	}
}

func TestOldSaveFormatResumes(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	data := RandomBytes(2*mib+1, 13)
	want := srv.WriteFile(t, "old.bin", data)

	fm := testManager(srv)
	f := buildFetcher(fm)
	resolveTask(t, f, ftpURL(srv, "ftp", "", "old.bin"), &pftp.OptsExtra{Connections: 1})

	// A save blob that has only the segment fields every version writes.
	const done = 700000
	blob := []byte(fmt.Sprintf(`{"Segments":[{"Begin":0,"End":%d,"Downloaded":%d}]}`, len(data)-1, done))
	name := f.Meta().SingleFilepath()
	if err := os.WriteFile(name, append(data[:done:done], make([]byte, len(data)-done)...), 0o644); err != nil {
		t.Fatal(err)
	}

	restored := restoreFetcher(t, fm, f.Meta(), blob)
	if got := restored.Progress().TotalDownloaded(); got != done {
		t.Fatalf("restored progress %d, want %d", got, done)
	}
	startAndWait(t, restored)
	checkFile(t, name, want)
	if got := srv.RetrOffsets(); !slices.Equal(got, []int64{done}) {
		t.Fatalf("RETR offsets %v, want [%d]", got, done)
	}
}

func TestDownloadThroughSOCKS5(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{TLS: pftp.TLSExplicit})
	want := srv.WriteFile(t, "proxied.bin", RandomBytes(2*mib+1, 14))

	var dials atomic.Int32
	proxySrv, err := socks5.New(&socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dials.Add(1)
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go proxySrv.Serve(ln)

	f := buildFetcher(testManager(srv))
	req := &base.Request{
		URL: ftpURL(srv, "ftpes", "", "proxied.bin"),
		Proxy: &base.RequestProxy{
			Mode:   base.RequestProxyModeCustom,
			Scheme: "socks5",
			Host:   ln.Addr().String(),
		},
	}
	if err := f.Resolve(req, &base.Options{Path: t.TempDir(), Extra: &pftp.OptsExtra{Connections: 2}}); err != nil {
		t.Fatal(err)
	}
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)
	// Resolve's control and listing, then per worker one control and one
	// data connection at least: every one of them went through the proxy.
	if got := dials.Load(); got < 4 {
		t.Fatalf("the proxy saw %d connections, want every control and data connection", got)
	}
}

func TestHTTPProxyFallsBackToDirect(t *testing.T) {
	srv := StartFTPServer(t, ServerOptions{})
	want := srv.WriteFile(t, "direct.bin", RandomBytes(100000, 15))

	var logs bytes.Buffer
	logger := zerolog.New(&logs)
	fm := testManager(srv)
	f := fm.Build().(*Fetcher)
	ctl := testController(fm)
	ctl.Logger = &logger
	f.Setup(ctl)
	req := &base.Request{
		URL: ftpURL(srv, "ftp", "", "direct.bin"),
		Proxy: &base.RequestProxy{
			Mode:   base.RequestProxyModeCustom,
			Scheme: "http",
			Host:   "127.0.0.1:1",
			Usr:    "proxyuser",
			Pwd:    "proxysecret",
		},
	}
	if err := f.Resolve(req, &base.Options{Path: t.TempDir(), Extra: &pftp.OptsExtra{Connections: 2}}); err != nil {
		t.Fatal(err)
	}
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)

	out := logs.String()
	if n := strings.Count(out, "HTTP proxy"); n != 1 {
		t.Fatalf("want exactly one HTTP proxy warning, got %d in %q", n, out)
	}
	if strings.Contains(out, "proxysecret") {
		t.Fatalf("the proxy password leaked into the log: %q", out)
	}
}

func TestTorrentNameIsFile(t *testing.T) {
	fm := testManager(nil)
	for _, u := range []string{"ftp://h/x.torrent", "ftps://h/x.torrent", "ftpes://h/x.torrent", "FTP://h/v.m3u8"} {
		matched := false
		for _, filter := range fm.Filters() {
			if filter.Match(u) {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("%s is not matched by the FTP filters", u)
		}
	}
	if fm.Name() != "ftp" {
		t.Fatalf("manager name %q", fm.Name())
	}

	srv := StartFTPServer(t, ServerOptions{})
	want := srv.WriteFile(t, "x.torrent", []byte("d8:announce0:e"))
	f := buildFetcher(testManager(srv))
	resolveTask(t, f, ftpURL(srv, "ftp", "", "x.torrent"), nil)
	res := f.Meta().Res
	if res.Name != "" || len(res.Files) != 1 || res.Files[0].Name != "x.torrent" {
		t.Fatalf("x.torrent must resolve to one FTP file, got name %q and %d files", res.Name, len(res.Files))
	}
	startAndWait(t, f)
	checkFile(t, f.Meta().SingleFilepath(), want)
}

func TestParseNameHidesCredentials(t *testing.T) {
	fm := testManager(nil)
	for u, want := range map[string]string{
		"ftp://u:p%40ss@example.com/pub/a.iso": "a.iso",
		"ftp://u:p%40ss@example.com/":          "example.com",
		"ftps://u:p%40ss@example.com":          "example.com",
		"ftp://u:p%40ss@example.com/pub/dir/":  "dir",
	} {
		if got := fm.ParseName(u); got != want {
			t.Fatalf("ParseName(%q) = %q, want %q", u, got, want)
		}
	}
}

func TestUnsafeEntryNames(t *testing.T) {
	for name, ok := range map[string]bool{
		"a.txt": true, "..a": true, "a b": true,
		"": false, ".": false, "..": false, "a/b": false, `a\b`: false, "../x": false, "x\x00y": false,
	} {
		if got := safeEntryName(name); got != ok {
			t.Fatalf("safeEntryName(%q) = %v, want %v", name, got, ok)
		}
	}
}
