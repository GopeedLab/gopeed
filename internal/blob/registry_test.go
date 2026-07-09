package blob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestRegistryBlobHTTPGetAndRange(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateBlob([]byte("hello world"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if !registry.IsURL(url) {
		t.Fatal("expected registry to recognize created url")
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected GET status: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Length"); got != "11" {
		t.Fatalf("unexpected content length: %q", got)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("unexpected accept ranges: %q", got)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("unexpected content type: %q", got)
	}
	if string(body) != "hello world" {
		t.Fatalf("unexpected GET body: %q", string(body))
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=6-10")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("unexpected range status: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Range"); got != "bytes 6-10/11" {
		t.Fatalf("unexpected content range: %q", got)
	}
	if string(body) != "world" {
		t.Fatalf("unexpected range body: %q", string(body))
	}
}

func TestRegistryOpenerSizeWithoutRange(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	payload := []byte("hello opener")
	url, err := registry.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		if req.Offset != 0 || req.End != -1 {
			t.Fatalf("unexpected non-range request: %#v", req)
		}
		return io.NopCloser(bytes.NewReader(payload)), nil
	}, &CreateOptions{
		ContentType: "text/plain",
		Size:        int64(len(payload)),
		Range:       false,
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=6-11")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Length"); got != "12" {
		t.Fatalf("unexpected content length: %q", got)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "" {
		t.Fatalf("unexpected accept ranges: %q", got)
	}
	if string(body) != string(payload) {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestRegistryOpenerRangeConcurrentRequests(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	payload := []byte("abcdefghijklmnopqrstuvwxyz")
	var mu sync.Mutex
	var calls []OpenRequest
	url, err := registry.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		mu.Lock()
		calls = append(calls, req)
		mu.Unlock()

		start := int(req.Offset)
		end := len(payload)
		if req.End >= 0 {
			end = int(req.End) + 1
		}
		if start < 0 || start > end || end > len(payload) {
			return nil, errors.New("bad test range")
		}
		return io.NopCloser(bytes.NewReader(payload[start:end])), nil
	}, &CreateOptions{
		Size:  int64(len(payload)),
		Range: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ranges := []struct {
		header string
		want   string
	}{
		{header: "bytes=0-4", want: "abcde"},
		{header: "bytes=10-15", want: "klmnop"},
		{header: "bytes=20-25", want: "uvwxyz"},
	}
	var wg sync.WaitGroup
	for _, item := range ranges {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Set("Range", item.header)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				t.Error(err)
				return
			}
			if resp.StatusCode != http.StatusPartialContent {
				t.Errorf("unexpected status for %s: %d", item.header, resp.StatusCode)
			}
			if string(body) != item.want {
				t.Errorf("unexpected body for %s: %q", item.header, string(body))
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != len(ranges) {
		t.Fatalf("unexpected opener call count: %d", len(calls))
	}
	seen := map[OpenRequest]bool{}
	for _, call := range calls {
		seen[call] = true
	}
	for _, want := range []OpenRequest{{Offset: 0, End: 4}, {Offset: 10, End: 15}, {Offset: 20, End: 25}} {
		if !seen[want] {
			t.Fatalf("missing opener request %#v, got %#v", want, calls)
		}
	}
}

func TestRegistryRejectsRangeWithoutSize(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	_, err := registry.CreateOpener(func(ctx context.Context, req OpenRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("")), nil
	}, &CreateOptions{Range: true})
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("expected invalid options error, got %v", err)
	}
}

func TestRegistryHTTPErrorStatuses(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	defer registry.Close()

	url, err := registry.CreateBlob([]byte("secret"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	badURL := url + "-missing"
	resp, err := http.Get(badURL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected missing source status: %d", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected method status: %d", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=99-100")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("unexpected invalid range status: %d", resp.StatusCode)
	}

	if err := registry.Revoke(url); err != nil {
		t.Fatal(err)
	}
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected revoked source status: %d", resp.StatusCode)
	}
	if registry.IsURL(url) {
		t.Fatal("revoked source should not be usable")
	}
}
