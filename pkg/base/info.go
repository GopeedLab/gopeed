package base

// Version is the build version, set at build time, using `go build -ldflags "-X github.com/GopeedLab/gopeed/pkg/base.Version=1.0.0"`.
var Version string
var InDocker string

func init() {
	if Version == "" {
		Version = "dev"
	}
}
