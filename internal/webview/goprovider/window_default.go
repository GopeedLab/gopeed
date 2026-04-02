//go:build cgo && !darwin

package goprovider

import (
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	webview "github.com/GopeedLab/webview_go"
)

func applyWindowOptions(_ webview.WebView, _ enginewebview.OpenOptions) {}
