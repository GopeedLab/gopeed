package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestFetcher_OptionsSizeHintAndPatch(t *testing.T) {
	for _, known := range []bool{false, true} {
		for _, payload := range []string{"", "0123456789"} {
			t.Run(fmt.Sprintf("known=%v/bytes=%d", known, len(payload)), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if known {
						w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
					}
					w.(http.Flusher).Flush()
					fmt.Fprint(w, payload)
				}))
				defer server.Close()
				f := buildFetcher()
				defer f.Close()
				f.Meta().Opts = &base.Options{Size: 7, Path: t.TempDir(), Name: "output.bin"}
				if err := f.Resolve(&base.Request{URL: server.URL}); err != nil {
					t.Fatal(err)
				}
				want := int64(7)
				if known {
					want = int64(len(payload))
				}
				if f.meta.Res.Size != want || f.hasKnownSize() != known {
					t.Fatalf("resolved size=%d known=%v", f.meta.Res.Size, f.hasKnownSize())
				}
				if err := f.Patch(nil, &base.Options{Size: 9}); err != nil {
					t.Fatal(err)
				}
				if !known {
					want = 9
				}
				if f.meta.Res.Size != want {
					t.Fatal("hint update overrode a real size or failed to update an estimate")
				}
				if err := f.Start(); err != nil {
					t.Fatal(err)
				}
				if err := f.Wait(); err != nil {
					t.Fatal(err)
				}
				if f.meta.Res.Size != int64(len(payload)) || !f.hasKnownSize() {
					t.Fatalf("final size=%d known=%v", f.meta.Res.Size, f.hasKnownSize())
				}
				snapshot, err := new(FetcherManager).Store(f)
				if err != nil {
					t.Fatal(err)
				}
				fd := snapshot.(*fetcherData)
				if fd.SizeKnown == nil || !*fd.SizeKnown {
					t.Fatal("snapshot lost confirmed size authority")
				}
				_, restore := new(FetcherManager).Restore()
				restored := restore(f.meta, fd).(*Fetcher)
				if !restored.hasKnownSize() || restored.knownSize() != int64(len(payload)) {
					t.Fatal("restored size lost authority, including known zero")
				}
			})
		}
	}
}

func TestFetcher_SizeHintCannotCompleteForbiddenUnknownRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.(http.Flusher).Flush()
		fmt.Fprint(w, "0123456789")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	f := buildFetcher()
	defer f.Close()
	f.Meta().Opts = &base.Options{Size: 3, Path: t.TempDir(), Name: "output.bin"}
	if err := f.Resolve(&base.Request{URL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err == nil {
		t.Fatal("a failed unknown-length Range request was completed from an estimate")
	}
}

func TestFetcher_SizeHintPreservesHTTPIntegrity(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, broken := range []bool{false, true} {
			t.Run(fmt.Sprintf("chunked=%v/broken=%v", chunked, broken), func(t *testing.T) {
				payload := []byte("0123456789")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if chunked && broken {
						conn, rw, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						defer conn.Close()
						fmt.Fprint(rw, "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\na\r\n0123456789\r\n")
						rw.Flush() // Omit the final chunk to simulate a connection interruption.
						return
					}
					if !chunked {
						length := len(payload)
						if broken {
							length++
						}
						w.Header().Set("Content-Length", fmt.Sprint(length))
					}
					w.(http.Flusher).Flush()
					w.Write(payload)
				}))
				defer server.Close()
				f := buildFetcher()
				defer f.Close()
				f.Meta().Opts = &base.Options{Size: 3, Path: t.TempDir(), Name: "output.bin"}
				if err := f.Resolve(&base.Request{URL: server.URL}); err != nil {
					t.Fatal(err)
				}
				if !chunked {
					length := int64(len(payload))
					if broken {
						length++
					}
					if f.meta.Res.Size != length || f.sequentialSizeUnknown {
						t.Fatal("HTTP length did not override size hint")
					}
				} else if f.meta.Res.Size != 3 || !f.sequentialSizeUnknown {
					t.Fatal("unknown HTTP length did not retain the display hint")
				}
				if err := f.Start(); err != nil {
					t.Fatal(err)
				}
				err := f.Wait()
				if broken {
					if err == nil {
						t.Fatal("truncated HTTP response succeeded after exceeding the size hint")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if f.meta.Res.Size != int64(len(payload)) || f.sequentialSizeUnknown {
					t.Fatalf("completed size=%d unknown=%v", f.meta.Res.Size, f.sequentialSizeUnknown)
				}
			})
		}
	}
}

func TestFetcher_SizeHintSurvivesRestore(t *testing.T) {
	for _, partial := range []bool{false, true} {
		for _, fixedLength := range []bool{false, true} {
			t.Run(fmt.Sprintf("partial=%v/fixedLength=%v", partial, fixedLength), func(t *testing.T) {
				payload := []byte("0123456789")
				var requests atomic.Int64
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					request := requests.Add(1)
					if got := r.Header.Get("Range"); got != "" {
						t.Errorf("estimated sizes must not be used for resume ranges: %s", got)
					}
					if fixedLength && request > 1 {
						w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
					}
					w.(http.Flusher).Flush()
					if fixedLength && request == 2 {
						w.Write(payload[:4]) // Retry must discard this interrupted attempt.
						return
					}
					w.Write(payload)
				}))
				defer server.Close()
				f := buildFetcher()
				opts := &base.Options{Size: 3, Path: t.TempDir(), Name: "output.bin"}
				f.Meta().Opts = opts
				if err := f.Resolve(&base.Request{URL: server.URL}); err != nil {
					t.Fatal(err)
				}
				fm := new(FetcherManager)
				snapshot, err := fm.Store(f)
				if err != nil {
					t.Fatal(err)
				}
				data, err := json.Marshal(snapshot)
				if err != nil {
					t.Fatal(err)
				}
				var saved fetcherData
				if err := json.Unmarshal(data, &saved); err != nil {
					t.Fatal(err)
				}
				if !saved.SequentialSizeUnknown || saved.SizeKnown == nil || *saved.SizeKnown || *saved.ResourceSize != 3 {
					t.Fatal("snapshot lost the size hint or its advisory state")
				}
				f.Close()
				// An interrupted earlier attempt left a tail longer than the new output.
				if err := os.WriteFile(filepath.Join(opts.Path, opts.Name), bytes.Repeat([]byte("old"), 100), 0600); err != nil {
					t.Fatal(err)
				}
				if partial {
					chunk := newChunk(0, 0)
					chunk.Downloaded = 7 // Already exceeds the display estimate.
					saved.Connections = []*connection{{ID: 0, Role: rolePrimary, State: connNotStarted, Chunk: chunk, Downloaded: 7}}
				}
				_, restore := fm.Restore()
				restored := restore(f.meta, &saved).(*Fetcher)
				restored.Setup(f.ctl)
				defer restored.Close()
				if err := restored.Start(); err != nil {
					t.Fatal(err)
				}
				if err := restored.Wait(); err != nil {
					t.Fatal(err)
				}
				body, err := os.ReadFile(restored.meta.SingleFilepath())
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(body, payload) || restored.meta.Res.Size != int64(len(payload)) || restored.Progress().TotalDownloaded() != int64(len(payload)) {
					t.Fatalf("restored stream retained an estimate or stale bytes: body=%q size=%d progress=%d", body, restored.meta.Res.Size, restored.Progress().TotalDownloaded())
				}
			})
		}
	}
}
