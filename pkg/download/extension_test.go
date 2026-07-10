package download

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/logger"
	"github.com/GopeedLab/gopeed/pkg/base"
	gojaerror "github.com/GopeedLab/gopeed/pkg/download/engine/inject/error"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"github.com/dop251/goja"
)

func TestDownloader_InstallExtensionByFolder(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/basic", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByFolderDevMode(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/basic", true); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_Extension_BlobBlob(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://example.com/blob",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if rr.ID != "" {
			t.Fatalf("expected empty resolve id for extension resource, got %q", rr.ID)
		}

		doneCh := make(chan error, 1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyDone || event.Key == EventKeyError {
				doneCh <- event.Err
			}
		})

		dir := t.TempDir()
		if _, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
		}); err != nil {
			t.Fatal(err)
		}

		select {
		case err := <-doneCh:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for blob blob download")
		}

		data, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "hello world" {
			t.Fatalf("unexpected blob download content: %q", string(data))
		}

		if _, err := os.Stat(filepath.Join(downloader.cfg.StorageDir, "blob")); !os.IsNotExist(err) {
			t.Fatalf("expected blob spool dir not to be created, got err=%v", err)
		}
	})
}

func TestDownloader_Extension_BlobFunctionOpener(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://example.com/opener",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if rr.ID != "" {
			t.Fatalf("expected empty resolve id for extension resource, got %q", rr.ID)
		}

		dir := t.TempDir()
		id, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
		})
		if err != nil {
			t.Fatal(err)
		}
		waitForTaskTerminal(t, downloader, id, 5*time.Second)

		data, err := os.ReadFile(filepath.Join(dir, "opener.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "line 1\nline 2\n" {
			t.Fatalf("unexpected stream download content: %q", string(data))
		}
	})
}

func TestDownloader_Extension_BlobFunctionOpenerUnknownSize(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://example.com/opener-unknown",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if rr.ID != "" {
			t.Fatalf("expected empty resolve id for extension resource, got %q", rr.ID)
		}
		if got := rr.Res.Files[0].Size; got != 0 {
			t.Fatalf("expected unknown size in resolve result, got %d", got)
		}

		dir := t.TempDir()
		id, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
		})
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(60 * time.Millisecond)
		task := downloader.GetTask(id)
		if task == nil {
			t.Fatal("task not found")
		}
		if task.Status == base.DownloadStatusDone {
			t.Fatal("task finished before writer.close()")
		}
		waitForTaskTerminal(t, downloader, id, 5*time.Second)

		data, err := os.ReadFile(filepath.Join(dir, "opener-unknown.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "line 1\nline 2\n" {
			t.Fatalf("unexpected unknown-size stream content: %q", string(data))
		}

		task = downloader.GetTask(id)
		if task == nil {
			t.Fatal("task not found after completion")
		}
		if got := task.Meta.Res.Size; got != int64(len(data)) {
			t.Fatalf("unexpected final task size: got %d want %d", got, len(data))
		}
	})
}

func TestDownloader_Extension_BlobSourceSizePropagatesToCreatedTask(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://example.com/opener",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) != 1 {
			t.Fatalf("unexpected resolved file count: %d", len(rr.Res.Files))
		}

		file := rr.Res.Files[0]

		dir := t.TempDir()
		id, err := downloader.CreateDirect(file.Req, &base.Options{
			Path: dir,
			Name: file.Name,
		})
		if err != nil {
			t.Fatal(err)
		}

		task := downloader.GetTask(id)
		if task == nil {
			t.Fatal("task not found")
		}
		waitForTaskTerminal(t, downloader, id, 5*time.Second)

		task = downloader.GetTask(id)
		if task == nil {
			t.Fatal("task not found after completion")
		}
		if task.Meta == nil || task.Meta.Res == nil {
			t.Fatal("task meta resource not seeded")
		}
		if got := task.Meta.Res.Size; got != file.Size {
			t.Fatalf("unexpected task size: got %d want %d", got, file.Size)
		}
		if len(task.Meta.Res.Files) != 1 || task.Meta.Res.Files[0].Size != file.Size {
			t.Fatalf("unexpected task file metadata: %#v", task.Meta.Res.Files)
		}
	})
}

func TestDownloader_Extension_BlobFetchDrainedStreamReportsEarlyProgress(t *testing.T) {
	chunkDelay := 350 * time.Millisecond
	chunks := []string{
		strings.Repeat("a", 4096),
		strings.Repeat("b", 4096),
		strings.Repeat("c", 4096),
		strings.Repeat("d", 4096),
		strings.Repeat("e", 4096),
	}
	payload := strings.Join(chunks, "")
	upstreamDuration := time.Duration(len(chunks)) * chunkDelay
	firstProgressTimeout := upstreamDuration * 3 / 5
	completionTimeout := upstreamDuration + 4*time.Second
	requireEarlyProgressWindow(t, firstProgressTimeout, upstreamDuration)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/single-fetch-stream") || r.URL.Query().Get("source") != "1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			if _, err := w.Write([]byte(chunk)); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(chunkDelay)
		}
	}))
	defer server.Close()

	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob_stream_bridge", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: fmt.Sprintf("%s/single-fetch-stream?size=%d", server.URL, len(payload)),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) != 1 {
			t.Fatalf("unexpected resolve files: %#v", rr.Res.Files)
		}

		dir := t.TempDir()
		id, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
		})
		if err != nil {
			t.Fatal(err)
		}

		filePath := filepath.Join(dir, "single.bin")
		waitForFileSizeAtLeast(t, filePath, 1, firstProgressTimeout)
		waitForTaskTerminal(t, downloader, id, completionTimeout)

		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != payload {
			t.Fatalf("unexpected single stream content length=%d want=%d", len(data), len(payload))
		}
	})
}

