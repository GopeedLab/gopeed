package http

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	gohttp "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/httpclient/httpclienttest"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func TestFetcher_Resolve(t *testing.T) {
	testResolve(test.StartTestFileServer, test.BuildName, t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: true,
			Files: []*base.FileInfo{
				{
					Name: test.BuildName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	testResolve(test.StartTestCustomServer, "disposition", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.BuildName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	testResolve(test.StartTestCustomServer, "encoded-word", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.TestChineseFileName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	testResolve(test.StartTestCustomServer, "no-encode", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.TestChineseFileName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	testResolve(test.StartTestCustomServer, "%E6%B5%8B%E8%AF%95.zip", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  0,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.TestChineseFileName,
					Size: 0,
				},
			},
		}, nil
	})
	testResolve(test.StartTestCustomServer, test.BuildName, t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  0,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.BuildName,
					Size: 0,
				},
			},
		}, nil
	})
	// Test mixed encoding Content-Disposition where mime.ParseMediaType fails
	// due to invalid characters, but filename*= contains the correct UTF-8 encoded name
	testResolve(test.StartTestCustomServer, "mixed-encoding", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.TestChineseFileName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	// Test filename*= only (RFC 5987 format)
	testResolve(test.StartTestCustomServer, "filename-star", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.TestChineseFileName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	// Test GBK-encoded filename (common on Chinese Windows servers)
	// Before fix: "测试.zip" sent as GBK bytes -> parsed as "²âÊÔ.zip" (garbled)
	// After fix: correctly decoded back to "测试.zip"
	testResolve(test.StartTestCustomServer, "gbk-encoded", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: test.TestChineseFileName,
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	// Test filename with plus signs (e.g., C++ Primer)
	// Before fix: %2B decoded to space -> "C++ Primer" became "C  Primer"
	// After fix: %2B correctly decoded to + -> "C++  Primer  Plus.mobi"
	testResolve(test.StartTestCustomServer, "plus-sign-encoded", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: "C++  Primer  Plus.mobi",
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	// Test filename with plus sign in URL path
	// Before fix: %2B decoded to space
	// After fix: %2B correctly decoded to +
	testResolve(test.StartTestCustomServer, "C%2B%2B%20Primer.txt", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  0,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: "C++ Primer.txt",
					Size: 0,
				},
			},
		}, nil
	})
	// Test filename with HTML-encoded ampersand (fixes issue with & being truncated)
	// Before fix: "查询处理&amp;优化.pptx" -> "查询处理&amp" (truncated at semicolon)
	// After fix: correctly decoded to "查询处理&优化.pptx"
	testResolve(test.StartTestCustomServer, "ampersand-encoded", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: "查询处理&优化.pptx",
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	// Test unquoted filename with HTML-encoded ampersand
	testResolve(test.StartTestCustomServer, "ampersand-unquoted", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  test.BuildSize,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: "test&file.txt",
					Size: test.BuildSize,
				},
			},
		}, nil
	})
	// Test URL without file path - should use domain/host as filename
	testResolve(test.StartTestCustomServer, "", t, func(err error) (*base.Resource, error) {
		return &base.Resource{
			Size:  0,
			Range: false,
			Files: []*base.FileInfo{
				{
					Name: "127.0.0.1",
					Size: 0,
				},
			},
		}, nil
	})
	// Test 403 Forbidden response handling
	testResolve(test.StartTestCustomServer, "forbidden", t, func(err error) (*base.Resource, error) {
		requestError := extractRequestError(err)
		if requestError != nil && requestError.Code == 403 {
			return nil, nil
		}
		return nil, err
	})
}

func TestFetcher_ResolveWithHostHeader(t *testing.T) {
	listener := test.StartTestHostHeaderServer()
	defer listener.Close()

	fetcher := buildFetcher()
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/",
		Extra: &http.ReqExtra{
			Header: map[string]string{
				"Host": "test",
			},
		},
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
	})
	// The server should return 400 for invalid Host header
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("Resolve() got = %v, want error containing 400", err)
	}
}

func TestFetcher_ResolveWithInvalidHeader(t *testing.T) {
	listener := test.StartTestCustomServer()
	defer listener.Close()

	fetcher := buildFetcher()
	defer fetcher.Pause() // Close the resolve response to allow server shutdown
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/",
		Extra: &http.ReqExtra{
			Header: map[string]string{
				"Referer": "\rtest",
			},
		},
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
	})
	// Invalid header with \r should be sanitized by Go's http client, allowing the request to succeed
	if err != nil {
		t.Errorf("Resolve() got = %v, want nil (invalid headers should be sanitized)", err)
	}
}

func TestFetcher_ResolveAutomaticallyFallsBackToBrowserFingerprint(t *testing.T) {
	server := httpclienttest.NewFingerprintServer(t, httpclienttest.FingerprintServerOptions{
		RequiredProfile: httpclienttest.ProfileAnyBrowser,
		Handler: gohttp.HandlerFunc(func(w gohttp.ResponseWriter, _ *gohttp.Request) {
			w.Header().Set(base.HttpHeaderContentLength, "4")
			w.Header().Set(base.HttpHeaderContentDisposition, `attachment; filename="file.bin"`)
			_, _ = w.Write([]byte("data"))
		}),
	})

	fetcher := buildFetcher()
	defer fetcher.Pause()
	err := fetcher.Resolve(&base.Request{
		URL:            server.URL + "/file.bin",
		SkipVerifyCert: true,
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if fetcher.meta.Res == nil || len(fetcher.meta.Res.Files) != 1 {
		t.Fatalf("resource = %+v, want one file", fetcher.meta.Res)
	}
	if fetcher.meta.Res.Files[0].Name != "file.bin" {
		t.Fatalf("file name = %q, want file.bin", fetcher.meta.Res.Files[0].Name)
	}
}

func TestFetcherSharesAndClearsImpersonationSession(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, request *gohttp.Request) {
		requests.Add(1)
		if request.Header.Get("Sec-Ch-Ua") == "" {
			w.WriteHeader(gohttp.StatusForbidden)
			return
		}
		w.WriteHeader(gohttp.StatusOK)
	}))
	defer server.Close()

	fetcher := buildFetcher()
	fetcher.meta.Req = &base.Request{URL: server.URL}

	for range 2 {
		client := fetcher.buildClient()
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		response.Body.Close()
		client.CloseIdleConnections()
	}
	if requests.Load() != 3 {
		t.Fatalf("requests before completion = %d, want 3", requests.Load())
	}

	if err := fetcher.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	client := fetcher.buildClient()
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() after close error = %v", err)
	}
	response.Body.Close()
	client.CloseIdleConnections()
	if requests.Load() != 5 {
		t.Fatalf("requests after close = %d, want 5", requests.Load())
	}
}

func testResolve(startTestServer func() net.Listener, path string, t *testing.T, wantFn func(error) (*base.Resource, error)) {
	listener := startTestServer()
	defer listener.Close()
	fetcher := buildFetcher()
	defer fetcher.Pause() // Close the resolve response to allow server shutdown
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + path,
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
	})
	want, err := wantFn(err)
	if err != nil {
		t.Fatal(err)
	}
	if want != nil && !test.AssertResourceEqual(want, fetcher.meta.Res) {
		t.Errorf("Resolve() got = %+v, want %+v", fetcher.meta.Res, want)
	}
}

func TestFetcher_DownloadNormal(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)
	downloadNormal(listener, 5, t)
	downloadNormal(listener, 8, t)
	downloadNormal(listener, 16, t)
}

func TestFetcher_DownloadContinue(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadContinue(listener, 1, t)
	downloadContinue(listener, 5, t)
	downloadContinue(listener, 8, t)
	downloadContinue(listener, 16, t)
}

func TestFetcher_PauseBeforeStartDiscardsPrefetchProgress(t *testing.T) {
	payload := make([]byte, 32*1024*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	prefetchServed := make(chan struct{}, 1)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if r.Header.Get(base.HttpHeaderRange) == "" {
			w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(payload)))
			w.WriteHeader(gohttp.StatusOK)
			if _, err := w.Write(payload); err == nil {
				prefetchServed <- struct{}{}
			}
			return
		}
		gohttp.ServeContent(w, r, test.BuildName, time.Unix(0, 0), bytes.NewReader(payload))
	}))
	defer server.Close()

	fetcher := buildFetcher()
	if err := fetcher.Resolve(&base.Request{URL: server.URL + "/" + test.BuildName}, &base.Options{
		Path: t.TempDir(),
		Name: test.DownloadName,
		Extra: &http.OptsExtra{
			Connections: 2,
		},
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-prefetchServed:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for resolve prefetch response")
	}

	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.Wait(); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("%x", md5.Sum(payload))
	got := test.FileMd5(fetcher.Meta().SingleFilepath())
	if want != got {
		t.Fatalf("download after pause before start got MD5 %s, want %s", got, want)
	}
}

