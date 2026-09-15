package hls

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestFetcherE2E_SameURLTwoTasks(t *testing.T) {
	const total = 6
	mux := http.NewServeMux()
	var playlist strings.Builder
	playlist.WriteString("#EXTM3U\n")
	for i := 0; i < total; i++ {
		playlist.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
	}
	playlist.WriteString("#EXT-X-ENDLIST\n")
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist.String()))
	})
	segments := make(map[int][]byte)
	for i := 0; i < total; i++ {
		segments[i] = []byte(fmt.Sprintf("SHARED-SEGMENT-%02d", i))
		idx := i
		mux.HandleFunc(fmt.Sprintf("/seg-%d.ts", idx), func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(120 * time.Millisecond)
			w.Write(segments[idx])
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := testConfig()
	cfg.SegmentConnections = 2
	cfg.PrefetchContentLength = false
	dir := t.TempDir()

	// Two tasks for the same playlist URL into the same directory, like a
	// user duplicating a task. The engine would rename the second output
	// (AutoRename); mirror that here.
	f1 := newTestFetcher(t, cfg)
	if err := f1.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: dir}); err != nil {
		t.Fatal(err)
	}
	f2 := newTestFetcher(t, cfg)
	if err := f2.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: dir, Name: "video-2.ts"}); err != nil {
		t.Fatal(err)
	}
	if f1.state.StagingDir == "" || f1.state.StagingDir == f2.state.StagingDir {
		t.Fatalf("staging dirs must be distinct per task: %q vs %q", f1.state.StagingDir, f2.state.StagingDir)
	}

	// Start task 1, let it stage some segments, then pause it.
	if err := f1.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := f1.Pause(); err != nil {
		t.Fatal(err)
	}
	staged1, err := os.ReadDir(f1.state.StagingDir)
	if err != nil || len(staged1) == 0 {
		t.Fatalf("task 1 should have staged segments, dir: %s", f1.state.StagingDir)
	}

	// Task 2 must complete without touching task 1's staging data...
	if err := f2.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Wait(); err != nil {
		t.Fatal(err)
	}
	stillThere, err := os.ReadDir(f1.state.StagingDir)
	if err != nil || len(stillThere) != len(staged1) {
		t.Fatalf("task 2 disturbed task 1 staging: had %d entries, now %d", len(staged1), len(stillThere))
	}

	// ...and task 1 resumes to a complete, correct output of its own.
	if err := f1.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f1.Wait(); err != nil {
		t.Fatal(err)
	}
	var want []byte
	for i := 0; i < total; i++ {
		want = append(want, segments[i]...)
	}
	if !bytes.Equal(readOutput(t, f1), want) || !bytes.Equal(readOutput(t, f2), want) {
		t.Error("same-URL task outputs mismatch")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected only the two outputs to remain, got %v", names)
	}
}

