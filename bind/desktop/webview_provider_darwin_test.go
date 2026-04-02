//go:build darwin

package main

import (
	"testing"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func TestNewDesktopWebViewProviderRequiresRPCConfigOnDarwin(t *testing.T) {
	provider := newDesktopWebViewProvider(nil)
	if provider == nil {
		t.Fatal("expected provider")
	}
	if provider.IsAvailable() {
		t.Fatal("expected unavailable provider when darwin rpc config is missing")
	}
}

func TestNewDesktopWebViewProviderUsesRPCOnDarwinWhenConfigured(t *testing.T) {
	provider := newDesktopWebViewProvider(&enginewebview.RPCConfig{
		Network: "tcp",
		Address: "127.0.0.1:1",
	})
	if provider == nil {
		t.Fatal("expected provider")
	}
	if provider.IsAvailable() {
		t.Fatal("expected rpc provider without live host to report unavailable")
	}
	if _, err := provider.Open(enginewebview.OpenOptions{}); err != enginewebview.ErrUnavailable {
		t.Fatalf("expected unavailable error from rpc provider without host, got %v", err)
	}
}
