// go build -ldflags="-w -s" -buildmode=c-shared -o bin/libgopeed.dll github.com/monkeyWie/gopeed/bind/desktop
// go build -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/macos//libgopeed.dylib github.com/monkeyWie/gopeed/bind/desktop
package main

import "C"
import (
	"encoding/json"
	"github.com/monkeyWie/gopeed/pkg/rest"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
)

func main() {}

//export Start
func Start(cfg *C.char) (int, *C.char) {
	var config model.StartConfig
	if err := json.Unmarshal([]byte(C.GoString(cfg)), &config); err != nil {
		return 0, C.CString(err.Error())
	}
	realPort, err := rest.Start(&config)
	if err != nil {
		return 0, C.CString(err.Error())
	}
	return realPort, nil
}

//export Stop
func Stop() {
	rest.Stop()
}
