//go:build cgo && webview && !darwin

package goprovider

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/webview/integrationtest"
)

func TestProviderContract(t *testing.T) {
	warmUpWebView(t)
	integrationtest.RunProviderContract(t, New(), integrationtest.ContractOptions{
		CookieDomainMode: integrationtest.CookieDomainModeRequired,
	})
}
