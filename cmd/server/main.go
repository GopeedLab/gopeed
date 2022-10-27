package main

import (
	"github.com/monkeyWie/gopeed/pkg/rest"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
)

func main() {
	if err := rest.StartSync(&model.StartConfig{
		Network: "tcp",
		Address: "127.0.0.1:9999",
		Storage: model.StorageMem,
	}); err != nil {
		panic(err)
	}
}