func TestFetcher_DownloadContinue_NoRangeRestart(t *testing.T) {
	listener := test.StartTestNoRangeSlowServer(time.Millisecond)
	defer listener.Close()

	fetcher := downloadReady(listener, 4, t)
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Millisecond)

	stats := fetcher.Stats().Snapshot.(*http.Stats)
	if len(stats.Connections) != 1 {
		t.Fatalf("expected a single non-range connection, got %d", len(stats.Connections))
	}
	if stats.Connections[0].Downloaded <= 0 || stats.Connections[0].Downloaded >= test.BuildSize {
		t.Fatalf("expected partial download before pause, got %d", stats.Connections[0].Downloaded)
	}

	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.Wait(); err != nil {
		t.Fatal(err)
	}

	finalStats := fetcher.Stats().Snapshot.(*http.Stats)
	if len(finalStats.Connections) != 1 {
		t.Fatalf("expected a single non-range connection after resume, got %d", len(finalStats.Connections))
	}
	if finalStats.Connections[0].Downloaded != test.BuildSize {
		t.Fatalf("downloaded bytes should restart cleanly: got %d, want %d", finalStats.Connections[0].Downloaded, test.BuildSize)
	}
	if total := fetcher.Progress().TotalDownloaded(); total != test.BuildSize {
		t.Fatalf("progress total = %d, want %d", total, test.BuildSize)
	}

	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func TestFetcher_DownloadNoRangeLeavesNoStaleCompletion(t *testing.T) {
	listener := test.StartTestNoRangeSlowServer(0)
	defer listener.Close()

	f := downloadReady(listener, 1, t).(*Fetcher)
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-f.doneCh:
		t.Fatalf("stale completion signal: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestFetcher_DownloadIgnoredRangeFallsBackToSequential(t *testing.T) {
	for _, connections := range []int{1, 4} {
		t.Run(fmt.Sprintf("connections_%d", connections), func(t *testing.T) {
			os.Remove(test.DownloadFile)
			t.Cleanup(func() { os.Remove(test.DownloadFile) })

			listener := test.StartTestIgnoredRangeServer(time.Millisecond)
			defer listener.Close()

			f := downloadReady(listener, connections, t).(*Fetcher)
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if err := f.Wait(); err != nil {
				t.Fatal(err)
			}

			if f.meta.Res.Range {
				t.Fatal("expected ignored Range response to disable range mode")
			}
			stats := f.Stats().Snapshot.(*http.Stats)
			if len(stats.Connections) != 1 {
				t.Fatalf("expected one sequential connection, got %d", len(stats.Connections))
			}
			if got, want := test.FileMd5(test.DownloadFile), test.FileMd5(test.BuildFile); got != want {
				t.Fatalf("download checksum = %s, want %s", got, want)
			}
		})
	}
}

func TestFetcher_SequentialRetryReprobesRange(t *testing.T) {
	payload := make([]byte, 4*1024*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	shortReplacement := make([]byte, 2*1024*1024)
	chunkedReplacement := make([]byte, 5*1024*1024)
	for i := range shortReplacement {
		shortReplacement[i] = byte((i*3 + 17) % 251)
	}
	for i := range chunkedReplacement {
		chunkedReplacement[i] = byte((i*7 + 29) % 251)
	}

	const (
		etagA        = `"range-reprobe-a"`
		etagB        = `"range-reprobe-b"`
		weakETag     = `W/"range-reprobe-b"`
		lastModified = "Wed, 21 Oct 2015 07:28:00 GMT"
		laterDate    = "Wed, 21 Oct 2015 07:28:02 GMT"
		failedSize   = 128 * 1024
	)

	tests := []struct {
		name                   string
		resolveETag            string
		ignoredETag            string
		lastModified           string
		date                   string
		wantIfRange            string
		resumeStatus           int
		wantRangeMode          bool
		wantParallel           bool
		wantRangeRequest       int32
		wantFullRestart        bool
		replacement            []byte
		fullRestartReplacement []byte
		chunked200             bool
	}{
		{
			name:             "ignored response replaces Resolve ETag",
			resolveETag:      etagA,
			ignoredETag:      etagB,
			wantIfRange:      etagB,
			resumeStatus:     gohttp.StatusPartialContent,
			wantRangeMode:    true,
			wantParallel:     true,
			wantRangeRequest: 2,
		},
		{
			name:             "strong Last-Modified resumes with 206",
			lastModified:     lastModified,
			date:             laterDate,
			wantIfRange:      lastModified,
			resumeStatus:     gohttp.StatusPartialContent,
			wantRangeMode:    true,
			wantParallel:     true,
			wantRangeRequest: 2,
		},
		{
			name:                   "missing validator restarts from zero",
			resumeStatus:           gohttp.StatusPartialContent,
			wantRangeRequest:       1,
			wantFullRestart:        true,
			fullRestartReplacement: shortReplacement,
		},
		{
			name:             "weak ETag blocks Last-Modified fallback",
			ignoredETag:      weakETag,
			lastModified:     lastModified,
			date:             laterDate,
			resumeStatus:     gohttp.StatusPartialContent,
			wantRangeRequest: 1,
			wantFullRestart:  true,
		},
		{
			name:             "same-second Last-Modified restarts from zero",
			lastModified:     lastModified,
			date:             lastModified,
			resumeStatus:     gohttp.StatusPartialContent,
			wantRangeRequest: 1,
			wantFullRestart:  true,
		},
		{
			name:             "guarded probe returning shorter 200 replaces resource",
			ignoredETag:      etagB,
			wantIfRange:      etagB,
			resumeStatus:     gohttp.StatusOK,
			wantRangeRequest: 2,
			replacement:      shortReplacement,
		},
		{
			name:             "guarded probe returning larger chunked 200 replaces resource",
			ignoredETag:      etagB,
			wantIfRange:      etagB,
			resumeStatus:     gohttp.StatusOK,
			wantRangeRequest: 2,
			replacement:      chunkedReplacement,
			chunked200:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rangeRequests atomic.Int32
			var fullRequests atomic.Int32
			var resumedAt atomic.Int64
			var sawIfRange atomic.Bool
			var workerValidatorMismatch atomic.Bool

			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				rangeHeader := r.Header.Get(base.HttpHeaderRange)
				w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)

				if rangeHeader == "" {
					requestNumber := fullRequests.Add(1)
					responsePayload := payload
					if requestNumber > 1 && tt.fullRestartReplacement != nil {
						responsePayload = tt.fullRestartReplacement
					}
					if requestNumber == 1 && tt.resolveETag != "" {
						w.Header().Set(base.HttpHeaderETag, tt.resolveETag)
					}
					w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(responsePayload)))
					w.WriteHeader(gohttp.StatusOK)
					if requestNumber == 1 {
						if flusher, ok := w.(gohttp.Flusher); ok {
							flusher.Flush()
						}
						for offset := 0; offset < len(payload); offset += 8 * 1024 {
							end := min(offset+8*1024, len(payload))
							if _, err := w.Write(payload[offset:end]); err != nil {
								return
							}
							time.Sleep(time.Millisecond)
						}
						return
					}
					_, _ = w.Write(responsePayload)
					return
				}

				requestNumber := rangeRequests.Add(1)
				if requestNumber == 1 {
					// Simulate the real origin: it first ignores Range, then drops the
					// sequential response after a useful prefix has been written.
					if tt.ignoredETag != "" {
						w.Header().Set(base.HttpHeaderETag, tt.ignoredETag)
					}
					if tt.lastModified != "" {
						w.Header().Set(base.HttpHeaderLastModified, tt.lastModified)
					}
					if tt.date != "" {
						w.Header().Set("Date", tt.date)
					}
					w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(payload)))
					w.WriteHeader(gohttp.StatusOK)
					if flusher, ok := w.(gohttp.Flusher); ok {
						flusher.Flush()
					}
					_, _ = w.Write(payload[:failedSize])
					return
				}

				var start, end int64
				if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
					w.WriteHeader(gohttp.StatusBadRequest)
					return
				}
				if requestNumber == 2 {
					resumedAt.Store(start)
					sawIfRange.Store(r.Header.Get(base.HttpHeaderIfRange) == tt.wantIfRange)
				} else if tt.wantParallel && r.Header.Get(base.HttpHeaderIfRange) != tt.wantIfRange {
					workerValidatorMismatch.Store(true)
				}
				if requestNumber == 2 && tt.resumeStatus == gohttp.StatusOK {
					responsePayload := payload
					if tt.replacement != nil {
						responsePayload = tt.replacement
					}
					w.Header().Set(base.HttpHeaderETag, `"replacement"`)
					if !tt.chunked200 {
						w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(responsePayload)))
					}
					w.WriteHeader(gohttp.StatusOK)
					if tt.chunked200 {
						w.(gohttp.Flusher).Flush()
					}
					for offset := 0; offset < len(responsePayload); offset += 8 * 1024 {
						end := min(offset+8*1024, len(responsePayload))
						if _, err := w.Write(responsePayload[offset:end]); err != nil {
							return
						}
					}
					return
				}
				w.Header().Set(base.HttpHeaderContentRange,
					fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
				w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", end-start+1))
				w.WriteHeader(gohttp.StatusPartialContent)
				if requestNumber == 2 && tt.wantParallel {
					w.(gohttp.Flusher).Flush()
					deadline := time.Now().Add(2 * time.Second)
					for rangeRequests.Load() < 3 && time.Now().Before(deadline) {
						time.Sleep(10 * time.Millisecond)
					}
				}
				for offset := start; offset <= end; offset += 8 * 1024 {
					chunkEnd := min(offset+8*1024, end+1)
					if _, err := w.Write(payload[offset:chunkEnd]); err != nil {
						return
					}
					time.Sleep(time.Millisecond)
				}
			}))
			defer server.Close()

			f := buildFetcher()
			if err := f.Resolve(&base.Request{URL: server.URL + "/dynamic-range.data"}, &base.Options{
				Path: t.TempDir(),
				Name: "dynamic-range.data",
				Extra: &http.OptsExtra{
					Connections: 4,
				},
			}); err != nil {
				t.Fatal(err)
			}
			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if err := f.Wait(); err != nil {
				t.Fatal(err)
			}

			if got := rangeRequests.Load(); got < tt.wantRangeRequest {
				t.Fatalf("Range requests = %d, want at least %d", got, tt.wantRangeRequest)
			}
			got, err := os.ReadFile(f.meta.SingleFilepath())
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFullRestart && fullRequests.Load() < 2 {
				t.Fatal("missing validator did not issue a full restart request")
			}
			if tt.wantIfRange != "" && resumedAt.Load() != failedSize {
				t.Fatalf("resume Range started at %d, want %d", resumedAt.Load(), failedSize)
			}
			if tt.wantIfRange != "" && !sawIfRange.Load() {
				t.Fatalf("resume probe did not include If-Range %q", tt.wantIfRange)
			}
			if f.meta.Res.Range != tt.wantRangeMode {
				t.Fatalf("Range mode = %v, want %v", f.meta.Res.Range, tt.wantRangeMode)
			}
			if tt.wantParallel {
				stats := f.Stats().Snapshot.(*http.Stats)
				if len(stats.Connections) <= 1 {
					f.slowStart.mu.Lock()
					totalLaunched := f.slowStart.totalLaunched
					batchPending := f.slowStart.batchPending
					batchReady := f.slowStart.batchReady
					maxConnections := f.slowStart.maxConnections
					nextBatchSize := f.slowStart.nextBatchSize
					paused := f.slowStart.paused
					f.slowStart.mu.Unlock()
					t.Fatalf("successful re-probe kept %d connection, want expansion (requests=%d state=%d launched=%d pending=%d ready=%d max=%d next=%d paused=%v)", len(stats.Connections), rangeRequests.Load(), f.getState(), totalLaunched, batchPending, batchReady, maxConnections, nextBatchSize, paused)
				}
				if workerValidatorMismatch.Load() {
					t.Fatalf("expanded Range worker omitted pinned If-Range %q", tt.wantIfRange)
				}
			}
			wantPayload := payload
			if tt.replacement != nil {
				wantPayload = tt.replacement
			} else if tt.fullRestartReplacement != nil {
				wantPayload = tt.fullRestartReplacement
			}
			if !bytes.Equal(got, wantPayload) {
				t.Fatal("downloaded file differs after sequential recovery")
			}
			if f.meta.Res.Size != int64(len(wantPayload)) || f.meta.Res.Files[0].Size != int64(len(wantPayload)) {
				t.Fatalf("resource size = %d/%d, want %d", f.meta.Res.Size, f.meta.Res.Files[0].Size, len(wantPayload))
			}
			if got := f.Progress().TotalDownloaded(); got != int64(len(wantPayload)) {
				t.Fatalf("progress = %d, want %d", got, len(wantPayload))
			}
			info, err := os.Stat(f.meta.SingleFilepath())
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Size(); got != int64(len(wantPayload)) {
				t.Fatalf("file size = %d, want %d", got, len(wantPayload))
			}
		})
	}
}

func TestFetcher_InitialIgnoredRangeUsesResponseSize(t *testing.T) {
	resolvePayload := bytes.Repeat([]byte("resolve"), 64*1024)
	replacement := bytes.Repeat([]byte("replacement"), 24*1024)
	var rangeRequests atomic.Int32

	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		if r.Header.Get(base.HttpHeaderRange) == "" {
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(resolvePayload)))
			w.WriteHeader(gohttp.StatusOK)
			_, _ = w.Write(resolvePayload)
			return
		}

		rangeRequests.Add(1)
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(replacement)))
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(replacement)
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/ignored-range.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "ignored-range.data",
		Extra: &http.OptsExtra{
			Connections: 4,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, replacement) {
		t.Fatal("initial ignored-Range response did not replace the resolved representation")
	}
	if rangeRequests.Load() != 1 {
		t.Fatalf("Range requests = %d, want 1", rangeRequests.Load())
	}
	wantSize := int64(len(replacement))
	if f.meta.Res.Size != wantSize || f.meta.Res.Files[0].Size != wantSize {
		t.Fatalf("resource size = %d/%d, want %d", f.meta.Res.Size, f.meta.Res.Files[0].Size, wantSize)
	}
	if got := f.Progress().TotalDownloaded(); got != wantSize {
		t.Fatalf("progress = %d, want %d", got, wantSize)
	}
	info, err := os.Stat(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Size(); got != wantSize {
		t.Fatalf("file size = %d, want %d", got, wantSize)
	}
}

