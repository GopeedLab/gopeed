package download

import (
	"runtime"

	"github.com/GopeedLab/gopeed/pkg/base"
)

type InstanceHost struct {
	Env HostEnv `json:"env"`
}

type HostEnv struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func newInstanceHost() *InstanceHost {
	return &InstanceHost{Env: HostEnv{
		Version: base.Version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}}
}
