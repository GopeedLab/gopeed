//go:build !moejs

package main

import (
	"fmt"
	"github.com/dop251/goja"
	"os"
)

func main() {
	source := "1+1"
	if len(os.Args) > 1 {
		source = os.Args[1]
	}
	rt := goja.New()
	v, e := rt.RunString(source)
	if e != nil {
		panic(e)
	}
	fmt.Println(v.Export())
}
