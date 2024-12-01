package libgopeed

// #cgo LDFLAGS: -static-libstdc++
import "C"
import (
	"github.com/GopeedLab/gopeed/bind"
	"github.com/GopeedLab/gopeed/pkg/api"
	"github.com/GopeedLab/gopeed/pkg/api/model"
)

func Create(cfg *model.StartConfig) string {
	return bind.Create(cfg)
}

func Invoke(index int, request *api.Request) string {
	return bind.Invoke(index, request)
}
