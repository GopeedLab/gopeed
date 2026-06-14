//go:build webview && darwin

package rpcprovider

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/webview/integrationtest"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

var (
	webViewRPCNetwork = flag.String("webview-rpc-network", "unix", "WebView RPC network for integration tests")
	webViewRPCAddress = flag.String("webview-rpc-address", defaultWebViewRPCAddress(), "WebView RPC address for integration tests")
	webViewRPCToken   = flag.String("webview-rpc-token", "", "WebView RPC token for integration tests")
)

func TestProviderContract(t *testing.T) {
	provider := New(enginewebview.RPCConfig{
		Network: *webViewRPCNetwork,
		Address: *webViewRPCAddress,
		Token:   *webViewRPCToken,
	})
	integrationtest.RunProviderContract(t, provider, integrationtest.ContractOptions{
		AvailabilityTimeout: 5 * time.Minute,
		CookieDomainMode:    integrationtest.CookieDomainModeRequired,
		CookieTestURL:       "https://example.com/",
	})
}

func defaultWebViewRPCAddress() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "Library", "Application Support", "com.gopeed.gopeed", "gopeed_webview.sock")
}
