package xhr

import (
	_ "embed"

	"github.com/dop251/goja"
)

//go:embed xhr.js
var script string

func Enable(runtime *goja.Runtime) error {
	_, err := runtime.RunString(script)
	return err
}
