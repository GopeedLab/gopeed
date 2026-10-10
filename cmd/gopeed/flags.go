package main

import (
	"flag"
	"fmt"
	"os"
)

type args struct {
	url          string
	connections  *int
	dir          *string
	checksum     *string
	checksumAlgo *string
}

func parse() *args {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	var args args
	args.connections = flag.Int("C", 16, "Concurrent connections.")
	args.dir = flag.String("D", dir, "Store directory.")
	args.checksum = flag.String("checksum", "", "Expected checksum hash (hex encoded).")
	args.checksumAlgo = flag.String("checksum-algo", "md5", "Checksum algorithm: md5, sha1, or sha256.")
	flag.Parse()
	t := flag.Args()
	if len(t) > 0 {
		args.url = t[0]
	} else {
		gPrintln("missing url parameter, for example: gopeed https://www.google.com or gopeed bt.torrent or gopeed magnet:?xt=urn:btih:...")
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
