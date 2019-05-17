package main

import (
	"fmt"
	"net"
)

func main() {
	ip := net.ParseIP("127.0.0.1")
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: ip, Port: 9981}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	conn.Write([]byte("hello"))
	conn.Write([]byte("test"))
	conn.Read([]byte{})
	fmt.Printf("<%s>\n", conn.RemoteAddr())
}
