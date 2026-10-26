package main

import (
	"fmt"
	"os"

	"github.com/GopeedLab/gopeed/cmd"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

// only for local development
func main() {
	cfg := &model.StartConfig{
		Network:   "tcp",
		Address:   "127.0.0.1:9999",
		Storage:   model.StorageBolt,
		WebEnable: true,
	}
	if err := cmd.Start(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
