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
	config.NativeMode = true
	applyWebViewProvider(&config)
	return rest.Start(&config)
}

func Stop() {
	rest.Stop()
}

func GetAPIServerState() string {
	return apiServerResult(rest.GetAPIServerState())
}

func StartAPIServer() string {
	return apiServerResult(rest.StartAPIServer())
}

func StopAPIServer() string {
	return apiServerResult(rest.StopAPIServer())
}

func RestartAPIServer() string {
	return apiServerResult(rest.RestartAPIServer())
}

func apiServerResult(state *model.APIServerState, err error) string {
	result := &model.APIServerOperationResult{State: state}
	if err != nil {
		result.Error = err.Error()
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func Invoke(method, path, query, body string) string {
	return rest.Invoke(method, path, query, body)
}

type TaskEventListener interface {
	OnTaskEvent(payload string)
}

func SubscribeTaskEvents(mask int64, listener TaskEventListener) {
	if mask == 0 || listener == nil {
		rest.SubscribeTaskEvents(0, nil)
		return
	}
	rest.SubscribeTaskEvents(uint64(mask), listener.OnTaskEvent)
}

func applyWebViewProvider(config *model.StartConfig) {
	if config == nil || config.WebViewRPCConfig == nil || !config.WebViewRPCConfig.Enabled() {
		return
	}
	config.WebViewProvider = rpcprovider.New(*config.WebViewRPCConfig)
}
