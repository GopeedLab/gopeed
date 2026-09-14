//go:build cgo && webview && !darwin

package goprovider

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/webview/integrationtest"
)

func TestProviderContract(t *testing.T) {
	integrationtest.RunProviderContract(t, New(), integrationtest.ContractOptions{
		CookieDomainMode: integrationtest.CookieDomainModeRequired,
	})
}

func TestProviderProfiles(t *testing.T) { integrationtest.RunProfileContract(t, New()) }