func TestDownloader_Extension_BlobRangeFetchDrainedStreamReportsEarlyProgress(t *testing.T) {
	chunkDelay := 350 * time.Millisecond
	chunkSize := 4096
	chunks := []string{
		strings.Repeat("a", chunkSize),
		strings.Repeat("b", chunkSize),
		strings.Repeat("c", chunkSize),
		strings.Repeat("d", chunkSize),
		strings.Repeat("e", chunkSize),
		strings.Repeat("f", chunkSize),
	}
	payload := strings.Join(chunks, "")
	upstreamDuration := time.Duration(len(chunks)) * chunkDelay
	firstProgressTimeout := upstreamDuration * 3 / 5
	completionTimeout := upstreamDuration + 4*time.Second
	requireEarlyProgressWindow(t, firstProgressTimeout, upstreamDuration)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/range-fetch-stream") || r.URL.Query().Get("source") != "1" {
			http.NotFound(w, r)
			return
		}

		start := 0
		end := len(payload) - 1
		status := http.StatusOK
		if rangeHeader := r.Header.Get("Range"); strings.HasPrefix(rangeHeader, "bytes=") {
			parts := strings.SplitN(strings.TrimPrefix(rangeHeader, "bytes="), "-", 2)
			if len(parts) != 2 || parts[0] == "" {
				http.Error(w, "bad range", http.StatusBadRequest)
				return
			}
			parsedStart, err := strconv.Atoi(parts[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			start = parsedStart
			if parts[1] != "" {
				parsedEnd, err := strconv.Atoi(parts[1])
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				end = parsedEnd
			}
			if start >= len(payload) || end < start {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if end >= len(payload) {
				end = len(payload) - 1
			}
			status = http.StatusPartialContent
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		}

		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		w.WriteHeader(status)

		flusher, _ := w.(http.Flusher)
		for offset := start; offset <= end; offset += chunkSize {
			next := offset + chunkSize
			if next > end+1 {
				next = end + 1
			}
			if _, err := w.Write([]byte(payload[offset:next])); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(chunkDelay)
		}
	}))
	defer server.Close()

	setupDownloader(func(downloader *Downloader) {
		downloader.cfg.RefreshInterval = 50
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob_stream_bridge", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: fmt.Sprintf("%s/range-fetch-stream?size=%d", server.URL, len(payload)),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) != 1 || !rr.Res.Range {
			t.Fatalf("unexpected resolve result: %#v", rr.Res)
		}

		dir := t.TempDir()
		id, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
			Extra: map[string]any{
				"connections": 1,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		waitForTaskDownloadedBetween(t, downloader, id, 1, int64(len(payload)-1), firstProgressTimeout)
		waitForTaskTerminal(t, downloader, id, completionTimeout)

		data, err := os.ReadFile(filepath.Join(dir, "range.bin"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != payload {
			t.Fatalf("unexpected range stream content length=%d want=%d", len(data), len(payload))
		}
	})
}

func TestDownloader_Extension_BlobMultiplexFetchStreamsReportEarlyProgress(t *testing.T) {
	chunkDelay := 300 * time.Millisecond
	videoChunks := []string{"video-0", "video-1", "video-2", "video-3"}
	audioChunks := []string{"audio-0", "audio-1", "audio-2", "audio-3"}
	videoPayload := strings.Join(videoChunks, "")
	audioPayload := strings.Join(audioChunks, "")
	upstreamDuration := time.Duration(len(videoChunks)+len(audioChunks)) * chunkDelay
	firstProgressTimeout := upstreamDuration / 2
	completionTimeout := upstreamDuration + 4*time.Second
	requireEarlyProgressWindow(t, firstProgressTimeout, upstreamDuration)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/multiplex-fetch-stream") || r.URL.Query().Get("source") != "1" {
			http.NotFound(w, r)
			return
		}
		flusher, _ := w.(http.Flusher)
		for i := range videoChunks {
			if _, err := fmt.Fprintf(w, "v:%s\n", videoChunks[i]); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(chunkDelay)
			if _, err := fmt.Fprintf(w, "a:%s\n", audioChunks[i]); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(chunkDelay)
		}
	}))
	defer server.Close()

	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob_stream_bridge", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: server.URL + "/multiplex-fetch-stream",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) != 2 {
			t.Fatalf("unexpected resolve files: %#v", rr.Res.Files)
		}

		dir := t.TempDir()
		videoID, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
		})
		if err != nil {
			t.Fatal(err)
		}
		audioID, err := downloader.CreateDirect(rr.Res.Files[1].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[1].Name,
		})
		if err != nil {
			t.Fatal(err)
		}

		videoPath := filepath.Join(dir, "video.bin")
		audioPath := filepath.Join(dir, "audio.bin")
		waitForFileSizeAtLeast(t, videoPath, 1, firstProgressTimeout)
		waitForFileSizeAtLeast(t, audioPath, 1, firstProgressTimeout)
		waitForTaskTerminal(t, downloader, videoID, completionTimeout)
		waitForTaskTerminal(t, downloader, audioID, completionTimeout)

		videoData, err := os.ReadFile(videoPath)
		if err != nil {
			t.Fatal(err)
		}
		audioData, err := os.ReadFile(audioPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(videoData) != videoPayload {
			t.Fatalf("unexpected video content %q want %q", string(videoData), videoPayload)
		}
		if string(audioData) != audioPayload {
			t.Fatalf("unexpected audio content %q want %q", string(audioData), audioPayload)
		}
	})
}

func TestDownloader_Extension_BlobFunctionOpenerRangeResume(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://example.com/opener-range",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) != 1 {
			t.Fatalf("unexpected resolved file count: %d", len(rr.Res.Files))
		}
		if !rr.Res.Range {
			t.Fatal("expected resumable blob resource")
		}

		file := rr.Res.Files[0]

		dir := t.TempDir()
		id, err := downloader.CreateDirect(file.Req, &base.Options{
			Path: dir,
			Name: file.Name,
		})
		if err != nil {
			t.Fatal(err)
		}

		waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 5*time.Second)

		filePath := filepath.Join(dir, file.Name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "line 1\nline 2\nline 3\n" {
			t.Fatalf("unexpected resumed content: %q", string(data))
		}
	})
}

