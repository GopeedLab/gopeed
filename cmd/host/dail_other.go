//go:build !windows
// +build !windows

package main

import (
	"net"
	"os"
	"path/filepath"
)

func Dial() (net.Conn, error) {
	// Get binary path
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return net.Dial("unix", filepath.Join(filepath.Dir(exe), "gopeed_host.sock"))
}