func TestFetcherE2E_ForeignJournalDiscarded(t *testing.T) {
	const total = 3
	var served atomic.Int64
	mux := http.NewServeMux()
	var playlist strings.Builder
	playlist.WriteString("#EXTM3U\n")
	for i := 0; i < total; i++ {
		playlist.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
	}
	playlist.WriteString("#EXT-X-ENDLIST\n")
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist.String()))
	})
	for i := 0; i < total; i++ {
		mux.HandleFunc(fmt.Sprintf("/seg-%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			served.Add(1)
			w.Write([]byte("DATA"))
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := testConfig()
	cfg.PrefetchContentLength = false
	f := newTestFetcher(t, cfg)
	if err := f.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	// Plant a journal that does not belong to this plan (wrong fingerprint).
	tempDir := f.resolveStagingDirLocked()
	if err := os.MkdirAll(tempDir, 0777); err != nil {
		t.Fatal(err)
	}
	foreign := journalFile{
		StagingID:    "some-other-task",
		PlanHash:     "deadbeef",
		SegmentCount: total,
		Sizes:        map[string]int64{"0": 4, "1": 4, "2": 4},
	}
	data, _ := json.Marshal(foreign)
	if err := os.WriteFile(filepath.Join(tempDir, "journal.json"), data, 0666); err != nil {
		t.Fatal(err)
	}

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	if got := served.Load(); got != total {
		t.Errorf("foreign journal must be discarded, want %d downloads, got %d", total, got)
	}
}

func TestFetcherE2E_MergeFailurePreservesStaging(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXTINF:4,\na.ts\n#EXTINF:4,\nb.ts\n#EXT-X-ENDLIST\n"))
	})
	mux.HandleFunc("/a.ts", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("AAAA")) })
	mux.HandleFunc("/b.ts", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("BBBB")) })
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	if err := f.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	run := &hlsRun{
		ctx:        context.Background(),
		cancel:     func() {},
		done:       make(chan struct{}),
		segs:       f.state.Segments,
		outputPath: f.meta.SingleFilepath(),
		stagingID:  f.state.StagingID,
		planHash:   planFingerprint(f.state),
		tempDir:    f.resolveStagingDirLocked(),
		keyCache:   make(map[string][]byte),
		journal:    make(map[int64]int64),
	}
	run.journalPath = filepath.Join(run.tempDir, "journal.json")
	if err := os.MkdirAll(run.tempDir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(segmentPath(run.tempDir, 0), []byte("AAAA"), 0666); err != nil {
		t.Fatal(err)
	}
	// Segment 1 is missing on disk: the merge must fail without deleting the
	// staged data or leaving a partial output behind.
	if err := f.merge(run); err == nil {
		t.Fatal("merge with a missing segment should fail")
	}
	if _, err := os.Stat(segmentPath(run.tempDir, 0)); err != nil {
		t.Fatal("merge failure must preserve staged segments")
	}
	entries, _ := os.ReadDir(filepath.Dir(run.outputPath))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".part") {
			t.Errorf("failed merge left a partial output behind: %s", e.Name())
		}
	}

	// Recover the segment: the retried merge succeeds.
	if err := os.WriteFile(segmentPath(run.tempDir, 1), []byte("BBBB"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := f.merge(run); err != nil {
		t.Fatalf("retried merge failed: %v", err)
	}
	got, err := os.ReadFile(run.outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "AAAABBBB" {
		t.Errorf("retried merge output mismatch: %q", got)
	}
}

func TestFetcherE2E_ReMergeOverExistingOutput(t *testing.T) {
	const total = 2
	mux := http.NewServeMux()
	var playlist strings.Builder
	playlist.WriteString("#EXTM3U\n")
	for i := 0; i < total; i++ {
		playlist.WriteString(fmt.Sprintf("#EXTINF:4,\nseg-%d.ts\n", i))
	}
	playlist.WriteString("#EXT-X-ENDLIST\n")
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist.String()))
	})
	for i := 0; i < total; i++ {
		mux.HandleFunc(fmt.Sprintf("/seg-%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("DATA"))
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	if err := f.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	tempDir := f.resolveStagingDirLocked()
	if err := os.MkdirAll(tempDir, 0777); err != nil {
		t.Fatal(err)
	}
	// Simulate the crash window after a completed rename but before the task
	// was marked done: a stale output file, a complete journal and all
	// segments staged.
	stale := filepath.Join(filepath.Dir(f.meta.SingleFilepath()), f.state.OutputName)
	if err := os.WriteFile(stale, []byte("STALE-OUTPUT"), 0666); err != nil {
		t.Fatal(err)
	}
	jr := &hlsRun{
		segs:        f.state.Segments,
		stagingID:   f.state.StagingID,
		planHash:    planFingerprint(f.state),
		journalPath: filepath.Join(tempDir, "journal.json"),
		journal:     map[int64]int64{0: 4, 1: 4},
	}
	if err := saveJournal(jr); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < total; i++ {
		if err := os.WriteFile(segmentPath(tempDir, int64(i)), []byte("DATA"), 0666); err != nil {
			t.Fatal(err)
		}
	}

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatalf("re-merge over an existing output failed: %v", err)
	}
	got, err := os.ReadFile(stale)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "DATADATA" {
		t.Errorf("stale output was not replaced: %q", got)
	}
}
