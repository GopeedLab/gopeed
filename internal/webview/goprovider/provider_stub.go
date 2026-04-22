//go:build !cgo || !webview

package goprovider

import enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"

func New() enginewebview.Provider {
	return enginewebview.NewUnavailableProvider()
}
