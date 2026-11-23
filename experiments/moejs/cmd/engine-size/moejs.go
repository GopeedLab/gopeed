//go:build moejs

package main

import (
	"fmt"
	"github.com/MoeForge/moejs"
	"os"
)

func main() {
	source := "1+1"
	if len(os.Args) > 1 {
		source = os.Args[1]
	}
	rt := moejs.NewRuntime(moejs.Options{MutableIntrinsics: true})
	script, e := moejs.CompileScript("size", source)
	if e != nil {
		panic(e)
	}
	v, e := rt.RunScript(script)
	if e != nil {
		panic(e)
	}
	out, e := rt.ToGo(v)
	if e != nil {
		panic(e)
	}
	fmt.Println(out)
}