func TestDownloader_Extension_BlobResumeAfterRestartViaOnError(t *testing.T) {
	storageDir := t.TempDir()
	downloadDir := t.TempDir()

	payload := strings.Repeat("x", 262144)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/restart-range") {
			http.NotFound(w, r)
			return
		}

		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			return
		}

		data := []byte(payload)
		start := 0
		rangeEnd := len(data) - 1
		status := http.StatusOK
		rangeHeader := r.Header.Get("Range")
		if strings.HasPrefix(rangeHeader, "bytes=") {
			parts := strings.SplitN(strings.TrimPrefix(rangeHeader, "bytes="), "-", 2)
			if len(parts) != 2 || parts[0] == "" {
				http.Error(w, "invalid range", http.StatusBadRequest)
				return
			}
			parsed, err := strconv.Atoi(parts[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			start = parsed
			if parts[1] != "" {
				parsedEnd, err := strconv.Atoi(parts[1])
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				rangeEnd = parsedEnd
			}
			if start > len(data) || rangeEnd < start {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if rangeEnd >= len(data) {
				rangeEnd = len(data) - 1
			}
			status = http.StatusPartialContent
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, rangeEnd, len(data)))
		}

		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(rangeEnd-start+1))
		w.WriteHeader(status)

		flusher, _ := w.(http.Flusher)
		chunkSize := 4096
		for offset := start; offset <= rangeEnd; offset += chunkSize {
			end := offset + chunkSize
			if end > rangeEnd+1 {
				end = rangeEnd + 1
			}
			if _, err := w.Write(data[offset:end]); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(20 * time.Millisecond)
		}
	}))
	defer server.Close()

	newDownloader := func() *Downloader {
		downloader := NewDownloader(&DownloaderConfig{
			Storage:    NewBoltStorage(storageDir),
			StorageDir: storageDir,
			DownloaderStoreConfig: &base.DownloaderStoreConfig{
				DownloadDir: downloadDir,
			},
		})
		downloader.cfg.RefreshInterval = 50
		if err := downloader.Setup(); err != nil {
			t.Fatal(err)
		}
		return downloader
	}

	downloader := newDownloader()
	if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob_restart", false); err != nil {
		t.Fatal(err)
	}

	rr, err := downloader.Resolve(&base.Request{
		URL: server.URL + "/restart-range",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rr.Res.Files) != 1 || !rr.Res.Range {
		t.Fatalf("unexpected resolve result: %#v", rr.Res)
	}

	file := rr.Res.Files[0]
	id, err := downloader.CreateDirect(file.Req, &base.Options{
		Path: downloadDir,
		Name: file.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	waitForTaskStatus(t, downloader, id, base.DownloadStatusRunning, 2*time.Second)

	deadline := time.Now().Add(2 * time.Second)
	var partialSize int64
	for time.Now().Before(deadline) {
		task := downloader.GetTask(id)
		if task == nil {
			t.Fatal("task not found")
		}
		partialSize = task.Progress.Downloaded
		if partialSize > 0 && partialSize < int64(len(payload)) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if partialSize <= 0 || partialSize >= int64(len(payload)) {
		t.Fatalf("expected partial downloaded bytes before restart, got %d", partialSize)
	}
	oldURL := file.Req.URL

	if err := downloader.Close(); err != nil {
		t.Fatal(err)
	}

	// Keep the old loopback address reachable after the Registry is gone. This
	// makes the stale capability URL exercise the ordinary HTTP failure path
	// deterministically (404), rather than the HTTP fetcher's retry-forever
	// policy for transient dial errors.
	parsedOldURL, err := neturl.Parse(oldURL)
	if err != nil {
		t.Fatal(err)
	}
	staleListener, err := net.Listen("tcp", parsedOldURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	staleServer := &http.Server{Handler: http.NotFoundHandler()}
	go func() {
		_ = staleServer.Serve(staleListener)
	}()
	defer staleServer.Close()

	downloader = newDownloader()
	defer func() {
		_ = downloader.Clear()
		os.RemoveAll(storageDir)
		os.RemoveAll(downloadDir)
	}()
	if len(downloader.GetExtensions()) == 0 {
		t.Fatal("expected restored downloader to load extensions")
	}

	task := downloader.GetTask(id)
	if task == nil {
		t.Fatal("restored task not found")
	}
	if task.Status != base.DownloadStatusPause {
		t.Fatalf("expected restored blob task to use normal paused HTTP state, got %s", task.Status)
	}
	if task.Protocol != "http" {
		t.Fatalf("expected restored task protocol http, got %s", task.Protocol)
	}
	if task.Meta == nil || task.Meta.Req == nil {
		t.Fatal("restored task request missing")
	}
	if task.Meta.Req.RawURL == "" {
		t.Fatal("restored task raw url missing")
	}
	if task.Meta.Req.Labels == nil || task.Meta.Req.Labels["mode"] != "restart" {
		t.Fatalf("unexpected restored task labels: %#v", task.Meta.Req.Labels)
	}
	if task.Meta.Req.URL != oldURL {
		t.Fatalf("restored URL changed before ordinary HTTP handling: old=%q current=%q", oldURL, task.Meta.Req.URL)
	}

	if err := downloader.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatal(err)
	}

	// A missing Registry entry is deliberately handled as an ordinary HTTP URL.
	// Wait for the HTTP fetcher's normal retry policy to exhaust before onError
	// rebuilds the capability URL.
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		task = downloader.GetTask(id)
		if task == nil {
			t.Fatal("restored task not found after continue")
		}
		if task.Meta != nil && task.Meta.Req != nil && task.Meta.Req.Labels["started"] != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if task == nil || task.Meta == nil || task.Meta.Req == nil || task.Meta.Req.Labels["started"] == "" {
		t.Fatalf("expected onError recovery to run, labels=%#v old=%q current=%q", task.Meta.Req.Labels, oldURL, task.Meta.Req.URL)
	}
	if task.Meta.Req.Labels["rebuilt"] != "true" {
		t.Fatalf("expected onError recovery to rebuild blob URL, labels=%#v old=%q current=%q rebuildError=%q", task.Meta.Req.Labels, oldURL, task.Meta.Req.URL, task.Meta.Req.Labels["rebuildError"])
	}
	if task.Meta.Req.URL == oldURL {
		t.Fatalf("expected onError recovery to rebuild blob URL, old=%q current=%q labels=%#v", oldURL, task.Meta.Req.URL, task.Meta.Req.Labels)
	}

	waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 15*time.Second)
	data, err := os.ReadFile(filepath.Join(downloadDir, file.Name))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != payload {
		t.Fatalf("unexpected restarted blob content length=%d want=%d", len(data), len(payload))
	}
}

func TestDownloader_Extension_BlobPauseContinueKeepsSource(t *testing.T) {
	payload := strings.Repeat("x", 262144)
	cases := []struct {
		name      string
		url       string
		wantRange bool
	}{
		{
			name:      "blob",
			url:       "https://example.com/pause-rebuild-blob",
			wantRange: true,
		},
		{
			name:      "opener",
			url:       "https://example.com/pause-rebuild-opener",
			wantRange: false,
		},
		{
			name:      "range opener",
			url:       "https://example.com/pause-rebuild-range",
			wantRange: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			downloader := NewDownloader(nil)
			downloader.cfg.RefreshInterval = 50
			if err := downloader.Setup(); err != nil {
				t.Fatal(err)
			}
			defer downloader.Clear()

			if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob_pause_rebuild", false); err != nil {
				t.Fatal(err)
			}

			rr, err := downloader.Resolve(&base.Request{
				URL: tc.url,
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(rr.Res.Files) != 1 || rr.Res.Range != tc.wantRange {
				t.Fatalf("unexpected resolve result: %#v", rr.Res)
			}

			file := rr.Res.Files[0]
			downloadDir := t.TempDir()
			id, err := downloader.CreateDirect(file.Req, &base.Options{
				Path: downloadDir,
				Name: file.Name,
				Extra: map[string]any{
					"connections": 1,
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			filePath := filepath.Join(downloadDir, file.Name)
			waitForFileSizeAtLeast(t, filePath, 4096, 2*time.Second)

			task := downloader.GetTask(id)
			if task == nil || task.Meta == nil || task.Meta.Req == nil {
				t.Fatal("task request missing before pause")
			}
			oldURL := task.Meta.Req.URL

			if err := downloader.Pause(&TaskFilter{IDs: []string{id}}); err != nil {
				t.Fatal(err)
			}
			task = downloader.GetTask(id)
			if task == nil || task.Status != base.DownloadStatusPause {
				t.Fatalf("expected paused task, got %#v", task)
			}
			if !downloader.blob.IsURL(oldURL) {
				t.Fatal("paused blob source should remain registered")
			}

			// Continue immediately after Pause returns. Pause is synchronous, so the
			// previous run cannot race with this new Start.
			if err := downloader.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
				t.Fatal(err)
			}
			waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 10*time.Second)
			task = downloader.GetTask(id)
			if task == nil || task.Meta == nil || task.Meta.Req == nil {
				t.Fatal("task request missing after continue")
			}
			if task.Meta.Req.URL != oldURL {
				t.Fatalf("pause/continue unexpectedly rebuilt URL: old=%q new=%q", oldURL, task.Meta.Req.URL)
			}
			if task.Meta.Req.Labels["rebuilt"] != "" {
				t.Fatalf("pause/continue should not invoke recovery: %#v", task.Meta.Req.Labels)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != payload {
				t.Fatalf("unexpected resumed content length=%d want=%d", len(data), len(payload))
			}
		})
	}
}

func TestDownloader_Extension_BlobUnknownSizePauseContinueKeepsSource(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob", false); err != nil {
			t.Fatal(err)
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://example.com/opener-unknown",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}

		dir := t.TempDir()
		id, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
			Path: dir,
			Name: rr.Res.Files[0].Name,
		})
		if err != nil {
			t.Fatal(err)
		}

		filePath := filepath.Join(dir, "opener-unknown.txt")
		waitForFileSizeAtLeast(t, filePath, int64(len("line 1\n")), 2*time.Second)

		if err := downloader.Pause(&TaskFilter{IDs: []string{id}}); err != nil {
			t.Fatal(err)
		}

		task := downloader.GetTask(id)
		if task == nil {
			t.Fatal("task not found after pause")
		}
		if task.Status != base.DownloadStatusPause {
			t.Fatalf("expected paused task, got %s", task.Status)
		}

		stat, err := os.Stat(filePath)
		if err != nil {
			t.Fatal(err)
		}
		pausedSize := stat.Size()

		time.Sleep(250 * time.Millisecond)

		stat, err = os.Stat(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if stat.Size() != pausedSize {
			t.Fatalf("expected paused file size to remain %d, got %d", pausedSize, stat.Size())
		}

		oldURL := task.Meta.Req.URL
		if !downloader.blob.IsURL(oldURL) {
			t.Fatal("paused unknown-size source should remain registered")
		}
		if err := downloader.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
			t.Fatal(err)
		}
		waitForTaskStatus(t, downloader, id, base.DownloadStatusDone, 5*time.Second)
		task = downloader.GetTask(id)
		if task.Meta.Req.URL != oldURL {
			t.Fatalf("pause/continue unexpectedly replaced URL: old=%q new=%q", oldURL, task.Meta.Req.URL)
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "line 1\nline 2\n" {
			t.Fatalf("unexpected resumed unknown-size content %q", string(data))
		}
	})
}

func TestDownloader_Extension_BlobRecoverOnError(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/blob_recover", false); err != nil {
			t.Fatal(err)
		}

		for _, path := range []string{"recover", "recover-range"} {
			path := path
			t.Run(path, func(t *testing.T) {
				rawURL := "https://example.com/" + path
				rr, err := downloader.Resolve(&base.Request{URL: rawURL}, nil)
				if err != nil {
					t.Fatal(err)
				}

				dir := t.TempDir()
				id, err := downloader.CreateDirect(rr.Res.Files[0].Req, &base.Options{
					Path: dir,
					Name: rr.Res.Files[0].Name,
				})
				if err != nil {
					t.Fatal(err)
				}

				filePath := filepath.Join(dir, "recover.txt")
				deadline := time.Now().Add(15 * time.Second)
				for time.Now().Before(deadline) {
					task := downloader.GetTask(id)
					if task != nil && downloader.taskStatus(task) == base.DownloadStatusDone {
						break
					}
					time.Sleep(20 * time.Millisecond)
				}

				task := downloader.GetTask(id)
				if task == nil {
					t.Fatal("task not found after recovery")
				}
				if downloader.taskStatus(task) != base.DownloadStatusDone {
					var fileSize int64 = -1
					if info, statErr := os.Stat(filePath); statErr == nil {
						fileSize = info.Size()
					}
					t.Fatalf(
						"timeout waiting for recovered blob download: status=%s downloaded=%d url=%q rawUrl=%q labels=%#v fileSize=%d",
						task.Status,
						task.Progress.Downloaded,
						task.Meta.Req.URL,
						task.Meta.Req.RawURL,
						task.Meta.Req.Labels,
						fileSize,
					)
				}

				data, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "ok\n" {
					t.Fatalf("unexpected recovered blob file content: %q", string(data))
				}

				if task.Meta.Req.RawURL != rawURL {
					t.Fatalf("unexpected raw url: %q", task.Meta.Req.RawURL)
				}
			})
		}
	})
}

