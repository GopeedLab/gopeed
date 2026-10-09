package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	gohttp "net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

// serveRangeContent serves byte range requests or full responses from data.
func serveRangeContent(w gohttp.ResponseWriter, r *gohttp.Request, data []byte) {
	w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
	rangeHdr := r.Header.Get(base.HttpHeaderRange)
	total := int64(len(data))
	if rangeHdr == "" {
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", total))
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(data)
		return
	}

	// Parse bytes=start-end or bytes=start-
	if !strings.HasPrefix(rangeHdr, "bytes=") {
		w.WriteHeader(gohttp.StatusBadRequest)
		return
	}
	spec := strings.TrimPrefix(rangeHdr, "bytes=")
	parts := strings.Split(spec, "-")
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= total {
		w.WriteHeader(gohttp.StatusRequestedRangeNotSatisfiable)
		return
	}
	end := total - 1
	if len(parts) > 1 && parts[1] != "" {
		if parsedEnd, err := strconv.ParseInt(parts[1], 10, 64); err == nil && parsedEnd >= start && parsedEnd < total {
			end = parsedEnd
		}
	}

	contentLength := end - start + 1
	w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, total))
	w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", contentLength))
	w.WriteHeader(gohttp.StatusPartialContent)
	_, _ = w.Write(data[start : end+1])
}

func TestFetcher_ChecksumMismatch_Retry(t *testing.T) {
	corruptContent := []byte("Corrupt Initial Download Data 33B")
	correctContent := []byte("Correct Retried Download Data 33B")

	correctSha256Hasher := sha256.New()
	correctSha256Hasher.Write(correctContent)
	correctSha256 := hex.EncodeToString(correctSha256Hasher.Sum(nil))

	var phase atomic.Int32 // 0: serve corruptContent, 1: serve correctContent
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		var served []byte
		if phase.Load() == 0 {
			served = corruptContent
		} else {
			served = correctContent
		}
		serveRangeContent(w, r, served)
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

	// Flip phase to 1 after observing checksum mismatch
	phase.Store(1)

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

func TestFetcher_ChecksumMismatch_Retry_PartialPrefetch(t *testing.T) {
	corruptContent := []byte("Corrupt Initial Download Data 33B")
	correctContent := []byte("Correct Retried Download Data 33B")

	correctSha256Hasher := sha256.New()
	correctSha256Hasher.Write(correctContent)
	correctSha256 := hex.EncodeToString(correctSha256Hasher.Sum(nil))

	var phase atomic.Int32 // 0: corrupt, 1: correct
	var isFirstRequest atomic.Bool
	isFirstRequest.Store(true)

	releasePrefetch := make(chan struct{})
	var closeReleaseOnce sync.Once

	var mu sync.Mutex
	var phase1Ranges []string

	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		curPhase := phase.Load()
		if curPhase == 1 {
			mu.Lock()
			phase1Ranges = append(phase1Ranges, r.Header.Get(base.HttpHeaderRange))
			mu.Unlock()
			serveRangeContent(w, r, correctContent)
			return
		}

		// Phase 0: First request is Resolve GET without Range header.
		// Write exactly 7 bytes and flush, then block until release channel closes or context is done.
		if isFirstRequest.CompareAndSwap(true, false) {
			w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(corruptContent)))
			w.WriteHeader(gohttp.StatusOK)
			_, _ = w.Write(corruptContent[:7])
			if flusher, ok := w.(gohttp.Flusher); ok {
				flusher.Flush()
			}
			select {
			case <-releasePrefetch:
			case <-r.Context().Done():
			}
			return
		}

		// Subsequent requests in Phase 0 (such as Range request from Start)
		serveRangeContent(w, r, corruptContent)
	}))
	defer server.Close()
	defer closeReleaseOnce.Do(func() { close(releasePrefetch) })

	downloadDir := t.TempDir()
	fileName := "test_checksum_prefetch_retry.txt"

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

	// 1. Resolve phase
	if err := f.Resolve(&base.Request{URL: server.URL + "/prefetch_retry.txt"}, opts); err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	// Wait until prefetch has received exactly 7 bytes
	for i := 0; i < 200; i++ {
		if f.prefetchSize.Load() >= 7 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if f.prefetchSize.Load() < 7 {
		t.Fatalf("expected at least 7 prefetched bytes, got %d", f.prefetchSize.Load())
	}

	// Release prefetch handler right before Start
	closeReleaseOnce.Do(func() { close(releasePrefetch) })

	// 2. Start initial download attempt (will complete with corruptContent and fail checksum)
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

	// 3. Flip to phase 1 (correctContent) and retry
	phase.Store(1)

	if err := f.Start(); err != nil {
		t.Fatalf("Start error on retry: %v", err)
	}
	if err := f.Wait(); err != nil {
		t.Fatalf("Wait error on retry (expected success): %v", err)
	}
	if f.getState() != stateDone {
		t.Fatalf("expected stateDone after successful retry, got %v", f.getState())
	}

	// Verify that in Phase 1 at least one Range request started at byte 0
	mu.Lock()
	rangesSeen := append([]string(nil), phase1Ranges...)
	mu.Unlock()

	hasByteZero := false
	for _, r := range rangesSeen {
		if strings.HasPrefix(r, "bytes=0-") {
			hasByteZero = true
			break
		}
	}
	if !hasByteZero {
		t.Fatalf("expected at least one Phase 1 Range request starting at byte 0, saw: %v", rangesSeen)
	}

	// Verify on-disk content matches correctContent exactly
	filePath := f.meta.SingleFilepath()
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read completed file: %v", err)
	}
	if string(fileBytes) != string(correctContent) {
		t.Fatalf("on-disk file content mismatch:\ngot:  %q\nwant: %q", string(fileBytes), string(correctContent))
	}

	// Compute hash independently and verify
	actualHasher := sha256.New()
	actualHasher.Write(fileBytes)
	actualHash := hex.EncodeToString(actualHasher.Sum(nil))
	if actualHash != correctSha256 {
		t.Fatalf("re-computed on-disk hash mismatch: got %s, want %s", actualHash, correctSha256)
	}
}

