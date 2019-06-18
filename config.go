package main

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	HTTPDownloadAddress string
	ParrallelsNumber    int
}

var config Config

func Init() {
	flag.StringVarP(&config.HTTPDownloadAddress, "http-download", "d", "", "http download url")
	flag.IntVarP(&config.ParrallelsNumber, "http-parrallels", "n", 8, "http download parralles")

	flag.Parse()
}