func TestDownloader_InstallExtensionByGit(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByGit("https://github.com/GopeedLab/gopeed-extension-samples#github-release-sample"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByGitSimple(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByGit("github.com/GopeedLab/gopeed-extension-samples#github-release-sample"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByGitFull(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByGit("https://github.com/GopeedLab/gopeed-extension-samples.git#github-release-sample"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_UpgradeExtension(t *testing.T) {
	getSetting := func(settings []*Setting, name string) *Setting {
		for _, setting := range settings {
			if setting.Name == name {
				return setting
			}
		}
		return nil
	}

	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/update", false)
		if err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if len(extensions) == 0 {
			t.Fatal("extension not installed")
		}
		oldVersion := installedExt.Version
		// fetch new version from git
		newVersion, err := downloader.UpgradeCheckExtension(installedExt.Identity)
		if err != nil {
			t.Fatal(err)
		}
		if newVersion == "" {
			t.Fatal("new version not found")
		}
		// update extension
		if err = downloader.UpgradeExtension(installedExt.Identity); err != nil {
			t.Fatal(err)
		}
		upgradeExt := downloader.getExtension(installedExt.Identity)
		if upgradeExt.Version == oldVersion {
			t.Fatal("extension update fail")
		}

		// check setting update
		s1 := getSetting(upgradeExt.Settings, "s1")
		if s1.Title == "S1 old" {
			t.Fatal("setting update fail")
		}
		// check setting type update
		s2 := getSetting(upgradeExt.Settings, "s2")
		if s2.Type == "number" {
			t.Fatal("setting type update fail")
		}
		// check setting remove
		d1 := getSetting(upgradeExt.Settings, "d1")
		if d1 != nil {
			t.Fatal("setting remove fail")
		}
		// check setting add
		s3 := getSetting(upgradeExt.Settings, "s3")
		if s3 == nil {
			t.Fatal("setting add fail")
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://test.com",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if rr.Res.Name != "test" {
			t.Fatal("script update fail")
		}
	})
}

func TestDownloader_Extension_OnStart(t *testing.T) {
	downloadAndCheck := func(req *base.Request) {
		setupDownloader(func(downloader *Downloader) {
			if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/on_start", false); err != nil {
				t.Fatal(err)
			}
			errCh := make(chan error, 1)
			downloader.Listener(func(event *Event) {
				if event.Key == EventKeyFinally {
					errCh <- event.Err
				}
			})
			id, err := downloader.CreateDirect(req, nil)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case err = <-errCh:
				break
			case <-time.After(time.Second * 30): // Increased timeout for real network requests
				err = errors.New("timeout")
			}
			if err != nil {
				panic("extension on start download error: " + err.Error())
			}
			task := downloader.GetTask(id)
			if task.Meta.Req.URL != "https://github.com" {
				t.Fatalf("except url: https://github.com, actual: %s", task.Meta.Req.URL)
			}
			if task.Meta.Req.Labels["modified"] != "true" {
				t.Fatalf("except label: modified=true, actual: %s", task.Meta.Req.Labels["modified"])
			}
		})
	}

	// url match
	downloadAndCheck(&base.Request{
		URL: "https://github.com/gopeed/test/404",
	})

	// label match
	downloadAndCheck(&base.Request{
		URL: "https://test.com",
		Labels: map[string]string{
			"test": "true",
		},
	})
}

func TestDownloader_Extension_OnError(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/on_error", false); err != nil {
			t.Fatal(err)
		}
		errCh := make(chan error, 1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyFinally {
				errCh <- event.Err
			}
		})
		id, err := downloader.CreateDirect(&base.Request{
			URL: "https://github.com/gopeed/test/404",
			Labels: map[string]string{
				"test": "true",
			},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err = <-errCh:
			break
		case <-time.After(time.Second * 30): // Increased timeout for real network requests
			err = errors.New("timeout")
		}

		if err != nil {
			panic("extension on error download error: " + err.Error())
		}
		// extension on error modify url and continue download
		task := downloader.GetTask(id)
		if task.Status != base.DownloadStatusDone {
			t.Fatalf("except status is done, actual: %s", task.Status)
		}
	})
}