func TestFetcher_ChecksumMismatch_Retry_NonRange(t *testing.T) {
	corruptContent := []byte("Corrupt Non-Range Initial Data 33B")
	correctContent := []byte("Correct Non-Range Retried Data 33B")

	correctSha256Hasher := sha256.New()
	correctSha256Hasher.Write(correctContent)
	correctSha256 := hex.EncodeToString(correctSha256Hasher.Sum(nil))

	var phase atomic.Int32 // 0: corrupt, 1: correct
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		// Server does not support Range (always returns full content 200 OK)
		var served []byte
		if phase.Load() == 0 {
			served = corruptContent
		} else {
			served = correctContent
		}
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(served)))
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(served)
	}))
	defer server.Close()

	downloadDir := t.TempDir()
	fileName := "test_checksum_nonrange_retry.txt"

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

	// 1. Resolve and start initial attempt
	if err := f.Resolve(&base.Request{URL: server.URL + "/nonrange_retry.txt"}, opts); err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if err := f.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	err := f.Wait()
	if err == nil {
		t.Fatal("expected checksum mismatch error on initial non-range attempt, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected 'checksum mismatch' in error, got %v", err)
	}
	if f.getState() != stateError {
		t.Fatalf("expected stateError after checksum mismatch, got %v", f.getState())
	}

	// 2. Switch phase and retry
	phase.Store(1)

	if err := f.Start(); err != nil {
		t.Fatalf("Start error on non-range retry: %v", err)
	}
	if err := f.Wait(); err != nil {
		t.Fatalf("Wait error on non-range retry (expected success): %v", err)
	}
	if f.getState() != stateDone {
		t.Fatalf("expected stateDone after successful non-range retry, got %v", f.getState())
	}

	// 3. Verify on-disk file content
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
