package http

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	gohttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func TestFetcher_ChecksumVerification(t *testing.T) {
	content := []byte("Hello, Gopeed Checksum Verification Test Data!")
	md5Hasher := md5.New()
	md5Hasher.Write(content)
	correctMd5 := hex.EncodeToString(md5Hasher.Sum(nil))

	sha256Hasher := sha256.New()
	sha256Hasher.Write(content)
	correctSha256 := hex.EncodeToString(sha256Hasher.Sum(nil))

	wrongHash := "0123456789abcdef0123456789abcdef"

	server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
		w.Header().Set(base.HttpHeaderContentLength, fmt.Sprintf("%d", len(content)))
		w.Header().Set(base.HttpHeaderAcceptRanges, base.HttpHeaderBytes)
		w.WriteHeader(gohttp.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	t.Run("Valid MD5", func(t *testing.T) {
		f := buildFetcher()
		defer f.Close()

		opts := &base.Options{
			Path: t.TempDir(),
			Name: "test_valid_md5.txt",
			Extra: &http.OptsExtra{
				Connections: 1,
			},
			Checksum: &base.ChecksumOption{
				Algorithm: "md5",
				Expected:  correctMd5,
			},
		}

		if err := f.Resolve(&base.Request{URL: server.URL + "/test.txt"}, opts); err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if err := f.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		if err := f.Wait(); err != nil {
			t.Fatalf("Wait error (expected success): %v", err)
		}
	})

	t.Run("Valid SHA256", func(t *testing.T) {
		f := buildFetcher()
		defer f.Close()

		opts := &base.Options{
			Path: t.TempDir(),
			Name: "test_valid_sha256.txt",
			Extra: &http.OptsExtra{
				Connections: 1,
			},
			Checksum: &base.ChecksumOption{
				Algorithm: "sha256",
				Expected:  correctSha256,
			},
		}

		if err := f.Resolve(&base.Request{URL: server.URL + "/test.txt"}, opts); err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if err := f.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		if err := f.Wait(); err != nil {
			t.Fatalf("Wait error (expected success): %v", err)
		}
	})

	t.Run("Wrong MD5 mismatch", func(t *testing.T) {
		f := buildFetcher()
		defer f.Close()

		opts := &base.Options{
			Path: t.TempDir(),
			Name: "test_wrong_md5.txt",
			Extra: &http.OptsExtra{
				Connections: 1,
			},
			Checksum: &base.ChecksumOption{
				Algorithm: "md5",
				Expected:  wrongHash,
			},
		}

		if err := f.Resolve(&base.Request{URL: server.URL + "/test.txt"}, opts); err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if err := f.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		err := f.Wait()
		if err == nil {
			t.Fatal("Wait expected checksum mismatch error, got nil")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatalf("expected 'checksum mismatch' in error, got %v", err)
		}
	})

	t.Run("Unsupported Algorithm", func(t *testing.T) {
		f := buildFetcher()
		defer f.Close()

		opts := &base.Options{
			Path: t.TempDir(),
			Name: "test_unsupported.txt",
			Extra: &http.OptsExtra{
				Connections: 1,
			},
			Checksum: &base.ChecksumOption{
				Algorithm: "invalid-algo",
				Expected:  wrongHash,
			},
		}

		if err := f.Resolve(&base.Request{URL: server.URL + "/test.txt"}, opts); err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if err := f.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		err := f.Wait()
		if err == nil {
			t.Fatal("Wait expected error for unsupported algorithm, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported checksum algorithm") {
			t.Fatalf("expected 'unsupported checksum algorithm' in error, got %v", err)
		}
	})

	t.Run("No Checksum (Opt-in backward compatibility)", func(t *testing.T) {
		f := buildFetcher()
		defer f.Close()

		opts := &base.Options{
			Path: t.TempDir(),
			Name: test.DownloadName,
			Extra: &http.OptsExtra{
				Connections: 1,
			},
		}

		if err := f.Resolve(&base.Request{URL: server.URL + "/test.txt"}, opts); err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if err := f.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		if err := f.Wait(); err != nil {
			t.Fatalf("Wait error: %v", err)
		}
	})
}