func TestFetcher_InterruptedChunkedReplacementRestartsFromZero(t *testing.T) {
	payload := bytes.Repeat([]byte("original"), 512*1024)
	replacement := bytes.Repeat([]byte("replacement"), 256*1024)
	const firstPrefix = 128 * 1024
	const replacementPrefix = 96 * 1024
	var rangeRequests atomic.Int32
	var fullRequests atomic.Int32
	var sawFullRestart atomic.Bool

	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		rangeHeader := r.Header.Get(base.HttpHeaderRange)
		if rangeHeader == "" {
			requestNumber := fullRequests.Add(1)
			responsePayload := payload
			if requestNumber > 1 {
				responsePayload = replacement
				sawFullRestart.Store(true)
			}
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(responsePayload)))
			w.WriteHeader(gohttp.StatusOK)
			_, _ = w.Write(responsePayload)
			return
		}

		requestNumber := rangeRequests.Add(1)
		switch requestNumber {
		case 1:
			w.Header().Set(base.HttpHeaderETag, `"prefix-v1"`)
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(payload)))
			w.WriteHeader(gohttp.StatusOK)
			w.(gohttp.Flusher).Flush()
			_, _ = w.Write(payload[:firstPrefix])
		case 2:
			if got := r.Header.Get(base.HttpHeaderIfRange); got != `"prefix-v1"` {
				t.Errorf("If-Range = %q, want %q", got, `"prefix-v1"`)
			}
			conn, rw, err := w.(gohttp.Hijacker).Hijack()
			if err != nil {
				t.Errorf("Hijack() error = %v", err)
				return
			}
			defer conn.Close()
			_, _ = fmt.Fprintf(rw, "HTTP/1.1 200 OK\r\nETag: \"replacement-v2\"\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n%x\r\n", replacementPrefix)
			_, _ = rw.Write(replacement[:replacementPrefix])
			_, _ = rw.WriteString("\r\n")
			_ = rw.Flush()
		default:
			t.Errorf("unexpected Range retry after unknown-size replacement: %q", rangeHeader)
			w.WriteHeader(gohttp.StatusBadRequest)
		}
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/chunked-replacement.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "chunked-replacement.data",
		Extra: &http.OptsExtra{
			Connections: 4,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, replacement) {
		t.Fatal("downloaded file differs after interrupted chunked replacement")
	}
	if rangeRequests.Load() != 2 {
		t.Fatalf("Range requests = %d, want 2", rangeRequests.Load())
	}
	if !sawFullRestart.Load() {
		t.Fatal("interrupted chunked replacement did not restart with a full request")
	}
	wantSize := int64(len(replacement))
	if f.meta.Res.Size != wantSize || f.meta.Res.Files[0].Size != wantSize {
		t.Fatalf("resource size = %d/%d, want %d", f.meta.Res.Size, f.meta.Res.Files[0].Size, wantSize)
	}
	if got := f.Progress().TotalDownloaded(); got != wantSize {
		t.Fatalf("progress = %d, want %d", got, wantSize)
	}
	info, err := os.Stat(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Size(); got != wantSize {
		t.Fatalf("file size = %d, want %d", got, wantSize)
	}
	if f.sequentialSizeUnknown {
		t.Fatal("completed replacement kept unknown-size recovery state")
	}
}

func TestExtractIfRangeValidator(t *testing.T) {
	const (
		lastModified = "Wed, 21 Oct 2015 07:28:00 GMT"
		laterDate    = "Wed, 21 Oct 2015 07:28:02 GMT"
	)
	tests := []struct {
		name string
		etag string
		lm   string
		date string
		want string
	}{
		{name: "strong ETag", etag: `"v1"`, want: `"v1"`},
		{name: "malformed ETag", etag: `"bad value"`},
		{name: "weak ETag blocks date", etag: `W/"v1"`, lm: lastModified, date: laterDate},
		{name: "strong Last-Modified", lm: lastModified, date: laterDate, want: lastModified},
		{name: "same-second Last-Modified", lm: lastModified, date: lastModified},
		{name: "Last-Modified without Date", lm: lastModified},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := make(gohttp.Header)
			if tt.etag != "" {
				header.Set(base.HttpHeaderETag, tt.etag)
			}
			if tt.lm != "" {
				header.Set(base.HttpHeaderLastModified, tt.lm)
			}
			if tt.date != "" {
				header.Set("Date", tt.date)
			}
			if got := extractIfRangeValidator(header); got != tt.want {
				t.Fatalf("extractIfRangeValidator() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetcher_OriginalURLFallbackPreservesResumeHeaders(t *testing.T) {
	const (
		start   = int64(128)
		end     = int64(255)
		ifRange = `"resume-v2"`
	)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		if got, want := r.Header.Get(base.HttpHeaderRange), "bytes=128-255"; got != want {
			t.Errorf("Range = %q, want %q", got, want)
		}
		if got := r.Header.Get(base.HttpHeaderIfRange); got != ifRange {
			t.Errorf("If-Range = %q, want %q", got, ifRange)
		}
		w.Header().Set(base.HttpHeaderContentRange, "bytes 128-255/256")
		w.WriteHeader(gohttp.StatusPartialContent)
	}))
	defer server.Close()

	f := &Fetcher{
		config: &config{},
		meta:   &fetcher.FetcherMeta{Req: &base.Request{URL: server.URL}},
	}
	rangeEnd := int64(end)
	resp, err := f.tryFallbackToOriginalURL(context.Background(), server.Client(), start, &rangeEnd, true, ifRange)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestFetcher_UsesResolveResponseWhenRangeTakeoverFails(t *testing.T) {
	tests := []struct {
		name         string
		rangeHandler func(gohttp.ResponseWriter, *gohttp.Request)
	}{
		{
			name: "forbidden",
			rangeHandler: func(w gohttp.ResponseWriter, _ *gohttp.Request) {
				w.WriteHeader(gohttp.StatusForbidden)
			},
		},
		{
			name: "invalid partial content",
			rangeHandler: func(w gohttp.ResponseWriter, r *gohttp.Request) {
				var start, end int64
				if _, err := fmt.Sscanf(r.Header.Get(base.HttpHeaderRange), "bytes=%d-%d", &start, &end); err != nil {
					w.WriteHeader(gohttp.StatusBadRequest)
					return
				}
				w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start+1, end, 2*1024*1024))
				w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", end-start))
				w.WriteHeader(gohttp.StatusPartialContent)
			},
		},
		{
			name: "connection closes before response",
			rangeHandler: func(w gohttp.ResponseWriter, _ *gohttp.Request) {
				conn, _, err := w.(gohttp.Hijacker).Hijack()
				if err == nil {
					conn.Close()
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, 2*1024*1024)
			for i := range payload {
				payload[i] = byte(i % 251)
			}

			var requests atomic.Int32
			var rangeRequests atomic.Int32
			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				requestNumber := requests.Add(1)
				if requestNumber == 1 {
					w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
					w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(payload)))
					w.WriteHeader(gohttp.StatusOK)
					w.(gohttp.Flusher).Flush()
					for offset := 0; offset < len(payload); offset += 8 * 1024 {
						end := min(offset+8*1024, len(payload))
						if _, err := w.Write(payload[offset:end]); err != nil {
							return
						}
						time.Sleep(time.Millisecond)
					}
					return
				}

				if r.Header.Get(base.HttpHeaderRange) != "" {
					rangeRequests.Add(1)
				}
				tt.rangeHandler(w, r)
			}))
			defer server.Close()

			f := buildFetcher()
			if err := f.Resolve(&base.Request{URL: server.URL + "/resolve-fallback.data"}, &base.Options{
				Path: t.TempDir(),
				Name: "resolve-fallback.data",
				Extra: &http.OptsExtra{
					Connections: 1,
				},
			}); err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			if err := f.Start(); err != nil {
				t.Fatal(err)
			}
			if err := f.Wait(); err != nil {
				t.Fatalf("download should continue from the Resolve response when Range takeover fails: %v", err)
			}
			if rangeRequests.Load() == 0 {
				t.Fatal("download did not attempt a Range request")
			}
			got, err := os.ReadFile(f.meta.SingleFilepath())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatal("downloaded file differs from the Resolve response")
			}
		})
	}
}

func TestFetcher_ValidRangeTakesOverResolveResponse(t *testing.T) {
	const resolveETag = `"resolve-v1"`
	payload := make([]byte, 4*1024*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	var rangeRequests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		rangeHeader := r.Header.Get(base.HttpHeaderRange)
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		if rangeHeader == "" {
			w.Header().Set(base.HttpHeaderETag, resolveETag)
			w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(payload)))
			w.WriteHeader(gohttp.StatusOK)
			w.(gohttp.Flusher).Flush()
			for offset := 0; offset < len(payload); offset += 8 * 1024 {
				end := min(offset+8*1024, len(payload))
				if _, err := w.Write(payload[offset:end]); err != nil {
					return
				}
				time.Sleep(time.Millisecond)
			}
			return
		}

		rangeRequests.Add(1)
		if got := r.Header.Get(base.HttpHeaderIfRange); got != resolveETag {
			t.Errorf("If-Range = %q, want Resolve ETag %q", got, resolveETag)
		}
		var start, end int64
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
			w.WriteHeader(gohttp.StatusBadRequest)
			return
		}
		w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", end-start+1))
		w.WriteHeader(gohttp.StatusPartialContent)
		for offset := start; offset <= end; offset += 8 * 1024 {
			chunkEnd := min(offset+8*1024, end+1)
			if _, err := w.Write(payload[offset:chunkEnd]); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/resolve-takeover.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "resolve-takeover.data",
		Extra: &http.OptsExtra{
			Connections: 4,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	if rangeRequests.Load() == 0 {
		t.Fatal("download did not attempt a Range takeover")
	}
	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded file differs after valid Range takeover")
	}
}

func TestFetcher_UnknownSizeRangeUsesOpenEndedWorker(t *testing.T) {
	const resolveETag = `"unknown-size-v1"`
	payload := make([]byte, 4*1024*1024)
	for i := range payload {
		payload[i] = byte((i*11 + 7) % 251)
	}

	var openEndedRequests atomic.Int32
	var rangeRequests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		rangeHeader := r.Header.Get(base.HttpHeaderRange)
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		w.Header().Set(base.HttpHeaderETag, resolveETag)
		if rangeHeader == "" {
			w.WriteHeader(gohttp.StatusOK)
			w.(gohttp.Flusher).Flush()
			for offset := 0; offset < len(payload); offset += 8 * 1024 {
				end := min(offset+8*1024, len(payload))
				if _, err := w.Write(payload[offset:end]); err != nil {
					return
				}
				w.(gohttp.Flusher).Flush()
				time.Sleep(2 * time.Millisecond)
			}
			return
		}

		rangeRequests.Add(1)
		if got := r.Header.Get(base.HttpHeaderIfRange); got != resolveETag {
			t.Errorf("If-Range = %q, want %q", got, resolveETag)
		}
		var start, end int64
		if strings.HasSuffix(rangeHeader, "-") {
			openEndedRequests.Add(1)
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil {
				t.Errorf("parse open-ended Range %q: %v", rangeHeader, err)
				w.WriteHeader(gohttp.StatusBadRequest)
				return
			}
			end = int64(len(payload) - 1)
		} else if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
			t.Errorf("parse bounded Range %q: %v", rangeHeader, err)
			w.WriteHeader(gohttp.StatusBadRequest)
			return
		}
		if start > end {
			w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes */%d", len(payload)))
			w.WriteHeader(gohttp.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", end-start+1))
		w.WriteHeader(gohttp.StatusPartialContent)
		for offset := start; offset <= end; offset += 8 * 1024 {
			chunkEnd := min(offset+8*1024, end+1)
			if _, err := w.Write(payload[offset:chunkEnd]); err != nil {
				return
			}
			w.(gohttp.Flusher).Flush()
			time.Sleep(time.Millisecond)
		}
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/unknown-size-range.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "unknown-size-range.data",
		Extra: &http.OptsExtra{
			Connections: 4,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if f.meta.Res.Size != 0 || !f.meta.Res.Range {
		t.Fatalf("resolved resource = %+v, want unknown size with Range support", f.meta.Res)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	if got := openEndedRequests.Load(); got != 1 {
		t.Fatalf("open-ended Range requests = %d, want 1", got)
	}
	if got := rangeRequests.Load(); got < 2 {
		t.Fatalf("Range requests = %d, want slow-start expansion after size discovery", got)
	}
	if got, want := f.meta.Res.Size, int64(len(payload)); got != want {
		t.Fatalf("resource size = %d, want %d", got, want)
	}
	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded file differs after unknown-size Range takeover")
	}
}

func TestFetcher_UnknownSizeRangeRetriesShortPartialBody(t *testing.T) {
	payload := make([]byte, 1024*1024)
	for i := range payload {
		payload[i] = byte((i*13 + 5) % 251)
	}

	var rangeRequests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		rangeHeader := r.Header.Get(base.HttpHeaderRange)
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		w.Header().Set(base.HttpHeaderETag, `"short-range-v1"`)
		if rangeHeader == "" {
			w.WriteHeader(gohttp.StatusOK)
			w.(gohttp.Flusher).Flush()
			for offset := 0; offset < len(payload); offset += 8 * 1024 {
				end := min(offset+8*1024, len(payload))
				if _, err := w.Write(payload[offset:end]); err != nil {
					return
				}
				w.(gohttp.Flusher).Flush()
				time.Sleep(2 * time.Millisecond)
			}
			return
		}

		requestNumber := rangeRequests.Add(1)
		var start, requestedEnd int64
		openEnded := strings.HasSuffix(rangeHeader, "-")
		if openEnded {
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil {
				t.Errorf("parse open-ended Range %q: %v", rangeHeader, err)
				w.WriteHeader(gohttp.StatusBadRequest)
				return
			}
			requestedEnd = int64(len(payload) - 1)
		} else if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &requestedEnd); err != nil {
			t.Errorf("parse Range %q: %v", rangeHeader, err)
			w.WriteHeader(gohttp.StatusBadRequest)
			return
		}

		w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, requestedEnd, len(payload)))
		w.WriteHeader(gohttp.StatusPartialContent)
		w.(gohttp.Flusher).Flush()
		if requestNumber == 1 {
			// End the chunked response cleanly after only part of the declared
			// Content-Range. The worker must retry from the materialized offset.
			shortEnd := min(start+64*1024, requestedEnd+1)
			_, _ = w.Write(payload[start:shortEnd])
			return
		}
		_, _ = w.Write(payload[start : requestedEnd+1])
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/short-range.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "short-range.data",
		Extra: &http.OptsExtra{
			Connections: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	if got := rangeRequests.Load(); got < 2 {
		t.Fatalf("Range requests = %d, want a retry after the short body", got)
	}
	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded file differs after retrying the short Range body")
	}
}

