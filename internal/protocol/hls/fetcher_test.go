package hls

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func pkcs7PadForTest(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func encryptSegment(plain, key, iv []byte) []byte {
	padded := pkcs7PadForTest(plain, aes.BlockSize)
	block, _ := aes.NewCipher(key)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, padded)
	return encrypted
}

// serveContentWithReferer is a helper handler that enforces an optional
// Referer requirement and reports request counts.
type cdnHandler struct {
	mux            *http.ServeMux
	requireReferer string
	requests       atomic.Int64
}

func (h *cdnHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.requests.Add(1)
	if h.requireReferer != "" && r.Header.Get("Referer") != h.requireReferer {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	h.mux.ServeHTTP(w, r)
}

func newTestFetcher(t *testing.T, cfg config) *Fetcher {
	t.Helper()
	fm := &FetcherManager{}
	f := fm.Build().(*Fetcher)
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		if c, ok := v.(*config); ok {
			*c = cfg
		}
	}
	f.Setup(ctl)
	return f
}

func testConfig() config {
	return config{
		SegmentConnections:    4,
		MaxRetries:            2,
		TimeoutSeconds:        5,
		PrefetchContentLength: true,
	}
}

func mustResolve(t *testing.T, f *Fetcher, playlistURL, referer string) {
	t.Helper()
	req := &base.Request{URL: playlistURL}
	if referer != "" {
		req.Extra = &ReqExtra{Header: map[string]string{"Referer": referer}}
	}
	if err := f.Resolve(req, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatalf("resolve: %v", err)
	}
}

func mustRun(t *testing.T, f *Fetcher) {
	t.Helper()
	if err := f.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := f.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
}

func statsOf(t *testing.T, f *Fetcher) Stats {
	t.Helper()
	s, ok := f.Stats().Snapshot.(*Stats)
	if !ok {
		t.Fatalf("unexpected stats snapshot type %T", f.Stats().Snapshot)
	}
	return *s
}

func readOutput(t *testing.T, f *Fetcher) []byte {
	t.Helper()
	data, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestFetcherE2E_MasterTS(t *testing.T) {
	seg := func(i int) []byte {
		return []byte(fmt.Sprintf("SEGMENT-%02d-PAYLOAD", i))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/master.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n" +
			"#EXT-X-STREAM-INF:BANDWIDTH=800000\nv1/index.m3u8\n" +
			"#EXT-X-STREAM-INF:BANDWIDTH=2000000\nv2/index.m3u8\n"))
	})
	mux.HandleFunc("/v1/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXTINF:4,\nlow-0.ts\n#EXT-X-ENDLIST\n"))
	})
	mux.HandleFunc("/v2/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("#EXTM3U\n")
		for i := 0; i < 5; i++ {
			b.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
		}
		b.WriteString("#EXT-X-ENDLIST\n")
		w.Write([]byte(b.String()))
	})
	for i := 0; i < 5; i++ {
		content := seg(i)
		path := "/v2/seg-" + strconv.Itoa(i) + ".ts"
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Write(content)
		})
	}
	server := httptest.NewServer(&cdnHandler{mux: mux})
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	mustResolve(t, f, server.URL+"/master.m3u8", "")
	// Master resolution picked the 2000000 variant.
	if f.state.MediaURL != server.URL+"/v2/index.m3u8" {
		t.Fatalf("wrong variant picked: %s", f.state.MediaURL)
	}
	// HEAD prefetch should have computed the exact total size.
	if f.meta.Res.Size != int64(5*len(seg(0))) {
		t.Fatalf("prefetched size wrong: %d", f.meta.Res.Size)
	}
	mustRun(t, f)

	var want []byte
	for i := 0; i < 5; i++ {
		want = append(want, seg(i)...)
	}
	got := readOutput(t, f)
	if !bytes.Equal(got, want) {
		t.Errorf("merged output mismatch: %q", got)
	}
	// The staging folder must be cleaned up after a successful merge; only
	// the output file remains in the task directory.
	if entries, _ := os.ReadDir(filepath.Dir(f.meta.SingleFilepath())); len(entries) != 1 {
		for _, e := range entries {
			t.Logf("leftover entry: %s", e.Name())
		}
		t.Errorf("staging dir should be cleaned up after merge")
	}
	if f.meta.Res.Size != int64(len(want)) {
		t.Errorf("final size wrong: %d", f.meta.Res.Size)
	}
}

