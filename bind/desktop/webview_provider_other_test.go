//go:build !darwin

package main

import (
	"testing"

	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func TestApplyWebViewProviderUsesLocalProviderOutsideDarwin(t *testing.T) {
	cfg := &model.StartConfig{}
	applyWebViewProvider(cfg)
	if cfg.WebViewProvider == nil {
		t.Fatal("expected local provider to be injected")
	}
}
