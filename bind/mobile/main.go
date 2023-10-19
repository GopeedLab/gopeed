package libgopeed

// #cgo LDFLAGS: -static-libstdc++
import "C"
import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/pkg/rest"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func Start(cfg string) (int, error) {
	var config model.StartConfig
	if err := json.Unmarshal([]byte(cfg), &config); err != nil {
		return 0, err
	}
	config.ProductionMode = true
	return rest.Start(&config)
}

func Stop() {
	rest.Stop()
}
