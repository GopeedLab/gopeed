//go:build darwin

package main

import (
	"github.com/GopeedLab/gopeed/internal/webview/rpcprovider"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func applyWebViewProvider(config *model.StartConfig) {
	if config == nil {
		return
	}
	config.WebViewProvider = newDesktopWebViewProvider(config.WebViewRPCConfig)
}

func newDesktopWebViewProvider(rpcCfg *enginewebview.RPCConfig) enginewebview.Provider {
	if rpcCfg != nil && rpcCfg.Enabled() {
		return rpcprovider.New(*rpcCfg)
	}
	return enginewebview.NewUnavailableProvider()
}
