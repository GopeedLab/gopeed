package http

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func TestResolveResponseSize(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		status               int
		length, contentRange string
		size                 int64
		invalid              bool
	}{
		{"full", 200, "801571015", "", 801571015, false},
		{"partial_prefix", 206, "472321339", "bytes 0-472321338/801571015", 801571015, false},
		{"partial_nonzero", 206, "100000", "bytes 100000-199999/801571015", 801571015, false},
		{"second_report", 206, "335134409", "bytes 0-335134408/552140234", 552140234, false},
		{"length_mismatch", 206, "10", "bytes 0-19/100", 0, true},
		{"unknown_total", 206, "472321339", "bytes 0-472321338/*", 0, true},
		{"missing_range", 206, "10", "", 0, true},
		{"malformed_range", 206, "10", "bytes invalid", 0, true},
		{"invalid_bounds", 206, "10", "bytes 20-10/100", 0, true},
		{"invalid_total", 206, "10", "bytes 0-9/9", 0, true},
		{"overflow", 206, "10", "bytes 0-9/9223372036854775808", 0, true},
		{"chunked_partial", 206, "", "bytes 0-9/100", 100, false},
		{"unknown_full", 200, "", "", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				if tc.contentRange != "" {
					w.Header().Set("Content-Range", tc.contentRange)
				}
				w.WriteHeader(tc.status)
				w.(http.Flusher).Flush()
			}))
			defer server.Close()
			f := buildFetcher()
			defer f.Close()
			err := f.Resolve(&base.Request{URL: server.URL + "/file"}, &base.Options{Path: t.TempDir()})
			if tc.invalid {
				if !errors.Is(err, errInvalidRangeResponse) {
					t.Fatalf("Resolve error = %v, want invalid range", err)
				}
				if f.meta.Res != nil || f.resolveResp != nil || f.getState() != stateError {
					t.Fatal("invalid response was retained or published")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if f.meta.Res.Size != tc.size || f.meta.Res.Files[0].Size != tc.size {
				t.Fatalf("resource/file size = %d/%d, want %d", f.meta.Res.Size, f.meta.Res.Files[0].Size, tc.size)
			}
			if tc.status == 206 && !f.meta.Res.Range {
				t.Fatal("partial response must enable Range downloads")
			}
		})
	}
}

func TestDownloadPartialResolve(t *testing.T) {
	payload := make([]byte, 256*1024)
	for i := range payload {
		payload[i] = byte(i*31 + i/251)
	}
	for _, start := range []int{0, 100000} {
		for _, connections := range []int{1, 4} {
			t.Run(fmt.Sprintf("start_%d_connections_%d", start, connections), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					begin, end := start, start+9999
					if h := r.Header.Get("Range"); h != "" {
						if _, err := fmt.Sscanf(h, "bytes=%d-%d", &begin, &end); err != nil {
							t.Errorf("invalid request range %q: %v", h, err)
							w.WriteHeader(400)
							return
						}
					}
					w.Header().Set("Content-Length", fmt.Sprint(end-begin+1))
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", begin, end, len(payload)))
					w.WriteHeader(206)
					w.Write(payload[begin : end+1])
				}))
				defer server.Close()
				f := buildFetcher()
				defer f.Close()
				if err := f.Resolve(&base.Request{URL: server.URL + "/file"}, &base.Options{Path: t.TempDir(), Name: "file", Extra: &fhttp.OptsExtra{Connections: connections}}); err != nil {
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
				if !bytes.Equal(got, payload) {
					t.Fatalf("download content mismatch: got %d bytes, want %d", len(got), len(payload))
				}
			})
		}
	}
}
