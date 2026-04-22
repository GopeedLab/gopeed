//go:build !cgo || !webview

package goprovider

import (
	"testing"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func TestStubProviderIsUnavailable(t *testing.T) {
	provider := New()
	if provider.IsAvailable() {
		t.Fatal("stub provider should not be available")
	}
	_, err := provider.Open(enginewebview.OpenOptions{})
	if err != enginewebview.ErrUnavailable {
		t.Fatalf("expected ErrUnavailable, got: %v", err)
	}
}
