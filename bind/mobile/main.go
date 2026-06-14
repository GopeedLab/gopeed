package libgopeed

// #cgo LDFLAGS: -static-libstdc++
import "C"
import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/internal/webview/rpcprovider"
	"github.com/GopeedLab/gopeed/pkg/rest"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func Start(cfg string) (int, error) {
	var config model.StartConfig
	if err := json.Unmarshal([]byte(cfg), &config); err != nil {
		return 0, err
	}
	config.ProductionMode = true
	applyWebViewProvider(&config)
	return rest.Start(&config)
}

func Stop() {
	rest.Stop()
}

func applyWebViewProvider(config *model.StartConfig) {
	if config == nil || config.WebViewRPCConfig == nil || !config.WebViewRPCConfig.Enabled() {
		return
	}
	config.WebViewProvider = rpcprovider.New(*config.WebViewRPCConfig)
}