func TestDownloader_Extension_OnDone(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/on_done", false); err != nil {
			t.Fatal(err)
		}
		errCh := make(chan error, 1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyFinally {
				errCh <- event.Err
			}
		})
		id, err := downloader.CreateDirect(&base.Request{
			URL: "https://github.com",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err = <-errCh:
			break
		case <-time.After(time.Second * 30): // Increased timeout for real network requests
			err = errors.New("timeout")
		}
		// wait for script execution
		time.Sleep(time.Millisecond * 3000)

		if err != nil {
			panic("extension on done download error: " + err.Error())
		}
		// extension on error modify url and continue download
		task := downloader.GetTask(id)
		if task.Meta.Req.Labels["modified"] != "true" {
			t.Fatalf("except label: modified=true, actual: %s", task.Meta.Req.Labels["modified"])
		}
		if task.Status != base.DownloadStatusDone {
			t.Fatalf("except status is done, actual: %s", task.Status)
		}
	})
}

func TestDownloader_Extension_Errors(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/script_error", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 2 {
			t.Fatal("script error catch failed")
		}
	})

	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/function_error", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 2 {
			t.Fatal("function error catch failed")
		}
	})

	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/message_error", false); err != nil {
			t.Fatal(err)
		}
		_, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err == nil {
			t.Fatalf("except error, but got nil")
		}
		me, ok := err.(*gojaerror.MessageError)
		if !ok {
			t.Fatalf("except MessageError type, but got %s", err)
		}
		want := "test"
		if me.Error() != want {
			t.Fatalf("except MessageError message %s, but got %s", want, me.Message)
		}
	})
}

