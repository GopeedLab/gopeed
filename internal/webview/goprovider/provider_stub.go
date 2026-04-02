//go:build !cgo

package goprovider

import enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"

func New() enginewebview.Provider {
	return enginewebview.NewUnavailableProvider()
}
