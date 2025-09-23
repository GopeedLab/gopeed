//go:build !windows
// +build !windows

package main

import (
	"net"
)

func Dial() (net.Conn, error) {
	return net.Dial("unix", "/tmp/gopeed_host.sock")
}
