package download

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

// hlsTestCdn spins up a master-playlist HLS site whose segments require a
// Referer header (anti-hotlink) and returns the server plus expected bytes.
func hlsTestCdn(t *testing.T, segmentDelay time.Duration) (*httptest.Server, []byte) {
	t.Helper()
	const total = 6
	segments := make(map[int][]byte)
	for i := 0; i < total; i++ {
		segments[i] = []byte(fmt.Sprintf("ENGINE-SEGMENT-%02d", i))
	}
	var want []byte
	for i := 0; i < total; i++ {
		want = append(want, segments[i]...)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/master.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n" +
			"#EXT-X-STREAM-INF:BANDWIDTH=500000\nlow/index.m3u8\n" +
			"#EXT-X-STREAM-INF:BANDWIDTH=2000000\nhigh/index.m3u8\n"))
	})
	mux.HandleFunc("/low/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXTINF:4,\nseg-0.ts\n#EXT-X-ENDLIST\n"))
	})
	mux.HandleFunc("/high/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("#EXTM3U\n")
		for i := 0; i < total; i++ {
			b.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
		}
		b.WriteString("#EXT-X-ENDLIST\n")
		w.Write([]byte(b.String()))
	})
	for i := 0; i < total; i++ {
		idx := i
		mux.HandleFunc(fmt.Sprintf("/high/seg-%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Referer") == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if segmentDelay > 0 {
				time.Sleep(segmentDelay)
			}
			w.Write(segments[idx])
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, want
}

func hlsResolveReferer() *base.Request {
	return &base.Request{
		URL:   "",
		Extra: map[string]any{"header": map[string]string{"Referer": "https://player.example.com/page"}},
	}
}

func TestDownloader_HlsProtocolEndToEnd(t *testing.T) {
	server, want := hlsTestCdn(t, 0)
	downloader := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloadDir := t.TempDir()
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DownloadDir = downloadDir
	if err := downloader.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}

	req := hlsResolveReferer()
	req.URL = server.URL + "/master.m3u8"
	rr, err := downloader.Resolve(req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := rr.Res.Files[0].Name; got != "index.ts" {
		t.Errorf("resolved name = %s, want index.ts", got)
	}
	taskID, err := downloader.Create(rr.ID)
	if err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, taskID, base.DownloadStatusDone, 10*time.Second)

	task := downloader.GetTask(taskID)
	if task.Protocol != "hls" {
		t.Errorf("task protocol = %s, want hls", task.Protocol)
	}
	data, err := os.ReadFile(filepath.Join(downloadDir, "index.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(want) {
		t.Errorf("merged file mismatch: %q", data)
	}
}

func TestDownloader_HlsProtocolPauseContinue(t *testing.T) {
	server, want := hlsTestCdn(t, 120*time.Millisecond)
	downloader := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()
	downloadDir := t.TempDir()
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DownloadDir = downloadDir
	// Throttle the protocol so the pause lands mid-download.
	cfg.ProtocolConfig["hls"] = map[string]any{
		"segmentConnections":    2,
		"maxRetries":            3,
		"timeoutSeconds":        5,
		"prefetchContentLength": false,
	}
	if err := downloader.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}

	req := hlsResolveReferer()
	req.URL = server.URL + "/master.m3u8"
	req.Extra = map[string]any{"header": map[string]string{"Referer": "https://player.example.com/page"}}
	// CreateDirect: engine resolves lazily on start, like a direct task.
	taskID, err := downloader.CreateDirect(req, &base.Options{Name: "movie.ts"})
	if err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, taskID, base.DownloadStatusRunning, 10*time.Second)
	if err := downloader.Pause(&TaskFilter{IDs: []string{taskID}}); err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, taskID, base.DownloadStatusPause, 5*time.Second)

	if err := downloader.Continue(&TaskFilter{IDs: []string{taskID}}); err != nil {
		t.Fatal(err)
	}
	waitForTaskStatus(t, downloader, taskID, base.DownloadStatusDone, 30*time.Second)

	task := downloader.GetTask(taskID)
	if task.Meta.Res.Size != int64(len(want)) {
		t.Errorf("final size = %d, want %d", task.Meta.Res.Size, len(want))
	}
	data, err := os.ReadFile(filepath.Join(downloadDir, "movie.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(want) {
		t.Errorf("merged file mismatch after pause/continue: %q", data)
	}
}
