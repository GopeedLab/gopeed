//go:build cgo && darwin && webview_native

package goprovider

import (
	"github.com/GopeedLab/gopeed/internal/webview/integrationtest"
	"os"
	"testing"
)

func TestMain(m *testing.M)           { os.Exit(RunMainThreadLoop(m.Run)) }
func TestProviderEvents(t *testing.T) { integrationtest.RunEventContract(t, New()) }