func TestFetcher_UnknownSizeRangeFallsBackToResolve(t *testing.T) {
	payload := make([]byte, 512*1024)
	for i := range payload {
		payload[i] = byte((i*17 + 3) % 251)
	}

	var rangeRequests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		if r.Header.Get(base.HttpHeaderRange) == "" {
			w.WriteHeader(gohttp.StatusOK)
			w.(gohttp.Flusher).Flush()
			for offset := 0; offset < len(payload); offset += 8 * 1024 {
				end := min(offset+8*1024, len(payload))
				if _, err := w.Write(payload[offset:end]); err != nil {
					return
				}
				w.(gohttp.Flusher).Flush()
				time.Sleep(time.Millisecond)
			}
			return
		}

		rangeRequests.Add(1)
		w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes 1-%d/%d", len(payload)-1, len(payload)))
		w.WriteHeader(gohttp.StatusPartialContent)
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/unknown-size-fallback.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "unknown-size-fallback.data",
		Extra: &http.OptsExtra{
			Connections: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatalf("Wait() should use the complete Resolve response: %v", err)
	}
	if got := rangeRequests.Load(); got != 1 {
		t.Fatalf("Range requests = %d, want 1", got)
	}
	if f.meta.Res.Range {
		t.Fatal("Resolve fallback should switch the task to sequential mode")
	}
	if got, want := f.meta.Res.Size, int64(len(payload)); got != want {
		t.Fatalf("resource size = %d, want %d", got, want)
	}
	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded file differs from the unknown-size Resolve fallback")
	}
}

func TestFetcher_UnknownSizeRangeRemainsOpenEndedAfterPause(t *testing.T) {
	payload := make([]byte, 1024*1024)
	for i := range payload {
		payload[i] = byte((i*19 + 1) % 251)
	}

	firstRangeStarted := make(chan struct{})
	var rangeRequests atomic.Int32
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		rangeHeader := r.Header.Get(base.HttpHeaderRange)
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		w.Header().Set(base.HttpHeaderETag, `"pause-open-range-v1"`)
		if rangeHeader == "" {
			w.WriteHeader(gohttp.StatusOK)
			w.(gohttp.Flusher).Flush()
			for offset := 0; offset < len(payload); offset += 8 * 1024 {
				end := min(offset+8*1024, len(payload))
				if _, err := w.Write(payload[offset:end]); err != nil {
					return
				}
				w.(gohttp.Flusher).Flush()
				time.Sleep(2 * time.Millisecond)
			}
			return
		}

		requestNumber := rangeRequests.Add(1)
		if requestNumber == 1 {
			close(firstRangeStarted)
			<-r.Context().Done()
			return
		}
		if !strings.HasSuffix(rangeHeader, "-") {
			t.Errorf("resumed Range = %q, want an open-ended range", rangeHeader)
			w.WriteHeader(gohttp.StatusBadRequest)
			return
		}
		var start int64
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil {
			t.Errorf("parse resumed Range %q: %v", rangeHeader, err)
			w.WriteHeader(gohttp.StatusBadRequest)
			return
		}
		end := int64(len(payload) - 1)
		w.Header().Set(base.HttpHeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", end-start+1))
		w.WriteHeader(gohttp.StatusPartialContent)
		_, _ = w.Write(payload[start:])
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/pause-open-range.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "pause-open-range.data",
		Extra: &http.OptsExtra{
			Connections: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstRangeStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the first open-ended worker")
	}
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	if got := rangeRequests.Load(); got != 2 {
		t.Fatalf("Range requests = %d, want an open-ended retry after pause", got)
	}
	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded file differs after pausing an unknown-size Range")
	}
}

func TestFetcher_UnknownSizeResolveReadErrorFailsTask(t *testing.T) {
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, _ *gohttp.Request) {
		conn, rw, err := w.(gohttp.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack response: %v", err)
			return
		}
		defer conn.Close()
		_, _ = fmt.Fprint(rw, "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n800\r\n")
		_, _ = rw.Write(make([]byte, 1024))
		_ = rw.Flush()
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/broken-chunked.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "broken-chunked.data",
		Extra: &http.OptsExtra{
			Connections: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err == nil {
		t.Fatal("Wait() succeeded after the unknown-size response ended unexpectedly")
	}
	if got := f.getState(); got != stateError {
		t.Fatalf("fetcher state = %v, want %v", got, stateError)
	}
}

func TestFetcher_UnknownSizeResolveCleanEOFUpdatesSize(t *testing.T) {
	payload := bytes.Repeat([]byte("gopeed-unknown-size\n"), 8192)
	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, _ *gohttp.Request) {
		w.WriteHeader(gohttp.StatusOK)
		w.(gohttp.Flusher).Flush()
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	f := buildFetcher()
	if err := f.Resolve(&base.Request{URL: server.URL + "/unknown-size.data"}, &base.Options{
		Path: t.TempDir(),
		Name: "unknown-size.data",
		Extra: &http.OptsExtra{
			Connections: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	if got, want := f.meta.Res.Size, int64(len(payload)); got != want {
		t.Fatalf("resource size = %d, want %d", got, want)
	}
	if got, want := f.meta.Res.Files[0].Size, int64(len(payload)); got != want {
		t.Fatalf("file size = %d, want %d", got, want)
	}
	got, err := os.ReadFile(f.meta.SingleFilepath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded file differs for a clean unknown-size response")
	}
}

func TestFetcher_RejectsInvalidPartialContent(t *testing.T) {
	const (
		requestedStart = int64(1024)
		requestedEnd   = int64(2047)
		totalSize      = int64(4096)
		rangeLength    = requestedEnd - requestedStart + 1
	)

	tests := []struct {
		name          string
		contentRange  string
		contentLength int64
		bodyLength    int
		chunked       bool
		wantError     bool
	}{
		{
			name:          "valid",
			contentRange:  "bytes 1024-2047/4096",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
		},
		{
			name:          "missing content range",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "invalid unit",
			contentRange:  "items 1024-2047/4096",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "malformed content range",
			contentRange:  "bytes 1024/4096",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "missing total separator",
			contentRange:  "bytes 1024-2047",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "unknown total size",
			contentRange:  "bytes 1024-2047/*",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "shifted content range",
			contentRange:  "bytes 0-1023/4096",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "wrong total size",
			contentRange:  "bytes 1024-2047/8192",
			contentLength: rangeLength,
			bodyLength:    int(rangeLength),
			wantError:     true,
		},
		{
			name:          "content length mismatch",
			contentRange:  "bytes 1024-2047/4096",
			contentLength: rangeLength - 1,
			bodyLength:    int(rangeLength - 1),
			wantError:     true,
		},
		{
			name:          "long chunked body",
			contentRange:  "bytes 1024-2047/4096",
			contentLength: -1,
			bodyLength:    int(rangeLength + 1),
			chunked:       true,
			wantError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(gohttp.HandlerFunc(func(writer gohttp.ResponseWriter, request *gohttp.Request) {
				if got, want := request.Header.Get(base.HttpHeaderRange), "bytes=1024-2047"; got != want {
					t.Errorf("Range header = %q, want %q", got, want)
				}
				if tc.contentRange != "" {
					writer.Header().Set(base.HttpHeaderContentRange, tc.contentRange)
				}
				if tc.contentLength >= 0 {
					writer.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", tc.contentLength))
				}
				writer.WriteHeader(base.HttpCodePartialContent)
				if tc.chunked {
					writer.(gohttp.Flusher).Flush()
				}
				_, _ = writer.Write([]byte(strings.Repeat("x", tc.bodyLength)))
			}))
			defer server.Close()

			f := buildFetcher()
			f.meta.Req = &base.Request{URL: server.URL}
			f.meta.Res = &base.Resource{Range: true, Size: totalSize}
			output, err := os.CreateTemp(t.TempDir(), "range-response-*")
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			f.file = output

			conn := &connection{
				ID:    0,
				Role:  rolePrimary,
				State: connNotStarted,
				Chunk: newChunk(requestedStart, requestedEnd),
				ctx:   context.Background(),
			}
			f.connections = []*connection{conn}
			f.wg.Add(1)
			go f.runConnection(conn)
			f.wg.Wait()

			if tc.wantError {
				if !conn.failed || conn.State != connFailed || conn.Completed {
					t.Fatalf("connection state = %v, failed = %v, completed = %v; want permanent failure", conn.State, conn.failed, conn.Completed)
				}
				if !errors.Is(conn.lastErr, errInvalidRangeResponse) {
					t.Fatalf("connection error = %v, want %v", conn.lastErr, errInvalidRangeResponse)
				}
				return
			}

			if conn.failed || conn.State != connCompleted || !conn.Completed {
				t.Fatalf("connection state = %v, failed = %v, completed = %v; want success", conn.State, conn.failed, conn.Completed)
			}
			if conn.Downloaded != rangeLength {
				t.Fatalf("downloaded = %d, want %d", conn.Downloaded, rangeLength)
			}
		})
	}
}

func TestFetcher_InvalidResponseLengthFailsTask(t *testing.T) {
	const (
		totalSize      = int64(4096)
		requestedStart = int64(1024)
		requestedEnd   = totalSize - 1
		rangeLength    = requestedEnd - requestedStart + 1
	)

	tests := []struct {
		name               string
		status             int
		contentRange       string
		contentLength      int64
		bodyChunks         []int
		resolvedDownloaded int64
		wantDownloaded     int64
		rangeMode          bool
	}{
		{
			name:           "sequential declared short body",
			status:         base.HttpCodeOK,
			contentLength:  1024,
			bodyChunks:     []int{1024},
			wantDownloaded: 0,
		},
		{
			name:           "sequential short chunked body",
			status:         base.HttpCodeOK,
			contentLength:  -1,
			bodyChunks:     []int{1024},
			wantDownloaded: 1024,
		},
		{
			name:           "sequential long chunked body",
			status:         base.HttpCodeOK,
			contentLength:  -1,
			bodyChunks:     []int{int(totalSize), 1},
			wantDownloaded: totalSize,
		},
		{
			name:               "partial content long chunked body",
			status:             base.HttpCodePartialContent,
			contentRange:       "bytes 1024-4095/4096",
			contentLength:      -1,
			bodyChunks:         []int{int(rangeLength), 1},
			resolvedDownloaded: requestedStart,
			wantDownloaded:     rangeLength,
			rangeMode:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(gohttp.HandlerFunc(func(writer gohttp.ResponseWriter, request *gohttp.Request) {
				wantRange := ""
				if tc.rangeMode {
					wantRange = "bytes=1024-4095"
				}
				if got := request.Header.Get(base.HttpHeaderRange); got != wantRange {
					t.Errorf("Range header = %q, want %q", got, wantRange)
				}
				if tc.contentRange != "" {
					writer.Header().Set(base.HttpHeaderContentRange, tc.contentRange)
				}
				if tc.contentLength >= 0 {
					writer.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", tc.contentLength))
				}
				writer.WriteHeader(tc.status)
				writer.(gohttp.Flusher).Flush()
				for i, chunkSize := range tc.bodyChunks {
					_, _ = writer.Write([]byte(strings.Repeat("x", chunkSize)))
					writer.(gohttp.Flusher).Flush()
					if i+1 < len(tc.bodyChunks) {
						time.Sleep(50 * time.Millisecond)
					}
				}
			}))
			defer server.Close()

			f := buildFetcher()
			f.meta.Req = &base.Request{URL: server.URL}
			f.meta.Res = &base.Resource{Range: tc.rangeMode, Size: totalSize}
			output, err := os.CreateTemp(t.TempDir(), "range-task-*")
			if err != nil {
				t.Fatal(err)
			}
			if err := output.Truncate(totalSize); err != nil {
				t.Fatal(err)
			}
			f.file = output

			connChunk := newChunk(0, totalSize-1)
			if tc.rangeMode {
				connChunk = newChunk(requestedStart, requestedEnd)
			}
			conn := &connection{
				ID:    0,
				Role:  rolePrimary,
				State: connNotStarted,
				Chunk: connChunk,
				ctx:   context.Background(),
			}
			f.connections = []*connection{conn}
			if tc.resolvedDownloaded > 0 {
				f.resolveConn = &connection{
					Role:       roleResolve,
					State:      connCompleted,
					Downloaded: tc.resolvedDownloaded,
					Completed:  true,
				}
			}

			f.wg.Add(1)
			go f.runConnection(conn)
			f.wg.Wait()
			f.onDownloadComplete()

			if err := f.Wait(); !errors.Is(err, errInvalidRangeResponse) {
				t.Fatalf("Wait() error = %v, want %v", err, errInvalidRangeResponse)
			}
			if f.getState() != stateError {
				t.Fatalf("fetcher state = %v, want %v", f.getState(), stateError)
			}
			if conn.Completed || conn.State != connFailed || !conn.failed {
				t.Fatalf("connection state = %v, failed = %v, completed = %v; want permanent failure", conn.State, conn.failed, conn.Completed)
			}
			if conn.Downloaded != tc.wantDownloaded {
				t.Fatalf("downloaded = %d, want %d", conn.Downloaded, tc.wantDownloaded)
			}
		})
	}
}

func TestFetcher_DownloadChunked(t *testing.T) {
	listener := test.StartTestCustomServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)
	downloadNormal(listener, 2, t)
}

func TestFetcher_DownloadPost(t *testing.T) {
	listener := test.StartTestPostServer()
	defer listener.Close()

	downloadPost(listener, 1, t)
}

func TestFetcher_DownloadRetry(t *testing.T) {
	listener := test.StartTestRetryServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)
}

