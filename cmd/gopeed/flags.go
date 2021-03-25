package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type args struct {
	url         string
	connections *int
	dir         *string
}

func parse() *args {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	var args args
	args.connections = flag.Int("C", 16, "Concurrent connections.")
	args.dir = flag.String("D", dir, "Save directory.")
	flag.Parse()
	t := flag.Args()
	if len(t) > 0 {
		args.url = t[0]
	} else {
		gPrintln("missing url parameter")
		gPrintln("try 'gopeed -h' for more information")
		os.Exit(1)
	}
	return &args
}

func gPrint(msg string) {
	fmt.Print("gopeed: " + msg)
}

func gPrintln(msg string) {
	gPrint(msg + "\n")
}
