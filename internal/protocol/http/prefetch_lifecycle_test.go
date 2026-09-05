package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

const (
	prefetchProcessModeEnv  = "GOPEED_TEST_PREFETCH_PROCESS_MODE"
	prefetchProcessReadyEnv = "GOPEED_TEST_PREFETCH_PROCESS_READY"
)

// processPrefetchBody first supplies actual data, then announces that the
// prefetch file has been populated and blocks until Close interrupts it.
type processPrefetchBody struct {
	readyPath string
	phase     atomic.Int32
	readyOnce sync.Once
	closeOnce sync.Once
	readyCh   chan error
	closedCh  chan struct{}
}

func newProcessPrefetchBody(readyPath string) *processPrefetchBody {
	return &processPrefetchBody{
		readyPath: readyPath,
		readyCh:   make(chan error, 1),
		closedCh:  make(chan struct{}),
	}
}

func (b *processPrefetchBody) Read(p []byte) (int, error) {
	if b.phase.CompareAndSwap(0, 1) {
		for i := range p {
			p[i] = byte(i % 251)
		}
		return len(p), nil
	}

	b.readyOnce.Do(func() {
		err := os.WriteFile(b.readyPath, []byte("ready"), 0o600)
		b.readyCh <- err
		close(b.readyCh)
	})
	<-b.closedCh
	return 0, io.EOF
}

func (b *processPrefetchBody) Close() error {
	b.closeOnce.Do(func() {
		close(b.closedCh)
	})
	return nil
}

// TestPrefetchTempProcessHelper is run in a subprocess by
// TestPrefetchTempProcessExitLeavesNoNamedData. Do not run it directly.
func TestPrefetchTempProcessHelper(t *testing.T) {
	mode := os.Getenv(prefetchProcessModeEnv)
	if mode == "" {
		return
	}

	body := newProcessPrefetchBody(os.Getenv(prefetchProcessReadyEnv))
	f := &Fetcher{
		resolveResp:    &gohttp.Response{Body: body},
		prefetchStopCh: make(chan struct{}),
		prefetchDoneCh: make(chan struct{}),
	}
	go f.asyncPrefetch()

	select {
	case err := <-body.readyCh:
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "prefetch helper did not populate its temporary file")
		os.Exit(2)
	}

	switch mode {
	case "cleanup":
		f.stopPrefetch()
		f.waitPrefetch()
		f.cleanupPrefetchFile()
		os.Exit(0)
	case "exit":
		// Deliberately omit Go cleanup. The operating system must reclaim the
		// data when a process exits normally too.
		os.Exit(0)
	case "kill":
		for {
			time.Sleep(time.Hour)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %q\n", mode)
		os.Exit(2)
	}
}

func TestPrefetchTempProcessExitLeavesNoNamedData(t *testing.T) {
	for _, mode := range []string{"cleanup", "exit", "kill"} {
		t.Run(mode, func(t *testing.T) {
			baseDir := t.TempDir()
			tempDir := filepath.Join(baseDir, "process-temp")
			if err := os.Mkdir(tempDir, 0o700); err != nil {
				t.Fatal(err)
			}
			readyPath := filepath.Join(baseDir, "ready")

			cmd := exec.Command(os.Args[0], "-test.run=^TestPrefetchTempProcessHelper$")
			cmd.Env = isolatedProcessTempEnv(os.Environ(), tempDir)
			cmd.Env = append(cmd.Env,
				prefetchProcessModeEnv+"="+mode,
				prefetchProcessReadyEnv+"="+readyPath,
			)
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}

			waited := false
			defer func() {
				if !waited {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()

			deadline := time.Now().Add(5 * time.Second)
			for {
				_, err := os.Stat(readyPath)
				if err == nil {
					break
				}
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
				if time.Now().After(deadline) {
					t.Fatalf("timed out waiting for helper: %s", output.String())
				}
				time.Sleep(10 * time.Millisecond)
			}

			if mode == "kill" {
				if err := cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				// A killed child is expected to report a non-success exit status.
				_ = cmd.Wait()
				waited = true
			} else {
				if err := cmd.Wait(); err != nil {
					t.Fatalf("helper failed: %v\n%s", err, output.String())
				}
				waited = true
			}

			paths, files, err := prefetchNamedResidues(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 0 {
				t.Fatalf("process exit left named prefetch data: %v", files)
			}
			if mode == "cleanup" && len(paths) != 0 {
				t.Fatalf("normal cleanup left temporary artifacts: %v", paths)
			}
			if runtime.GOOS != "windows" && len(paths) != 0 {
				t.Fatalf("POSIX process exit left directory entries: %v", paths)
			}
		})
	}
}

func isolatedProcessTempEnv(env []string, tempDir string) []string {
	filtered := make([]string, 0, len(env)+3)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "TMPDIR") || strings.EqualFold(key, "TMP") || strings.EqualFold(key, "TEMP") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, "TMPDIR="+tempDir, "TMP="+tempDir, "TEMP="+tempDir)
}