func TestFetcher_DownloadError(t *testing.T) {
	listener := test.StartTestErrorServer()
	defer listener.Close()

	downloadError(listener, 1, t)
}

func TestFetcher_DownloadLimit(t *testing.T) {
	listener := test.StartTestConLimitServer(4)
	defer listener.Close()

	downloadNormal(listener, 1, t)
	downloadNormal(listener, 2, t)
	downloadNormal(listener, 8, t)
}

func TestFetcher_DownloadResponseBodyReadTimeout(t *testing.T) {
	// Server will timeout once (first request delays longer than readTimeout),
	// then subsequent requests work normally
	listener := test.StartTestTimeoutOnceServer(readTimeout.Milliseconds() + 5000)
	defer listener.Close()

	for _, connections := range []int{1, 4} {
		os.Remove(test.DownloadFile)

		fetcher := downloadReady(listener, connections, t)
		if err := fetcher.Start(); err != nil {
			t.Fatal(err)
		}
		if err := fetcher.Wait(); err != nil {
			t.Fatal(err)
		}

		stats := fetcher.Stats().Snapshot.(*http.Stats)
		if len(stats.Connections) == 0 {
			t.Fatalf("expected connections stats for timeout test")
		}

		// Verify successful download after timeout recovery
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("Download() got = %v, want %v", got, want)
		}

		// Verify timeouts don't count as failures (retryTimes should be 0)
		for _, conn := range stats.Connections {
			if conn.Failed {
				t.Fatalf("expected no counted failures after timeout recovery, got retries=%d", conn.RetryTimes)
			}
			if conn.RetryTimes != 0 {
				t.Fatalf("expected retryTimes to stay zero for non-counted timeouts, got %d", conn.RetryTimes)
			}
		}
	}
}

func TestFetcher_Download500Recovery(t *testing.T) {
	// Server returns 500 for 15 seconds, then recovers
	listener := test.StartTestTemporary500Server(15 * time.Second)
	defer listener.Close()

	os.Remove(test.DownloadFile)
	fetcher := downloadReady(listener, 4, t)
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.Wait(); err != nil {
		t.Fatal(err)
	}

	// Verify successful download after 500 errors
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}

	// Verify 500 errors don't count as failures (retryTimes should be 0)
	stats := fetcher.Stats().Snapshot.(*http.Stats)
	for _, conn := range stats.Connections {
		if conn.RetryTimes != 0 {
			t.Errorf("Expected retryTimes to be 0 for 500 errors (exempt), got %d", conn.RetryTimes)
		}
	}
}

func TestFetcher_DownloadOnBugFileServer(t *testing.T) {
	listener := test.StartTestRangeBugServer()
	defer listener.Close()

	downloadNormal(listener, 1, t)

	fetcher := downloadReady(listener, 4, t)
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}
	if err := fetcher.Wait(); err == nil || !strings.Contains(err.Error(), errInvalidRangeResponse.Error()) {
		t.Fatalf("Wait() error = %v, want %v", err, errInvalidRangeResponse)
	}
}

func TestFetcher_DownloadResume(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloadResume(listener, 1, t)
	downloadResume(listener, 5, t)
	downloadResume(listener, 8, t)
	downloadResume(listener, 16, t)
}

func TestFetcher_DownloadWithProxy(t *testing.T) {
	httpListener := test.StartTestFileServer()
	defer httpListener.Close()
	proxyListener := test.StartSocks5Server("", "")
	defer proxyListener.Close()

	downloadWithProxy(httpListener, proxyListener, t)
}