func TestFetcherE2E_RefererEnforced(t *testing.T) {
	page := "https://player.example.com/watch/123"
	var playlist strings.Builder
	playlist.WriteString("#EXTM3U\n")
	segments := make(map[string][]byte)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("seg-%d.ts", i)
		segments["/"+name] = []byte(fmt.Sprintf("DATA-%d", i))
		playlist.WriteString(fmt.Sprintf("#EXTINF:4,\n%s\n", name))
	}
	playlist.WriteString("#EXT-X-ENDLIST\n")

	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist.String()))
	})
	for path, content := range segments {
		c := content
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Write(c)
		})
	}
	server := httptest.NewServer(&cdnHandler{mux: mux, requireReferer: page})
	defer server.Close()

	// Without Referer the resolve itself must fail (403 on the playlist).
	blocked := newTestFetcher(t, testConfig())
	req := &base.Request{URL: server.URL + "/video.m3u8"}
	if err := blocked.Resolve(req, &base.Options{Path: t.TempDir()}); err == nil {
		t.Fatal("resolve without Referer should fail")
	}

	// With the Referer set through the task extra headers it must succeed.
	allowed := newTestFetcher(t, testConfig())
	allowedReq := &base.Request{
		URL:   server.URL + "/video.m3u8",
		Extra: &ReqExtra{Header: map[string]string{"Referer": page}},
	}
	if err := allowed.Resolve(allowedReq, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	mustRun(t, allowed)
	want := []byte("DATA-0DATA-1DATA-2")
	if !bytes.Equal(readOutput(t, allowed), want) {
		t.Error("output mismatch with Referer download")
	}
}

func TestFetcherE2E_AES128KeyRotation(t *testing.T) {
	key1 := []byte("0123456789abcdef")
	key2 := []byte("fedcba9876543210")
	iv1 := []byte("aaaaaaaaaaaaaaaa")
	plain1 := []byte("ENCRYPTED-SEGMENT-ONE!!") // padded on encrypt
	plain2 := []byte("ENCRYPTED-SEGMENT-TWO!!")
	// seg-2 has no explicit IV: derived from media sequence 1.
	enc1 := encryptSegment(plain1, key1, iv1)
	enc2 := encryptSegment(plain2, key2, sequenceIV(1))

	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n" +
			"#EXT-X-MEDIA-SEQUENCE:0\n" +
			"#EXT-X-KEY:METHOD=AES-128,URI=\"key-1.bin\",IV=0x" + hexEncode(iv1) + "\n" +
			"#EXTINF:5,\nseg-1.ts\n" +
			"#EXT-X-KEY:METHOD=AES-128,URI=\"key-2.bin\"\n" +
			"#EXTINF:5,\nseg-2.ts\n" +
			"#EXT-X-ENDLIST\n"))
	})
	mux.HandleFunc("/key-1.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Write(key1)
	})
	mux.HandleFunc("/key-2.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Write(key2)
	})
	mux.HandleFunc("/seg-1.ts", func(w http.ResponseWriter, r *http.Request) {
		w.Write(enc1)
	})
	mux.HandleFunc("/seg-2.ts", func(w http.ResponseWriter, r *http.Request) {
		w.Write(enc2)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	mustResolve(t, f, server.URL+"/video.m3u8", "")
	// Ciphertext sizes overstate the final output after decryption, so an
	// encrypted plan must report an indeterminate size.
	if f.meta.Res.Size != 0 {
		t.Errorf("encrypted plan should report size 0, got %d", f.meta.Res.Size)
	}
	mustRun(t, f)

	want := append(append([]byte{}, plain1...), plain2...)
	if !bytes.Equal(readOutput(t, f), want) {
		t.Errorf("decrypted output mismatch: %q", readOutput(t, f))
	}
}

func TestFetcherE2E_ByteRange(t *testing.T) {
	data := []byte("0123456789ABCDEFGHIJ") // 20 bytes
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n" +
			"#EXT-X-BYTERANGE:10@0\nblob.bin\n" +
			"#EXT-X-BYTERANGE:10\nblob.bin\n" +
			"#EXT-X-ENDLIST\n"))
	})
	mux.HandleFunc("/blob.bin", func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		if rng == "" {
			w.Write(data)
			return
		}
		// bytes=start-end
		var start, end int
		fmt.Sscanf(rng, "bytes=%d-%d", &start, &end)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(data[start : end+1])
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	mustResolve(t, f, server.URL+"/video.m3u8", "")
	if f.meta.Res.Size != 20 {
		t.Fatalf("byterange sizes should be known from the playlist, got %d", f.meta.Res.Size)
	}
	mustRun(t, f)
	if !bytes.Equal(readOutput(t, f), data) {
		t.Errorf("byterange merge mismatch: %q", readOutput(t, f))
	}
}

