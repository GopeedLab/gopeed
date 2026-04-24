//go:build !darwin

package main

import (
	"github.com/GopeedLab/gopeed/internal/webview/goprovider"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func applyWebViewProvider(config *model.StartConfig) {
	if config == nil {
		return
	}
	config.WebViewProvider = goprovider.New()
}
