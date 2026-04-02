package gblob

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	internalfetcher "github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestFetcherClosedSourceBeforeExpectedSizeReturnsError(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	url, err := registry.CreateWritableStream(&CreateWritableStreamOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Write(url, []byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := registry.CloseSource(url); err != nil {
		t.Fatal(err)
	}

	downloadDir := t.TempDir()
	filePath := filepath.Join(downloadDir, "partial.bin")
	f := &Fetcher{
		registry: registry,
		meta: &internalfetcher.FetcherMeta{
			Req: &base.Request{URL: url},
			Res: &base.Resource{
				Size:  10,
				Range: true,
				Files: []*base.FileInfo{{Name: "partial.bin", Req: &base.Request{URL: url}}},
			},
			Opts: &base.Options{Path: downloadDir, Name: "partial.bin"},
		},
	}
	f.Setup(controller.NewController())

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-f.doneCh:
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("expected unexpected EOF, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for fetcher completion")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "abc" {
		t.Fatalf("unexpected partial file content: %q", string(data))
	}
}

func TestFetcherClosedUnknownSizeSourceCompletesSuccessfully(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	url, err := registry.CreateWritableStream(&CreateWritableStreamOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Write(url, []byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	if err := registry.CloseSource(url); err != nil {
		t.Fatal(err)
	}

	downloadDir := t.TempDir()
	filePath := filepath.Join(downloadDir, "stream.bin")
	f := &Fetcher{
		registry: registry,
		meta: &internalfetcher.FetcherMeta{
			Req: &base.Request{URL: url},
			Res: &base.Resource{
				Size:  0,
				Range: false,
				Files: []*base.FileInfo{{Name: "stream.bin", Req: &base.Request{URL: url}}},
			},
			Opts: &base.Options{Path: downloadDir, Name: "stream.bin"},
		},
	}
	f.Setup(controller.NewController())

	if err := f.Start(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-f.doneCh:
		if err != nil {
			t.Fatalf("expected nil completion error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for fetcher completion")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "abcdef" {
		t.Fatalf("unexpected stream file content: %q", string(data))
	}
	if got := f.meta.Res.Size; got != int64(len(data)) {
		t.Fatalf("unexpected resolved size: got %d want %d", got, len(data))
	}
	if got := f.meta.Res.Files[0].Size; got != int64(len(data)) {
		t.Fatalf("unexpected resolved file size: got %d want %d", got, len(data))
	}
}