func TestFetcher_ConfigConnections(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	fetcher := doDownloadReady(buildConfigFetcher(config{
		Connections: 16,
	}), listener, 0, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func TestFetcher_ConfigUseServerCtime(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	fetcher := doDownloadReady(buildConfigFetcher(config{
		Connections:    16,
		UseServerCtime: true,
	}), listener, 0, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func TestFetcher_Stats(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()
	fetcher := doDownloadReady(buildConfigFetcher(config{
		Connections: 16,
	}), listener, 0, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	stats := fetcher.Stats().Snapshot.(*http.Stats)
	// With slow-start strategy, connection count may be less than max if download is fast
	// Just verify we have at least 1 connection and no more than max
	if len(stats.Connections) < 1 || len(stats.Connections) > 16 {
		t.Errorf("Stats() connection count got = %v, want between 1 and 16", len(stats.Connections))
	}
	totalDownloaded := int64(0)
	expectedConnectionTotal := test.BuildSize / int64(len(stats.Connections))
	if test.BuildSize%int64(len(stats.Connections)) != 0 {
		expectedConnectionTotal++
	}
	for i, conn := range stats.Connections {
		t.Logf("Connection %d: Downloaded=%d, Completed=%v", i, conn.Downloaded, conn.Completed)
		totalDownloaded += conn.Downloaded
		if conn.Total != expectedConnectionTotal {
			t.Errorf("connection %d total = %d, want shared total %d", i, conn.Total, expectedConnectionTotal)
		}
	}
	if totalDownloaded != test.BuildSize {
		t.Errorf("Stats() got = %v, want %v", totalDownloaded, test.BuildSize)
	}
}

func TestFetcher_StatsMarksTerminalNonFailedConnectionsComplete(t *testing.T) {
	fetcher := &Fetcher{
		meta: &fetcher.FetcherMeta{Res: &base.Resource{Size: 4096, Range: true}},
		connections: []*connection{
			{State: connDownloading, Chunk: newChunk(0, 2047), Downloaded: 1024},
			{State: connCompleted, Chunk: newChunk(2048, 3071), Downloaded: 2048},
			{State: connFailed, Chunk: newChunk(3072, 4095), Downloaded: 512, failed: true, retryTimes: 3},
		},
	}
	fetcher.setState(stateDone)

	stats := fetcher.Stats().Snapshot.(*http.Stats)
	if !stats.Connections[0].Completed {
		t.Fatal("terminal non-failed connection was reported as downloading")
	}
	if !stats.Connections[1].Completed {
		t.Fatal("completed connection was not reported as completed")
	}
	if stats.Connections[2].Completed || !stats.Connections[2].Failed {
		t.Fatal("failed connection lost its failure state")
	}
	const averageTotal = int64(1366) // ceil(4096 / 3)
	for index, connection := range stats.Connections {
		if connection.Total != averageTotal {
			t.Fatalf("connection %d total = %d, want shared total %d", index, connection.Total, averageTotal)
		}
	}
}

func TestFetcher_StatsUsesUnknownTotalWithoutResourceSize(t *testing.T) {
	fetcher := &Fetcher{
		meta: &fetcher.FetcherMeta{Res: &base.Resource{}},
		connections: []*connection{
			{State: connDownloading, Chunk: newChunk(0, 0), Downloaded: 1024},
		},
	}

	stats := fetcher.Stats().Snapshot.(*http.Stats)
	if stats.Connections[0].Total != 0 {
		t.Fatalf("unknown resource total = %d, want 0", stats.Connections[0].Total)
	}
}

func TestFetcher_StatsExposesCompletedResolvePrefetch(t *testing.T) {
	fetcher := &Fetcher{meta: &fetcher.FetcherMeta{Res: &base.Resource{Size: 100}}}
	fetcher.resolveDataPos.Store(100)
	fetcher.setState(stateDone)

	stats := fetcher.Stats().Snapshot.(*http.Stats)
	if len(stats.Connections) != 1 {
		t.Fatalf("connection count = %d, want 1", len(stats.Connections))
	}
	connection := stats.Connections[0]
	if connection.Downloaded != 100 || connection.Total != 100 || !connection.Completed {
		t.Fatalf("unexpected prefetch connection: %+v", connection)
	}
}

// TestFetcher_DownloadOneTimeURL tests downloading from a URL that can only be accessed once
// This simulates signed URLs or one-time download links that expire after first use
func TestFetcher_DownloadOneTimeURL(t *testing.T) {
	listener := test.StartTestOneTimeServer()
	defer listener.Close()

	fetcher := buildFetcher()
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
		Extra: &http.OptsExtra{
			Connections: 4, // Try to use multiple connections, but only first should work
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}

	// Verify file content
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

// TestFetcher_SlowStartExpansion tests slow-start connection expansion edge cases
// Tests that slow-start expansion reaches exactly maxConns
// Expansion pattern: 1 -> 2 -> 4 -> 8 -> 16...
// For max=5: 1 -> 2 -> 4 -> 5 (capped)
// For max=9: 1 -> 2 -> 4 -> 8 -> 9 (capped)
func TestFetcher_SlowStartExpansion(t *testing.T) {
	testCases := []struct {
		name     string
		maxConns int
	}{
		{"MaxConns5", 5}, // 1->2->4->5
		{"MaxConns9", 9}, // 1->2->4->8->9
		{"MaxConns8", 8}, // 1->2->4->8
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Clean up any leftover files from previous tests
			os.Remove(test.DownloadFile)

			// Use 100ns delay per byte for faster test (~10MB/s theoretical)
			listener := test.StartTestLowSpeedServer(100 * time.Nanosecond)

			// Ensure cleanup happens before next subtest
			cleanup := func() {
				listener.Close()
				os.Remove(test.DownloadFile)
				// Wait for server to fully stop
				time.Sleep(50 * time.Millisecond)
			}

			fetcher := buildConfigFetcher(config{
				Connections: tc.maxConns,
			})

			err := fetcher.Resolve(&base.Request{
				URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
			}, &base.Options{
				Name: test.DownloadName,
				Path: test.Dir,
				Extra: &http.OptsExtra{
					Connections: tc.maxConns,
				},
			})
			if err != nil {
				cleanup()
				t.Fatal(err)
			}

			err = fetcher.Start()
			if err != nil {
				cleanup()
				t.Fatal(err)
			}

			err = fetcher.Wait()
			if err != nil {
				t.Logf("Wait() returned error: %v", err)
				cleanup()
				t.Fatal(err)
			}

			// Check final connection count equals maxConns exactly
			stats := fetcher.Stats().Snapshot.(*http.Stats)
			finalConns := len(stats.Connections)

			// Debug: show connection details and metadata
			httpFetcher := fetcher.(*Fetcher)
			t.Logf("Resource: Size=%d, Range=%v", httpFetcher.Meta().Res.Size, httpFetcher.Meta().Res.Range)
			for i, conn := range stats.Connections {
				t.Logf("Connection %d: Downloaded=%d, Completed=%v", i, conn.Downloaded, conn.Completed)
			}

			if finalConns != tc.maxConns {
				t.Errorf("Expected exactly %d connections, got %d", tc.maxConns, finalConns)
			}

			// Verify file content before cleanup
			want := test.FileMd5(test.BuildFile)
			got := test.FileMd5(test.DownloadFile)
			if want != got {
				t.Errorf("Download() got = %v, want %v", got, want)
			}

			cleanup()
		})
	}
}

// TestFetcher_AsyncPrefetch tests the async prefetch functionality
// where data is downloaded in background during resolve phase and reused in start
func TestFetcher_AsyncPrefetch(t *testing.T) {
	// Test 1: Prefetch completes entire file before Start is called
	t.Run("PrefetchComplete", func(t *testing.T) {
		listener := test.StartTestFileServer()
		defer listener.Close()

		fetcher := buildFetcher()
		err := fetcher.Resolve(&base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}, &base.Options{
			Name: test.DownloadName,
			Path: test.Dir,
			Extra: &http.OptsExtra{
				Connections: 4,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Poll until prefetch completes the entire file (with timeout)
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
	pollLoop:
		for {
			select {
			case <-timeout:
				t.Fatal("Timeout waiting for prefetch to complete")
			case <-ticker.C:
				if fetcher.prefetchSize.Load() >= test.BuildSize {
					break pollLoop
				}
			}
		}

		prefetchedBefore := fetcher.prefetchSize.Load()
		t.Logf("Prefetched bytes before Start: %d (%.2f MB)", prefetchedBefore, float64(prefetchedBefore)/(1024*1024))

		// Should have prefetched the entire file
		if prefetchedBefore != test.BuildSize {
			t.Errorf("Prefetch should complete entire file, got %d, want %d", prefetchedBefore, test.BuildSize)
		}

		// Now start the download
		err = fetcher.Start()
		if err != nil {
			t.Fatal(err)
		}

		// Wait for download to complete
		err = fetcher.Wait()
		if err != nil {
			t.Fatal(err)
		}

		// Check how much was utilized from prefetch
		prefetchedUsed := fetcher.resolveDataPos.Load()
		t.Logf("Prefetched bytes used: %d (%.2f MB)", prefetchedUsed, float64(prefetchedUsed)/(1024*1024))

		// Verify file is correct
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("Download() got = %v, want %v", got, want)
		}

		os.Remove(test.DownloadFile)
	})

	// Test 2: Prefetch only downloads partial data before Start is called
	t.Run("PrefetchPartial", func(t *testing.T) {
		// Use slow server with 100 nanosecond delay per byte
		// This means ~10MB/s speed, so 100ms should download ~1MB
		listener := test.StartTestLowSpeedServer(100 * time.Nanosecond)
		defer listener.Close()

		fetcher := buildFetcher()
		err := fetcher.Resolve(&base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}, &base.Options{
			Name: test.DownloadName,
			Path: test.Dir,
			Extra: &http.OptsExtra{
				Connections: 4,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Wait only 100ms - should only prefetch a small portion
		time.Sleep(100 * time.Millisecond)

		prefetchedBefore := fetcher.prefetchSize.Load()
		t.Logf("Prefetched bytes before Start: %d (%.2f KB)", prefetchedBefore, float64(prefetchedBefore)/1024)

		// Verify we have partial data (not zero, but not complete)
		if prefetchedBefore == 0 {
			t.Log("Warning: No data prefetched, may be too slow")
		}
		if prefetchedBefore >= test.BuildSize {
			t.Log("Warning: Prefetch completed entire file, test may not be valid")
		}

		// Now start the download
		err = fetcher.Start()
		if err != nil {
			t.Fatal(err)
		}

		// Wait for download to complete
		err = fetcher.Wait()
		if err != nil {
			t.Fatal(err)
		}

		// Check stats - should have connections that downloaded remaining data
		stats := fetcher.Stats().Snapshot.(*http.Stats)
		t.Logf("Final connections: %d", len(stats.Connections))

		prefetchedUsed := fetcher.resolveDataPos.Load()
		t.Logf("Prefetched bytes used: %d (%.2f KB)", prefetchedUsed, float64(prefetchedUsed)/1024)

		// Verify connections picked up where prefetch left off
		if len(stats.Connections) > 0 {
			firstConn := stats.Connections[0]
			t.Logf("First connection downloaded: %d bytes", firstConn.Downloaded)
		}

		// Verify file is correct
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("Download() got = %v, want %v", got, want)
		}

		os.Remove(test.DownloadFile)
	})
}

// TestFetcher_DownloadExpiringRedirectURL tests that the fetcher correctly handles
// expiring redirect URLs by falling back to the original URL and getting a new redirect.
func TestFetcher_DownloadExpiringRedirectURL(t *testing.T) {
	// Test with redirect expiring after 2 requests
	// This means:
	// - Request 1: Resolve (original URL redirects to temp URL v1)
	// - Request 2: First download request to temp URL v1 succeeds
	// - Request 3: Second download request to temp URL v1 returns 403
	// - Fetcher should then retry with original URL, get temp URL v2
	// - Continue downloading from temp URL v2
	t.Run("RedirectExpiresAfter2Requests", func(t *testing.T) {
		os.Remove(test.DownloadFile)

		// Create server with redirect expiring after 2 requests, with slow transfer to ensure
		// multiple connection attempts are needed
		listener := test.StartTestExpiringRedirectServer(2, 100*time.Nanosecond)
		defer listener.Close()

		fetcher := buildFetcher()
		err := fetcher.Resolve(&base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}, &base.Options{
			Name: test.DownloadName,
			Path: test.Dir,
			Extra: &http.OptsExtra{
				Connections: 4, // Use multiple connections to trigger redirect expiration
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		err = fetcher.Start()
		if err != nil {
			t.Fatal(err)
		}

		err = fetcher.Wait()
		if err != nil {
			t.Fatal(err)
		}

		// Verify file content is correct despite redirect expiration
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("Download() got = %v, want %v", got, want)
		}

		os.Remove(test.DownloadFile)
	})

	// Test with redirect expiring after 5 requests (more room for initial connections)
	t.Run("RedirectExpiresAfter5Requests", func(t *testing.T) {
		os.Remove(test.DownloadFile)

		listener := test.StartTestExpiringRedirectServer(5, 100*time.Nanosecond)
		defer listener.Close()

		fetcher := buildFetcher()
		err := fetcher.Resolve(&base.Request{
			URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		}, &base.Options{
			Name: test.DownloadName,
			Path: test.Dir,
			Extra: &http.OptsExtra{
				Connections: 8,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		err = fetcher.Start()
		if err != nil {
			t.Fatal(err)
		}

		err = fetcher.Wait()
		if err != nil {
			t.Fatal(err)
		}

		// Verify file content
		want := test.FileMd5(test.BuildFile)
		got := test.FileMd5(test.DownloadFile)
		if want != got {
			t.Errorf("Download() got = %v, want %v", got, want)
		}

		os.Remove(test.DownloadFile)
	})
}

// TestFetcher_RetryAfterError tests that the fetcher can retry downloading
// after a previous download attempt failed by calling Start() again.
func TestFetcher_RetryAfterError(t *testing.T) {
	os.Remove(test.DownloadFile)

	// Server fails first 3 requests (after resolve), then recovers
	// With 1 connection and 3 retries:
	// - First Start(): requests 2, 3, 4 → all fail (3 retries exhausted) → returns error
	// - Second Start(): request 5 → succeeds (server recovered after 3 failures)
	listener := test.StartTestFailThenRecoverServer(3)
	defer listener.Close()

	fetcher := buildFetcher()
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
		Extra: &http.OptsExtra{
			Connections: 1, // Use single connection to simplify test
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// This test exercises retrying a failed ranged download. Discard the live
	// Resolve response first so it cannot satisfy the task as a fallback.
	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}

	// First download attempt - should fail because server returns 416 after resolve
	err = fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}

	err = fetcher.Wait()
	// First attempt should fail with 416 error
	if err == nil {
		t.Fatal("Expected first download attempt to fail, but it succeeded")
	}
	t.Logf("First attempt failed as expected: %v", err)

	// Check that fetcher is in error state
	state := fetcher.getState()
	if state != stateError {
		t.Errorf("Expected fetcher to be in stateError, got %v", state)
	}

	// Verify that we can call Start() again after error
	// This tests the stateError handling in Start()
	err = fetcher.Start()
	if err != nil {
		t.Fatalf("Start() after error failed: %v", err)
	}

	// Wait for second attempt - should succeed now that server has recovered
	err = fetcher.Wait()
	if err != nil {
		t.Fatalf("Retry failed: %v", err)
	}

	// Verify file content
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}

	os.Remove(test.DownloadFile)
}

func TestFetcherManager_ParseName(t *testing.T) {
	type args struct {
		u string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "broken url",
			args: args{
				u: "https://!@#%github.com",
			},
			want: "",
		},
		{
			name: "file path",
			args: args{
				u: "https://github.com/index.html",
			},
			want: "index.html",
		},
		{
			name: "file path with query and hash",
			args: args{
				u: "https://github.com/a/b/index.html/#list?name=1",
			},
			want: "index.html",
		},
		{
			name: "no file path",
			args: args{
				u: "https://github.com",
			},
			want: "github.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := &FetcherManager{}
			if got := fm.ParseName(tt.args.u); got != tt.want {
				t.Errorf("ParseName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetcherManager_StoreCreatesAtomicSnapshot(t *testing.T) {
	f := &Fetcher{
		meta: &fetcher.FetcherMeta{
			Req: &base.Request{},
			Res: &base.Resource{
				Size:  100,
				Range: true,
				Files: []*base.FileInfo{{Size: 100}},
			},
			Opts: &base.Options{},
		},
		ifRange:              `"snapshot-v1"`,
		rangeValidatorPinned: true,
		connections: []*connection{{
			ID:    0,
			Role:  rolePrimary,
			State: connDownloading,
			Chunk: newChunk(0, 99),
		}},
	}
	fm := new(FetcherManager)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sequential := false
		for {
			select {
			case <-stop:
				return
			default:
			}
			f.connMu.Lock()
			if sequential {
				f.meta.Res.Range = false
				f.meta.Res.Size = 0
				f.meta.Res.Files[0].Size = 0
				f.rangeReprobeEligible = true
				f.rangeValidatorPinned = false
				f.sequentialSizeUnknown = true
				f.connections[0].Downloaded = 64
				f.connections[0].Chunk.Downloaded = 64
			} else {
				f.meta.Res.Range = true
				f.meta.Res.Size = 100
				f.meta.Res.Files[0].Size = 100
				f.rangeReprobeEligible = false
				f.rangeValidatorPinned = true
				f.sequentialSizeUnknown = false
				f.connections[0].Downloaded = 0
				f.connections[0].Chunk.Downloaded = 0
			}
			f.connMu.Unlock()
			sequential = !sequential
		}
	}()

	for i := 0; i < 1000; i++ {
		data, err := fm.Store(f)
		if err != nil {
			close(stop)
			<-done
			t.Fatal(err)
		}
		fd := data.(*fetcherData)
		if fd.Range == nil {
			close(stop)
			<-done
			t.Fatal("snapshot Range mode is nil")
		}
		if fd.ResourceSize == nil || fd.FileSize == nil {
			close(stop)
			<-done
			t.Fatal("snapshot sizes are nil")
		}
		if len(fd.Connections) != 1 || fd.Connections[0] == nil || fd.Connections[0].Chunk == nil {
			close(stop)
			<-done
			t.Fatalf("invalid connection snapshot: %#v", fd.Connections)
		}
		if fd.Connections[0] == f.connections[0] || fd.Connections[0].Chunk == f.connections[0].Chunk {
			close(stop)
			<-done
			t.Fatal("Store returned live connection state instead of a deep snapshot")
		}

		downloaded := fd.Connections[0].Chunk.Downloaded
		if *fd.Range {
			if fd.RangeReprobeEligible || !fd.RangeValidatorPinned || fd.SequentialSizeUnknown || downloaded != 0 || *fd.ResourceSize != 100 || *fd.FileSize != 100 {
				close(stop)
				<-done
				t.Fatalf("mixed ranged snapshot: eligible=%v pinned=%v unknown=%v downloaded=%d size=%d/%d", fd.RangeReprobeEligible, fd.RangeValidatorPinned, fd.SequentialSizeUnknown, downloaded, *fd.ResourceSize, *fd.FileSize)
			}
		} else if !fd.RangeReprobeEligible || fd.RangeValidatorPinned || !fd.SequentialSizeUnknown || downloaded != 64 || *fd.ResourceSize != 0 || *fd.FileSize != 0 {
			close(stop)
			<-done
			t.Fatalf("mixed sequential snapshot: eligible=%v pinned=%v unknown=%v downloaded=%d size=%d/%d", fd.RangeReprobeEligible, fd.RangeValidatorPinned, fd.SequentialSizeUnknown, downloaded, *fd.ResourceSize, *fd.FileSize)
		}
	}
	close(stop)
	<-done
}

func TestFetcherManager_RestoreUsesSnapshotRangeMode(t *testing.T) {
	fm := new(FetcherManager)
	rangeMode := false
	resourceSize := int64(0)
	fileSize := int64(0)
	saved := &fetcherData{
		Connections: []*connection{{
			ID:         0,
			Role:       rolePrimary,
			State:      connFailed,
			Chunk:      newChunk(0, 99),
			Downloaded: 64,
		}},
		IfRange:               `"snapshot-v1"`,
		RangeReprobeEligible:  true,
		SequentialSizeUnknown: true,
		Range:                 &rangeMode,
		ResourceSize:          &resourceSize,
		FileSize:              &fileSize,
	}
	saved.Connections[0].Chunk.Downloaded = 64
	encoded, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	var persisted fetcherData
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}

	staleMeta := &fetcher.FetcherMeta{
		Req: &base.Request{},
		Res: &base.Resource{
			Size:  100,
			Range: true,
			Files: []*base.FileInfo{{Size: 100}},
		},
		Opts: &base.Options{},
	}
	_, restore := fm.Restore()
	restored := restore(staleMeta, &persisted).(*Fetcher)
	if restored.meta.Res.Range {
		t.Fatal("Restore kept stale task Range mode instead of the connection snapshot mode")
	}
	if !restored.rangeReprobeEligible || !restored.sequentialSizeUnknown || restored.ifRange != `"snapshot-v1"` {
		t.Fatalf("restored recovery state = eligible:%v unknown:%v If-Range:%q", restored.rangeReprobeEligible, restored.sequentialSizeUnknown, restored.ifRange)
	}
	if restored.meta.Res.Size != 0 || restored.meta.Res.Files[0].Size != 0 {
		t.Fatalf("restored snapshot size = %d/%d, want 0/0", restored.meta.Res.Size, restored.meta.Res.Files[0].Size)
	}
	if got := restored.connections[0].Chunk.Downloaded; got != 64 {
		t.Fatalf("restored prefix = %d, want 64", got)
	}
	restored.connMu.Lock()
	canProbe := restored.canProbeSequentialResumeLocked(restored.connections[0])
	restored.connMu.Unlock()
	if canProbe {
		t.Fatal("unknown-size restored state attempted an If-Range resume against stale size")
	}

	pinnedRangeMode := true
	pinnedMeta := &fetcher.FetcherMeta{
		Req:  &base.Request{},
		Res:  &base.Resource{Size: 100, Range: false},
		Opts: &base.Options{},
	}
	pinned := restore(pinnedMeta, &fetcherData{
		IfRange:              `"snapshot-v2"`,
		RangeValidatorPinned: true,
		Range:                &pinnedRangeMode,
	}).(*Fetcher)
	if !pinned.meta.Res.Range || !pinned.rangeValidatorPinned || pinned.ifRange != `"snapshot-v2"` {
		t.Fatalf("restored pinned range state = range:%v pinned:%v If-Range:%q", pinned.meta.Res.Range, pinned.rangeValidatorPinned, pinned.ifRange)
	}

	// Records written before the Range snapshot field existed must retain the
	// mode from task metadata for backward compatibility.
	legacyMeta := &fetcher.FetcherMeta{
		Req:  &base.Request{},
		Res:  &base.Resource{Size: 100, Range: true},
		Opts: &base.Options{},
	}
	legacy := restore(legacyMeta, &fetcherData{}).(*Fetcher)
	if !legacy.meta.Res.Range {
		t.Fatal("legacy record unexpectedly overrode task Range mode")
	}
}

func TestFetcherManager_RestoreUsesAtomicSizeSnapshotForDownload(t *testing.T) {
	original := bytes.Repeat([]byte("old-representation"), 16*1024)
	replacement := bytes.Repeat([]byte("new-representation"), 12*1024)
	const savedPrefix = 64 * 1024
	const replacementPrefix = 96 * 1024
	var sawGuardedProbe atomic.Bool

	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		wantRange := fmt.Sprintf("bytes=%d-%d", savedPrefix, len(original)-1)
		if got := r.Header.Get(base.HttpHeaderRange); got != wantRange {
			t.Errorf("Range = %q, want %q", got, wantRange)
		}
		if got := r.Header.Get(base.HttpHeaderIfRange); got != `"old-v1"` {
			t.Errorf("If-Range = %q, want %q", got, `"old-v1"`)
		}
		sawGuardedProbe.Store(true)
		w.Header().Set(base.HttpHeaderETag, `"new-v2"`)
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(replacement)))
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(replacement)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	const filename = "restored-replacement.data"
	filepath := tempDir + string(os.PathSeparator) + filename
	if err := os.WriteFile(filepath, replacement[:replacementPrefix], 0o644); err != nil {
		t.Fatal(err)
	}

	rangeMode := false
	resourceSize := int64(len(original))
	fileSize := resourceSize
	saved := &fetcherData{
		Connections: []*connection{{
			ID:         0,
			Role:       rolePrimary,
			State:      connFailed,
			Chunk:      newChunk(0, resourceSize-1),
			Downloaded: savedPrefix,
		}},
		IfRange:              `"old-v1"`,
		RangeReprobeEligible: true,
		Range:                &rangeMode,
		ResourceSize:         &resourceSize,
		FileSize:             &fileSize,
	}
	saved.Connections[0].Chunk.Downloaded = savedPrefix
	// Simulate task metadata serialized after an interrupted chunked replacement
	// while the fetcher snapshot still describes the preceding representation.
	staleMeta := &fetcher.FetcherMeta{
		Req: &base.Request{URL: server.URL},
		Res: &base.Resource{
			Size:  0,
			Range: false,
			Files: []*base.FileInfo{{Name: filename, Size: 0}},
		},
		Opts: &base.Options{
			Path: tempDir,
			Name: filename,
			Extra: &http.OptsExtra{
				Connections: 1,
			},
		},
	}

	fm := new(FetcherManager)
	_, restore := fm.Restore()
	restored := restore(staleMeta, saved).(*Fetcher)
	if restored.meta.Res.Size != resourceSize || restored.meta.Res.Files[0].Size != fileSize {
		t.Fatalf("restored size = %d/%d, want %d/%d", restored.meta.Res.Size, restored.meta.Res.Files[0].Size, resourceSize, fileSize)
	}
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		if err := json.Unmarshal([]byte(test.ToJson(fm.DefaultConfig())), v); err != nil {
			t.Fatal(err)
		}
	}
	restored.Setup(ctl)
	if err := restored.Start(); err != nil {
		t.Fatal(err)
	}
	if err := restored.Wait(); err != nil {
		t.Fatal(err)
	}

	if !sawGuardedProbe.Load() {
		t.Fatal("restored download did not issue the guarded Range probe")
	}
	got, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, replacement) {
		t.Fatal("restored download retained bytes from the interrupted replacement")
	}
	wantSize := int64(len(replacement))
	if restored.meta.Res.Size != wantSize || restored.meta.Res.Files[0].Size != wantSize {
		t.Fatalf("completed size = %d/%d, want %d", restored.meta.Res.Size, restored.meta.Res.Files[0].Size, wantSize)
	}
	if got := restored.Progress().TotalDownloaded(); got != wantSize {
		t.Fatalf("progress = %d, want %d", got, wantSize)
	}
	info, err := os.Stat(filepath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Size(); got != wantSize {
		t.Fatalf("file size = %d, want %d", got, wantSize)
	}
}

func downloadReady(listener net.Listener, connections int, t *testing.T) fetcher.Fetcher {
	return doDownloadReady(buildFetcher(), listener, connections, t)
}

func doDownloadReady(f fetcher.Fetcher, listener net.Listener, connections int, t *testing.T) fetcher.Fetcher {
	var extra any = nil
	if connections > 0 {
		extra = &http.OptsExtra{
			Connections: connections,
		}
	}
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: extra,
	}
	err := f.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}, opts)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func downloadNormal(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadPost(listener net.Listener, connections int, t *testing.T) {
	// POST parameters must be set before Resolve since the new design
	// starts downloading during Resolve phase
	f := buildFetcher()
	var extra any = nil
	if connections > 0 {
		extra = &http.OptsExtra{
			Connections: connections,
		}
	}
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: extra,
	}
	req := &base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		Extra: &http.ReqExtra{
			Method: "POST",
			Header: map[string]string{
				"Authorization": "Bearer 123456",
			},
			Body: fmt.Sprintf(`{"name":"%s"}`, test.BuildName),
		},
	}
	err := f.Resolve(req, opts)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = f.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadContinue(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	if err := fetcher.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	if err := fetcher.Start(); err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadError(listener net.Listener, connections int, t *testing.T) {
	fetcher := buildFetcher()
	err := fetcher.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}, &base.Options{
		Name: test.DownloadName,
		Path: test.Dir,
	})
	// With the new async design, Resolve may succeed (HTTP response received)
	// but errors occur during async download or Start/Wait
	if err != nil {
		// Error detected in Resolve - this is fine
		return
	}

	// Resolve succeeded, error should occur during Start/Wait
	err = fetcher.Start()
	if err != nil {
		// Error detected in Start - this is fine
		return
	}

	err = fetcher.Wait()
	if err == nil {
		t.Errorf("Expected error during download, but got none")
	}
}

func downloadResume(listener net.Listener, connections int, t *testing.T) {
	fetcher := downloadReady(listener, connections, t)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}

	fb := new(FetcherManager)
	time.Sleep(time.Millisecond * 50)
	data, err := fb.Store(fetcher)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 50)
	fetcher.Pause()

	_, f := fb.Restore()
	f(fetcher.Meta(), data)
	if err != nil {
		t.Fatal(err)
	}
	fetcher.Setup(controller.NewController())
	fetcher.Start()

	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func downloadWithProxy(httpListener net.Listener, proxyListener net.Listener, t *testing.T) {
	fetcher := downloadReady(httpListener, 4, t)
	ctl := controller.NewController()
	ctl.GetProxy = func(requestProxy *base.RequestProxy) func(*gohttp.Request) (*url.URL, error) {
		return (&base.DownloaderProxyConfig{
			Enable: true,
			Scheme: "socks5",
			Host:   proxyListener.Addr().String(),
		}).ToHandler()
	}
	fetcher.Setup(ctl)
	err := fetcher.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = fetcher.Wait()
	if err != nil {
		t.Fatal(err)
	}
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

func buildFetcher() *Fetcher {
	fm := new(FetcherManager)
	fetcher := fm.Build()
	newController := controller.NewController()
	newController.GetConfig = func(v any) {
		json.Unmarshal([]byte(test.ToJson(fm.DefaultConfig())), v)
	}
	fetcher.Setup(newController)
	return fetcher.(*Fetcher)
}

func buildConfigFetcher(cfg config) fetcher.Fetcher {
	fetcher := new(FetcherManager).Build()
	newController := controller.NewController()
	newController.GetConfig = func(v any) {
		json.Unmarshal([]byte(test.ToJson(cfg)), v)
	}
	fetcher.Setup(newController)
	return fetcher
}

// TestFetcher_Patch_URLChange tests the Patch functionality where a failed download URL
// is replaced with a working one. This simulates:
// 1. Initial download attempt with a bad URL (returns 404)
// 2. Patching the task with a new working URL
// 3. Successful download after URL modification
func TestFetcher_Patch_URLChange(t *testing.T) {
	listener := test.StartTestPatchURLServer()
	defer listener.Close()

	f := buildFetcher()
	badURL := "http://" + listener.Addr().String() + "/bad-url"
	goodURL := "http://" + listener.Addr().String() + "/good-url"

	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: &http.OptsExtra{Connections: 1},
	}

	// Step 1: Try to resolve with bad URL - should fail with error
	err := f.Resolve(&base.Request{URL: badURL}, opts)
	if err == nil {
		t.Fatal("Expected error for bad URL, got nil")
	}

	// Step 2: Create a new fetcher and resolve with bad URL but don't wait
	// We need to test patching a task that has been created
	f2 := buildFetcher()

	// First resolve with good URL to create a valid fetcher state
	err = f2.Resolve(&base.Request{URL: goodURL}, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Verify initial URL
	if f2.meta.Req.URL != goodURL {
		t.Errorf("Initial URL = %v, want %v", f2.meta.Req.URL, goodURL)
	}

	// Step 3: Patch to change URL (simulating URL change scenario)
	newURL := "http://" + listener.Addr().String() + "/good-url"
	err = f2.Patch(&base.Request{URL: newURL}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify URL was patched
	if f2.meta.Req.URL != newURL {
		t.Errorf("Patched URL = %v, want %v", f2.meta.Req.URL, newURL)
	}

	// Step 4: Start download and verify success
	err = f2.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = f2.Wait()
	if err != nil {
		t.Fatal(err)
	}

	// Verify file was downloaded correctly
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("Download() got = %v, want %v", got, want)
	}
}

// TestFetcher_Patch_Labels tests patching request labels with merge behavior
func TestFetcher_Patch_Labels(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	f := buildFetcher()
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: &http.OptsExtra{Connections: 1},
	}

	err := f.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		Labels: map[string]string{
			"key1": "value1",
			"key3": "value3",
		},
	}, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Verify initial labels
	if f.meta.Req.Labels["key1"] != "value1" {
		t.Errorf("Initial label key1 = %v, want value1", f.meta.Req.Labels["key1"])
	}
	if f.meta.Req.Labels["key3"] != "value3" {
		t.Errorf("Initial label key3 = %v, want value3", f.meta.Req.Labels["key3"])
	}

	// Patch with new labels - key1 should be overwritten, key2 should be added, key3 should remain
	patchReq := &base.Request{
		Labels: map[string]string{
			"key1": "modified",
			"key2": "newValue",
		},
	}
	err = f.Patch(patchReq, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify labels were merged correctly
	if f.meta.Req.Labels["key1"] != "modified" {
		t.Errorf("Patched label key1 = %v, want modified", f.meta.Req.Labels["key1"])
	}
	if f.meta.Req.Labels["key2"] != "newValue" {
		t.Errorf("Patched label key2 = %v, want newValue", f.meta.Req.Labels["key2"])
	}
	// key3 should remain unchanged
	if f.meta.Req.Labels["key3"] != "value3" {
		t.Errorf("Label key3 = %v, want value3 (should remain unchanged)", f.meta.Req.Labels["key3"])
	}
}

// TestFetcher_Patch_Extra tests patching request Extra with merge behavior
func TestFetcher_Patch_Extra(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	f := buildFetcher()
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: &http.OptsExtra{Connections: 1},
	}

	// Resolve with initial Extra
	err := f.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
		Extra: &http.ReqExtra{
			Method: "GET",
			Body:   "initial body",
			Header: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "original",
			},
		},
	}, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Verify initial Extra
	initialExtra := f.meta.Req.Extra.(*http.ReqExtra)
	if initialExtra.Method != "GET" {
		t.Errorf("Initial Method = %v, want GET", initialExtra.Method)
	}
	if initialExtra.Body != "initial body" {
		t.Errorf("Initial Body = %v, want 'initial body'", initialExtra.Body)
	}
	if initialExtra.Header["Authorization"] != "Bearer token123" {
		t.Errorf("Initial Authorization header = %v, want 'Bearer token123'", initialExtra.Header["Authorization"])
	}
	if initialExtra.Header["X-Custom"] != "original" {
		t.Errorf("Initial X-Custom header = %v, want 'original'", initialExtra.Header["X-Custom"])
	}

	// Patch with partial Extra - only update some fields
	patchReq := &base.Request{
		Extra: &http.ReqExtra{
			Method: "POST", // Update method
			// Body is empty, should NOT update
			Header: map[string]string{
				"X-Custom": "modified", // Overwrite existing
				"X-New":    "added",    // Add new
				// Authorization is not in patch, should remain
			},
		},
	}
	err = f.Patch(patchReq, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify Extra was merged correctly
	patchedExtra := f.meta.Req.Extra.(*http.ReqExtra)

	// Method should be updated
	if patchedExtra.Method != "POST" {
		t.Errorf("Patched Method = %v, want POST", patchedExtra.Method)
	}

	// Body should remain unchanged (patch had empty body)
	if patchedExtra.Body != "initial body" {
		t.Errorf("Patched Body = %v, want 'initial body' (should remain unchanged)", patchedExtra.Body)
	}

	// Authorization header should remain unchanged
	if patchedExtra.Header["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization header = %v, want 'Bearer token123' (should remain unchanged)", patchedExtra.Header["Authorization"])
	}

	// X-Custom header should be overwritten
	if patchedExtra.Header["X-Custom"] != "modified" {
		t.Errorf("X-Custom header = %v, want 'modified'", patchedExtra.Header["X-Custom"])
	}

	// X-New header should be added
	if patchedExtra.Header["X-New"] != "added" {
		t.Errorf("X-New header = %v, want 'added'", patchedExtra.Header["X-New"])
	}
}

