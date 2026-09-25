package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func TestFetcher_ChecksumMismatch_Retry(t *testing.T) {
	corruptContent := []byte("Corrupt Initial Download Data!")
	correctContent := []byte("Correct Retried Download Data!")

	correctSha256Hasher := sha256.New()
	correctSha256Hasher.Write(correctContent)
	correctSha256 := hex.EncodeToString(correctSha256Hasher.Sum(nil))

	var reqCount atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		count := reqCount.Add(1)
		var served []byte
		if count == 1 {
			// First request: serve corrupted content
			served = corruptContent
		} else {
			// Second / subsequent request: serve correct content
			served = correctContent
		}
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(served)))
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(served)
	}))
	defer server.Close()

	downloadDir := t.TempDir()
	fileName := "test_checksum_retry.txt"

	f := buildFetcher()
	defer f.Close()

	opts := &base.Options{
		Path: downloadDir,
		Name: fileName,
		Extra: &http.OptsExtra{
			Connections: 1,
			Checksum: &http.ChecksumOption{
				Algorithm: "sha256",
				Expected:  correctSha256,
			},
		},
	}

	// 1. Initial attempt -> will download corruptContent, failing checksum verification
	if err := f.Resolve(&base.Request{URL: server.URL + "/retry_test.txt"}, opts); err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	// Ensure the reusable resolve response has been fully prefetched so the
	// first attempt deterministically verifies the corrupt initial payload.
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for !f.prefetchDone.Load() {
		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for checksum test prefetch")
		case <-ticker.C:
		}
	}
	if err := f.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	err := f.Wait()
	if err == nil {
		t.Fatal("expected checksum mismatch error on initial attempt, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected 'checksum mismatch' in error, got %v", err)
	}
	if f.getState() != stateError {
		t.Fatalf("expected stateError after checksum mismatch, got %v", f.getState())
	}

	// 2. Retry download -> should trigger forced full reset, truncate file, and re-download correctContent
	if err := f.Start(); err != nil {
		t.Fatalf("Start error on retry: %v", err)
	}
	if err := f.Wait(); err != nil {
		t.Fatalf("Wait error on retry (expected success): %v", err)
	}
	if f.getState() != stateDone {
		t.Fatalf("expected stateDone after successful retry, got %v", f.getState())
	}

	// 3. Verify on-disk file content matches the re-downloaded bytes
	filePath := f.meta.SingleFilepath()
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read completed file: %v", err)
	}
	if string(fileBytes) != string(correctContent) {
		t.Fatalf("on-disk file content mismatch:\ngot:  %q\nwant: %q", string(fileBytes), string(correctContent))
	}

	// 4. Compute hash independently and verify
	actualHasher := sha256.New()
	actualHasher.Write(fileBytes)
	actualHash := hex.EncodeToString(actualHasher.Sum(nil))
	if actualHash != correctSha256 {
		t.Fatalf("re-computed on-disk hash mismatch: got %s, want %s", actualHash, correctSha256)
	}
}