func TestFetcherE2E_PauseResume(t *testing.T) {
	const total = 8
	var served atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("#EXTM3U\n")
		for i := 0; i < total; i++ {
			b.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
		}
		b.WriteString("#EXT-X-ENDLIST\n")
		w.Write([]byte(b.String()))
	})
	segments := make(map[int][]byte)
	for i := 0; i < total; i++ {
		segments[i] = []byte(fmt.Sprintf("SLOW-SEGMENT-%02d", i))
	}
	for i := 0; i < total; i++ {
		idx := i
		mux.HandleFunc(fmt.Sprintf("/seg-%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(150 * time.Millisecond)
			served.Add(1)
			w.Write(segments[idx])
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := testConfig()
	cfg.SegmentConnections = 2
	cfg.PrefetchContentLength = false
	f := newTestFetcher(t, cfg)
	mustResolve(t, f, server.URL+"/video.m3u8", "")

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	// Let a couple of segments finish, then pause.
	time.Sleep(400 * time.Millisecond)
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	doneBefore := statsOf(t, f).SegmentsDone
	if doneBefore == 0 {
		t.Fatal("expected some segments to complete before pause")
	}
	if doneBefore == total {
		t.Fatal("all segments finished before pause, test is not exercising resume")
	}

	// Resume must finish the rest and produce the full output.
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	var want []byte
	for i := 0; i < total; i++ {
		want = append(want, segments[i]...)
	}
	if !bytes.Equal(readOutput(t, f), want) {
		t.Errorf("resumed output mismatch: %q", readOutput(t, f))
	}
}

func TestFetcherE2E_StoreRestoreResume(t *testing.T) {
	const total = 6
	var served atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("#EXTM3U\n")
		for i := 0; i < total; i++ {
			b.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
		}
		b.WriteString("#EXT-X-ENDLIST\n")
		w.Write([]byte(b.String()))
	})
	segments := make(map[int][]byte)
	for i := 0; i < total; i++ {
		segments[i] = []byte(fmt.Sprintf("RESTORE-SEGMENT-%02d", i))
	}
	for i := 0; i < total; i++ {
		idx := i
		mux.HandleFunc(fmt.Sprintf("/seg-%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(120 * time.Millisecond)
			served.Add(1)
			w.Write(segments[idx])
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := testConfig()
	cfg.SegmentConnections = 2
	cfg.PrefetchContentLength = false

	f := newTestFetcher(t, cfg)
	mustResolve(t, f, server.URL+"/video.m3u8", "")
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(350 * time.Millisecond)
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	doneBefore := statsOf(t, f).SegmentsDone
	if doneBefore == 0 {
		t.Fatal("expected some segments to finish before simulated restart")
	}

	// Simulate an app restart: persist, then restore into a fresh fetcher.
	fm := &FetcherManager{}
	state, err := fm.Store(f)
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("state should be persisted")
	}
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		if c, ok := v.(*config); ok {
			*c = cfg
		}
	}
	holder := fm.Build().(*Fetcher)
	holder.Setup(ctl)
	holder.meta = &fetcher.FetcherMeta{
		Req:  f.meta.Req,
		Opts: f.meta.Opts,
		Res:  f.meta.Res,
	}
	_, builder := fm.Restore()
	rf := builder(holder.meta, state).(*Fetcher)
	// The engine calls Setup on the builder result (restoreFetcher).
	rf.Setup(ctl)
	if rf.state == nil || len(rf.state.Segments) != total {
		t.Fatal("restored state missing segment plan")
	}

	if err := rf.Start(); err != nil {
		t.Fatal(err)
	}
	if err := rf.Wait(); err != nil {
		t.Fatal(err)
	}
	if done := statsOf(t, rf).SegmentsDone; done != total {
		t.Errorf("want %d done, got %d", total, done)
	}
	var want []byte
	for i := 0; i < total; i++ {
		want = append(want, segments[i]...)
	}
	if !bytes.Equal(readOutput(t, rf), want) {
		t.Errorf("restored output mismatch: %q", readOutput(t, rf))
	}
}

func hexEncode(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, v := range b {
		out = append(out, digits[v>>4], digits[v&0x0f])
	}
	return string(out)
}
