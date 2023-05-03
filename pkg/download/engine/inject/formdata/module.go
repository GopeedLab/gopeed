package formdata

import "github.com/dop251/goja"

type FormData struct {
	data map[string]any
}

func (fd *FormData) Append(name string, value any) {
	fd.data[name] = value
}

func (fd *FormData) Delete(name string) {
	delete(fd.data, name)
}

func (fd *FormData) Entries() []any {
	var entries []any
	for k, v := range fd.data {
		entries = append(entries, []any{k, v})
	}
	return entries
}

func (fd *FormData) Get(name string) any {
	return fd.data[name]
}

func (fd *FormData) GetAll(name string) []any {
	return []any{fd.data[name]}
}

func (fd *FormData) Has(name string) bool {
	_, ok := fd.data[name]
	return ok
}

func (fd *FormData) Keys() []string {
	var keys []string
	for k := range fd.data {
		keys = append(keys, k)
	}
	return keys
}

func (fd *FormData) Set(name string, value any) {
	fd.data[name] = value
}

func (fd *FormData) Values() []any {
	var values []any
	for _, v := range fd.data {
		values = append(values, v)
	}
	return values
}

func Enable(runtime *goja.Runtime) error {
	file := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
		instance := &FormData{
			data: make(map[string]any),
		}
		instanceValue := runtime.ToValue(instance).(*goja.Object)
		instanceValue.SetPrototype(call.This.Prototype())
		return instanceValue
	})
	return runtime.Set("FormData", file)
}
