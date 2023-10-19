package file

import (
	"errors"
	"github.com/dop251/goja"
	"io"
)

type File struct {
	io.Reader `json:""`
	io.Closer `json:""`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
}

func NewJsFile(runtime *goja.Runtime) (goja.Value, error) {
	fileCtor, ok := goja.AssertConstructor(runtime.Get("File"))
	if !ok {
		return nil, errors.New("file is not defined")
	}
	return fileCtor(nil)
}

func Enable(runtime *goja.Runtime) error {
	file := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		instance := &File{}
		instanceValue := runtime.ToValue(instance).(*goja.Object)
		instanceValue.SetPrototype(call.This.Prototype())
		return instanceValue
	})
	return runtime.Set("File", file)
}
