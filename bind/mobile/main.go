package libgopeed

// #cgo LDFLAGS: -static-libstdc++
import "C"
import (
	"github.com/GopeedLab/gopeed/bind"
	"github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/api/model"
)

func Init(cfg *model.StartConfig) error {
	return bind.Init(cfg)
}

func Invoke(request *api.Request) string {
	return bind.Invoke(request)
}
