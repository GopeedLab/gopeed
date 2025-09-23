package main

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func Dial() (net.Conn, error) {
	return winio.DialPipe(`\\.\pipe\gopeed_host`, nil)
}