func prefetchNamedResidues(tempDir string) (paths []string, files []string, err error) {
	matches, err := filepath.Glob(filepath.Join(tempDir, "gopeed-prefetch-*"))
	if err != nil {
		return nil, nil, err
	}
	for _, match := range matches {
		err = filepath.WalkDir(match, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, relErr := filepath.Rel(tempDir, path)
			if relErr != nil {
				return relErr
			}
			paths = append(paths, rel)
			if !entry.IsDir() {
				files = append(files, rel)
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	return paths, files, nil
}

type closeUnblocksReadBody struct {
	startedOnce sync.Once
	closedOnce  sync.Once
	startedCh   chan struct{}
	closedCh    chan struct{}
}

func newCloseUnblocksReadBody() *closeUnblocksReadBody {
	return &closeUnblocksReadBody{
		startedCh: make(chan struct{}),
		closedCh:  make(chan struct{}),
	}
}

func (b *closeUnblocksReadBody) Read([]byte) (int, error) {
	b.startedOnce.Do(func() { close(b.startedCh) })
	<-b.closedCh
	return 0, io.EOF
}

func (b *closeUnblocksReadBody) Close() error {
	b.closedOnce.Do(func() { close(b.closedCh) })
	return nil
}

func TestFetcherCloseInterruptsBlockedPrefetchRead(t *testing.T) {
	body := newCloseUnblocksReadBody()
	f := &Fetcher{
		resolveResp:    &gohttp.Response{Body: body},
		prefetchStopCh: make(chan struct{}),
		prefetchDoneCh: make(chan struct{}),
	}
	go f.asyncPrefetch()

	select {
	case <-body.startedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("prefetch never entered the blocking Read")
	}

	closed := make(chan error, 1)
	go func() { closed <- f.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not interrupt the blocked prefetch Read")
	}

	select {
	case <-f.prefetchDoneCh:
	default:
		t.Fatal("Close returned before the prefetch goroutine exited")
	}
	if f.prefetchFile != nil || f.prefetchFilePath != "" || f.prefetchDirPath != "" {
		t.Fatal("Close returned before cleaning prefetch temporary artifacts")
	}
}

func TestFetcherCompletedPrefetchStillFeedsDownload(t *testing.T) {
	payload := make([]byte, 256*1024+137)
	for i := range payload {
		payload[i] = byte((i*31 + 7) % 251)
	}
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		gohttp.ServeContent(w, r, "source.bin", time.Unix(1, 0), bytes.NewReader(payload))
	}))
	defer server.Close()

	f := buildFetcher()
	t.Cleanup(func() { _ = f.Close() })
	outputDir := t.TempDir()
	if err := f.Resolve(&base.Request{URL: server.URL + "/source.bin"}, &base.Options{
		Path: outputDir,
		Name: "prefetch-output.bin",
		Extra: &fhttp.OptsExtra{
			Connections: 2,
		},
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-f.prefetchDoneCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for completed prefetch")
	}
	if f.prefetchErr != nil {
		t.Fatalf("prefetch failed: %v", f.prefetchErr)
	}

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(outputDir, "prefetch-output.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded data differs after consuming completed prefetch: got %d bytes, want %d", len(got), len(payload))
	}
	if f.prefetchFile != nil || f.prefetchFilePath != "" || f.prefetchDirPath != "" {
		t.Fatal("completed download retained prefetch temporary artifacts")
	}
}
