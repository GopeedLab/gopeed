package http

import (
	"encoding/json"
	"fmt"
	"net"
	gohttp "net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
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
	listener := test.StartTestLimitServer(4, 0)
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

		stats := fetcher.Stats().(*http.Stats)
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
	stats := fetcher.Stats().(*http.Stats)
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
	downloadNormal(listener, 4, t)
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
	stats := fetcher.Stats().(*http.Stats)
	// With slow-start strategy, connection count may be less than max if download is fast
	// Just verify we have at least 1 connection and no more than max
	if len(stats.Connections) < 1 || len(stats.Connections) > 16 {
		t.Errorf("Stats() connection count got = %v, want between 1 and 16", len(stats.Connections))
	}
	totalDownloaded := int64(0)
	for i, conn := range stats.Connections {
		t.Logf("Connection %d: Downloaded=%d, Completed=%v", i, conn.Downloaded, conn.Completed)
		totalDownloaded += conn.Downloaded
	}
	if totalDownloaded != test.BuildSize {
		t.Errorf("Stats() got = %v, want %v", totalDownloaded, test.BuildSize)
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
			listener := test.StartTestSlowStartServer(100 * time.Nanosecond)

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
			stats := fetcher.Stats().(*http.Stats)
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
		listener := test.StartTestSlowStartServer(100 * time.Nanosecond)
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
		stats := fetcher.Stats().(*http.Stats)
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