func TestDownloader_Extension_Settings(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_empty", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("settings parse error")
		}
	})

	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_all", false)
		if err != nil {
			t.Fatal(err)
		}
		downloader.UpdateExtensionSettings(installedExt.Identity, map[string]any{
			"stringValued":  "valued",
			"numberValued":  1.1,
			"booleanValued": true,
		})
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("settings parse error")
		}
	})
}

func TestDownloader_ExtensionStorage(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/storage", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_SwitchExtension(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/basic", false)
		if err != nil {
			t.Fatal(err)
		}
		if installedExt.Disabled == true {
			t.Fatal("extension disabled")
		}
		if err = downloader.SwitchExtension(installedExt.Identity, false); err != nil {
			t.Fatal(err)
		}
		if installedExt.Disabled == false {
			t.Fatal("extension enabled")
		}
	})
}

func TestDownloader_DeleteExtension(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_all", false)
		if err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if err := downloader.DeleteExtension(installedExt.Identity); err != nil {
			t.Fatal(err)
		}
		extensions = downloader.GetExtensions()
		if len(extensions) != 0 {
			t.Fatal("extension delete fail")
		}
	})
}

func TestDownloader_Extension_Logger(t *testing.T) {
	logger := logger.NewLogger(false, "")
	il := newInstanceLogger(&Extension{
		Name: "test",
	}, logger)
	il.Debug(goja.NaN(), goja.Undefined())
	il.Info(goja.NaN(), goja.Undefined())
	il.Warn(goja.NaN(), goja.Undefined())
	il.Error(goja.NaN(), goja.Undefined())
}

func TestDownloader_ExtensionRuntimeWebViewInjected(t *testing.T) {
	downloader, cleanup, err := newTestExtensionEngineDownloader()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	runtime, err := newTestExtensionEngine(t, downloader)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	value, err := runtime.Eval(`({
			hasRuntime: !!gopeed.runtime,
			hasBlob: !!(gopeed.runtime && gopeed.runtime.blob),
			hasCreateObjectURL: typeof gopeed.runtime.blob.createObjectURL,
			hasRevokeObjectURL: typeof gopeed.runtime.blob.revokeObjectURL,
			hasWebView: !!(gopeed.runtime && gopeed.runtime.webview),
			hasOpen: typeof gopeed.runtime.webview.open,
			hasWebViewIsAvailable: typeof gopeed.runtime.webview.isAvailable,
		webViewAvailable: gopeed.runtime.webview.isAvailable()
	})`)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected eval result type: %T", value)
	}
	if result["hasRuntime"] != true {
		t.Fatalf("expected runtime injection, got %#v", result)
	}
	if result["hasBlob"] != true {
		t.Fatalf("expected blob injection, got %#v", result)
	}
	if result["hasCreateObjectURL"] != "function" || result["hasRevokeObjectURL"] != "function" {
		t.Fatalf("expected blob api functions, got %#v", result)
	}
	if result["hasWebView"] != true {
		t.Fatalf("expected webview injection, got %#v", result)
	}
	if result["hasOpen"] != "function" || result["hasWebViewIsAvailable"] != "function" {
		t.Fatalf("expected webview api functions, got %#v", result)
	}
	if result["webViewAvailable"] != false {
		t.Fatalf("expected unavailable webview runtime by default, got %#v", result["webViewAvailable"])
	}
}

