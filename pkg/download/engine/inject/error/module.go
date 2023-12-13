package error

import (
	"github.com/dop251/goja"
)

type MessageError struct {
	Message string `json:"message"`
}

func (e *MessageError) Error() string {
	return e.Message
}

func Enable(runtime *goja.Runtime) error {
	messageError := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		var message string
		if len(call.Arguments) > 0 {
			message = call.Arguments[0].String()
		}
		instance := &MessageError{
			Message: message,
		}
		instanceValue := runtime.ToValue(instance).(*goja.Object)
		instanceValue.SetPrototype(call.This.Prototype())
		return instanceValue
	})
	return runtime.Set("MessageError", messageError)
}