// TestFetcher_Patch_NilData tests that Patch with nil data doesn't cause errors
func TestFetcher_Patch_NilData(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	f := buildFetcher()
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: &http.OptsExtra{Connections: 1},
	}

	err := f.Resolve(&base.Request{
		URL: "http://" + listener.Addr().String() + "/" + test.BuildName,
	}, opts)
	if err != nil {
		t.Fatal(err)
	}

	originalURL := f.meta.Req.URL

	// Patch with nil data - should not cause error
	err = f.Patch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify URL unchanged
	if f.meta.Req.URL != originalURL {
		t.Errorf("URL changed after nil patch: got %v, want %v", f.meta.Req.URL, originalURL)
	}

	// Patch with empty request - should not cause error
	err = f.Patch(&base.Request{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify URL still unchanged
	if f.meta.Req.URL != originalURL {
		t.Errorf("URL changed after empty patch: got %v, want %v", f.meta.Req.URL, originalURL)
	}
}

// TestFetcher_Patch_CookieExpired tests the Patch functionality where a download fails
// mid-way due to expired cookie, then succeeds after patching with a new valid cookie.
// This simulates:
// 1. Initial resolve with valid cookie succeeds
// 2. Download starts but fails because cookie expires mid-download (server returns 401)
// 3. User patches the task with a new valid cookie
// 4. Download resumes and completes successfully
func TestFetcher_Patch_CookieExpired(t *testing.T) {
	listener := test.StartTestCookieExpiringServer()
	defer listener.Close()

	downloadURL := "http://" + listener.Addr().String() + "/" + test.BuildName
	opts := &base.Options{
		Name:  test.DownloadName,
		Path:  test.Dir,
		Extra: &http.OptsExtra{Connections: 1},
	}

	// Step 1: Resolve with old_token - should succeed (first request accepts old_token)
	f := buildFetcher()
	err := f.Resolve(&base.Request{
		URL: downloadURL,
		Extra: &http.ReqExtra{
			Header: map[string]string{
				"Cookie": "session=old_token",
			},
		},
	}, opts)
	if err != nil {
		t.Fatalf("Resolve should succeed with old_token: %v", err)
	}

	// Verify initial cookie
	initialExtra := f.meta.Req.Extra.(*http.ReqExtra)
	if initialExtra.Header["Cookie"] != "session=old_token" {
		t.Errorf("Initial Cookie = %v, want session=old_token", initialExtra.Header["Cookie"])
	}
	// This test exercises patching credentials after an authenticated request
	// fails. Discard the already-authorized Resolve response so the stale cookie
	// is observable on the first Start.
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}

	// Step 2: Start download - should fail because old_token is now expired
	// (server only accepts old_token for first request, subsequent requests need new_token)
	err = f.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err = f.Wait()
	// Download should fail with 401 error
	if err == nil {
		t.Fatal("Expected download to fail with expired cookie, but it succeeded")
	}
	t.Logf("Download failed as expected: %v", err)

	// Step 3: Patch with new valid cookie
	err = f.Patch(&base.Request{
		Extra: &http.ReqExtra{
			Header: map[string]string{
				"Cookie": "session=new_token",
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("Patch to update cookie failed: %v", err)
	}

	// Verify cookie was updated
	patchedExtra := f.meta.Req.Extra.(*http.ReqExtra)
	if patchedExtra.Header["Cookie"] != "session=new_token" {
		t.Errorf("Cookie should be updated: got %v, want session=new_token", patchedExtra.Header["Cookie"])
	}

	// Step 4: Restart download - should succeed with new cookie
	err = f.Start()
	if err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	err = f.Wait()
	if err != nil {
		t.Fatalf("Download after patch failed: %v", err)
	}

	// Verify download completed successfully
	want := test.FileMd5(test.BuildFile)
	got := test.FileMd5(test.DownloadFile)
	if want != got {
		t.Errorf("File MD5 mismatch: got %v, want %v", got, want)
	}
}