func TestDownloader_ExtensionRuntimeBlobRejectsReadableStreamSource(t *testing.T) {
	downloader, cleanup, err := newTestExtensionEngineDownloader()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	runtime, err := newTestExtensionEngine(t, downloader)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	value, err := runtime.Eval(`(() => {
		try {
			gopeed.runtime.blob.createObjectURL(new ReadableStream({
				start(controller) {
					controller.close();
				},
			}));
			return "";
		} catch (error) {
			return String(error && error.message ? error.message : error);
		}
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	message, ok := value.(string)
	if !ok {
		t.Fatalf("unexpected eval result type: %T", value)
	}
	if !strings.Contains(message, "Expected Blob or opener function") {
		t.Fatalf("expected ReadableStream source rejection, got %q", message)
	}
}

func TestDownloader_ExtensionRuntimeWebViewAvailabilityFromProvider(t *testing.T) {
	downloader := NewDownloader(&DownloaderConfig{
		Storage: NewMemStorage(),
		WebViewProvider: fakeExtensionWebViewProvider{
			available: true,
			opener:    &fakeRuntimeWebViewOpener{},
		},
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	runtime, err := newTestExtensionEngine(t, downloader)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	value, err := runtime.Eval(`gopeed.runtime.webview.isAvailable()`)
	if err != nil {
		t.Fatal(err)
	}
	if value != true {
		t.Fatalf("expected available webview runtime, got %#v", value)
	}
}

func TestDownloader_ExtensionRuntimeWebViewPageMethodsInjected(t *testing.T) {
	downloader := NewDownloader(&DownloaderConfig{
		Storage: NewMemStorage(),
		WebViewProvider: fakeExtensionWebViewProvider{
			available: true,
			opener:    &fakeRuntimeWebViewOpener{},
		},
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	runtime, err := newTestExtensionEngine(t, downloader)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	value, err := runtime.Eval(`(async () => {
		const page = await gopeed.runtime.webview.open();
		return {
			hasFocus: typeof page.focus,
			hasClick: typeof page.click,
			hasType: typeof page.type,
			hasWaitForSelector: typeof page.waitForSelector,
			hasWaitForFunction: typeof page.waitForFunction,
			hasWaitForLoad: typeof page.waitForLoad,
		};
	})()`)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected page methods result type: %T", value)
	}
	for _, key := range []string{
		"hasFocus",
		"hasClick",
		"hasType",
		"hasWaitForSelector",
		"hasWaitForFunction",
	} {
		if result[key] != "function" {
			t.Fatalf("expected %s to be a function, got %#v", key, result[key])
		}
	}
	if result["hasWaitForLoad"] != "undefined" {
		t.Fatalf("expected hasWaitForLoad to be undefined, got %#v", result["hasWaitForLoad"])
	}
}

func TestDownloader_ExtensionRuntimeWebViewExecuteAnonymousFunction(t *testing.T) {
	opener := &capturingRuntimeWebViewOpener{
		page: &capturingRuntimeWebViewPage{
			executeValue: map[string]any{
				"title":      "Hello",
				"url":        "https://example.com",
				"userAgent":  "UA",
				"readyState": "complete",
			},
		},
	}
	downloader := NewDownloader(&DownloaderConfig{
		Storage: NewMemStorage(),
		WebViewProvider: fakeExtensionWebViewProvider{
			available: true,
			opener:    opener,
		},
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	runtime, err := newTestExtensionEngine(t, downloader)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	value, err := runtime.Eval(`(async () => {
		const page = await gopeed.runtime.webview.open();
		return await page.execute(() => ({
			title: document.title || "",
			url: String(location.href || ""),
			userAgent: navigator.userAgent,
			readyState: document.readyState,
		}));
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected execute result type: %T", value)
	}
	if result["title"] != "Hello" || result["readyState"] != "complete" {
		t.Fatalf("unexpected execute result: %#v", result)
	}
	expected := `(() => ({
			title: document.title || "",
			url: String(location.href || ""),
			userAgent: navigator.userAgent,
			readyState: document.readyState,
		}))`
	if opener.page.lastExecuteSource != expected {
		t.Fatalf("unexpected execute source: %q", opener.page.lastExecuteSource)
	}
}

func TestDownloader_TriggerOnResolve_DetachedAsyncWorkDoesNotBlock(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		extDir := t.TempDir()
		manifest := `{
  "name": "detached-async-resolve",
  "author": "gopeed",
  "title": "Detached Async Resolve",
  "version": "0.0.1",
  "scripts": [
    {
      "event": "onResolve",
      "match": {
        "urls": ["*://example.com/*"]
      },
      "entry": "index.js"
    }
  ]
}`
		script := `gopeed.events.onResolve(async (ctx) => {
  (async () => {
    await new Promise((resolve) => setTimeout(resolve, 500));
    globalThis.__detachedDone = true;
  })();
  ctx.res = {
    name: 'done',
    files: [
      {
        name: 'out.txt',
        req: {
          url: 'https://example.com/file.txt'
        }
      }
    ]
  };
});`
		if err := os.WriteFile(filepath.Join(extDir, "manifest.json"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(extDir, "index.js"), []byte(script), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := downloader.InstallExtensionByFolder(extDir, false); err != nil {
			t.Fatal(err)
		}

		type result struct {
			res *base.Resource
			err error
		}
		done := make(chan result, 1)
		startedAt := time.Now()
		go func() {
			res, err := downloader.triggerOnResolve(&base.Request{URL: "https://example.com/test"})
			done <- result{res: res, err: err}
		}()

		select {
		case out := <-done:
			if out.err != nil {
				t.Fatal(out.err)
			}
			if time.Since(startedAt) > 200*time.Millisecond {
				t.Fatalf("triggerOnResolve blocked for %s", time.Since(startedAt))
			}
			if out.res == nil || len(out.res.Files) != 1 {
				t.Fatalf("unexpected resource: %#v", out.res)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for triggerOnResolve")
		}
	})
}

type fakeExtensionWebViewProvider struct {
	available bool
	opener    enginewebview.Opener
}

func (p fakeExtensionWebViewProvider) IsAvailable() bool {
	return p.available
}

func (p fakeExtensionWebViewProvider) Open(opts enginewebview.OpenOptions) (enginewebview.Page, error) {
	if p.opener == nil {
		return nil, nil
	}
	return p.opener.Open(opts)
}

type fakeRuntimeWebViewOpener struct{}

func (fakeRuntimeWebViewOpener) Open(enginewebview.OpenOptions) (enginewebview.Page, error) {
	return fakeRuntimeWebViewPage{}, nil
}

type fakeRuntimeWebViewPage struct{}

func (fakeRuntimeWebViewPage) AddInitScript(string) error {
	return nil
}

func (fakeRuntimeWebViewPage) Goto(string, enginewebview.GotoOptions) error {
	return nil
}

func (fakeRuntimeWebViewPage) Execute(string, ...any) (any, error) {
	return nil, nil
}

func (fakeRuntimeWebViewPage) GetCookies() ([]enginewebview.Cookie, error) {
	return nil, nil
}

func (fakeRuntimeWebViewPage) SetCookie(enginewebview.Cookie) error {
	return nil
}

func (fakeRuntimeWebViewPage) DeleteCookie(enginewebview.Cookie) error {
	return nil
}

func (fakeRuntimeWebViewPage) ClearCookies() error {
	return nil
}

func (fakeRuntimeWebViewPage) Close() error {
	return nil
}

type capturingRuntimeWebViewOpener struct {
	page *capturingRuntimeWebViewPage
}

func (o *capturingRuntimeWebViewOpener) Open(enginewebview.OpenOptions) (enginewebview.Page, error) {
	return o.page, nil
}

type capturingRuntimeWebViewPage struct {
	lastExecuteSource string
	lastExecuteArgs   []any
	executeValue      any
}

func (p *capturingRuntimeWebViewPage) AddInitScript(string) error {
	return nil
}

func (p *capturingRuntimeWebViewPage) Goto(string, enginewebview.GotoOptions) error {
	return nil
}

func (p *capturingRuntimeWebViewPage) Execute(expression string, args ...any) (any, error) {
	p.lastExecuteSource = expression
	p.lastExecuteArgs = args
	return p.executeValue, nil
}

func (p *capturingRuntimeWebViewPage) GetCookies() ([]enginewebview.Cookie, error) {
	return nil, nil
}

func (p *capturingRuntimeWebViewPage) SetCookie(enginewebview.Cookie) error {
	return nil
}

func (p *capturingRuntimeWebViewPage) DeleteCookie(enginewebview.Cookie) error {
	return nil
}

func (p *capturingRuntimeWebViewPage) ClearCookies() error {
	return nil
}

func (p *capturingRuntimeWebViewPage) Close() error {
	return nil
}

func setupDownloader(fn func(downloader *Downloader)) {
	storageDir, err := os.MkdirTemp("", "gopeed-test-storage-")
	if err != nil {
		panic(err)
	}
	downloadDir, err := os.MkdirTemp("", "gopeed-test-download-")
	if err != nil {
		_ = os.RemoveAll(storageDir)
		panic(err)
	}
	downloader := NewDownloader(&DownloaderConfig{
		StorageDir: storageDir,
	})
	if err := downloader.Setup(); err != nil {
		_ = os.RemoveAll(storageDir)
		_ = os.RemoveAll(downloadDir)
		panic(err)
	}
	cfg, err := downloader.GetConfig()
	if err != nil {
		_ = os.RemoveAll(storageDir)
		_ = os.RemoveAll(downloadDir)
		panic(err)
	}
	cfg.DownloadDir = downloadDir
	if err := downloader.PutConfig(cfg); err != nil {
		_ = os.RemoveAll(storageDir)
		_ = os.RemoveAll(downloadDir)
		panic(err)
	}
	defer func() {
		downloader.Clear()
		os.RemoveAll(downloader.cfg.StorageDir)
		os.RemoveAll(downloadDir)
	}()
	fn(downloader)
}

func newTestExtensionEngineDownloader() (*Downloader, func(), error) {
	downloader := NewDownloader(&DownloaderConfig{
		Storage: NewMemStorage(),
	})
	if err := downloader.Setup(); err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = downloader.Clear()
	}
	return downloader, cleanup, nil
}

func newTestExtensionEngine(t *testing.T, downloader *Downloader) (*ExtensionEngine, error) {
	t.Helper()
	return downloader.NewExtensionEngine(&Extension{
		Name:    "test-runtime",
		Author:  "gopeed",
		Title:   "Gopeed Test Script Runtime",
		Version: "0.0.0",
		DevMode: true,
	}, map[string]any{})
}

func waitForFileSizeAtLeast(t *testing.T, path string, size int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil && info.Size() >= size {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for file %s size >= %d", path, size)
}

func requireEarlyProgressWindow(t *testing.T, earlyTimeout, upstreamDuration time.Duration) {
	t.Helper()
	if earlyTimeout <= 0 {
		t.Fatalf("early progress timeout must be positive, got %s", earlyTimeout)
	}
	if upstreamDuration <= 0 {
		t.Fatalf("upstream duration must be positive, got %s", upstreamDuration)
	}
	if earlyTimeout >= upstreamDuration {
		t.Fatalf("early progress timeout %s must be shorter than upstream duration %s", earlyTimeout, upstreamDuration)
	}
}

func waitForTaskDownloadedBetween(t *testing.T, downloader *Downloader, id string, min, max int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task := downloader.GetTask(id)
		if task != nil && task.Progress != nil && task.Progress.Downloaded >= min && task.Progress.Downloaded <= max {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	task := downloader.GetTask(id)
	if task == nil {
		t.Fatalf("timeout waiting for task %s downloaded between %d and %d: task not found", id, min, max)
	}
	downloaded := int64(0)
	if task.Progress != nil {
		downloaded = task.Progress.Downloaded
	}
	t.Fatalf("timeout waiting for task %s downloaded between %d and %d: got %d status=%s", id, min, max, downloaded, task.Status)
}

func waitForTaskTerminal(t *testing.T, downloader *Downloader, id string, timeout time.Duration) {
	t.Helper()
	doneCh := make(chan error, 1)
	downloader.Listener(func(event *Event) {
		if event.Task == nil || event.Task.ID != id {
			return
		}
		if event.Key == EventKeyDone || event.Key == EventKeyError {
			select {
			case doneCh <- event.Err:
			default:
			}
		}
	})

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task := downloader.GetTask(id)
		if task != nil {
			switch task.Status {
			case base.DownloadStatusDone:
				return
			case base.DownloadStatusError:
				select {
				case err := <-doneCh:
					if err == nil {
						t.Fatalf("task %s ended with error status", id)
					}
					t.Fatal(err)
				default:
					t.Fatalf("task %s ended with error status", id)
				}
			}
		}
		select {
		case err := <-doneCh:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-time.After(10 * time.Millisecond):
		}
	}

	select {
	case err := <-doneCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(timeout):
		task := downloader.GetTask(id)
		if task == nil {
			t.Fatalf("timeout waiting for task %s: task not found", id)
		}
		t.Fatalf("timeout waiting for task %s: status=%s downloaded=%d total=%d", id, task.Status, task.Progress.Downloaded, task.Meta.Res.Size)
	}
}

func waitForTaskStatus(t *testing.T, downloader *Downloader, id string, status base.Status, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task := downloader.GetTask(id)
		if task != nil && task.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	task := downloader.GetTask(id)
	if task == nil {
		t.Fatalf("timeout waiting for task %s status %s: task not found", id, status)
	}
	total := int64(0)
	if task.Meta != nil && task.Meta.Res != nil {
		total = task.Meta.Res.Size
	}
	t.Fatalf("timeout waiting for task %s status %s: got %s downloaded=%d total=%d", id, status, task.Status, task.Progress.Downloaded, total)
}
