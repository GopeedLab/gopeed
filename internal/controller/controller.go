package controller

import (
	"golang.org/x/net/proxy"
	"net"
	"os"
	"time"
)

type Controller interface {
	Touch(name string, size int64) (file *os.File, err error)
	Open(name string) (file *os.File, err error)
	Write(name string, offset int64, buf []byte) (int, error)
	Close(name string) error
	ContextDialer() (proxy.Dialer, error)
}

type DefaultController struct {
	Files map[string]*os.File
}

func NewController() *DefaultController {
	return &DefaultController{Files: make(map[string]*os.File)}
}

func (c *DefaultController) Touch(name string, size int64) (file *os.File, err error) {
	file, err = os.Create(name)
	if size > 0 {
		err = os.Truncate(name, size)
		if err != nil {
			return nil, err
		}
	}
	if err == nil {
		c.Files[name] = file
	}
	return
}

func (c *DefaultController) Open(name string) (file *os.File, err error) {
	file, err = os.OpenFile(name, os.O_RDWR, os.ModePerm)
	if err == nil {
		c.Files[name] = file
	}
	return
}

func (c *DefaultController) Write(name string, offset int64, buf []byte) (int, error) {
	return c.Files[name].WriteAt(buf, offset)
}

func (c *DefaultController) Close(name string) error {
	err := c.Files[name].Close()
	delete(c.Files, name)
	return err
}

func (c *DefaultController) ContextDialer() (proxy.Dialer, error) {
	// return proxy.SOCKS5("tpc", "127.0.0.1:9999", nil, nil)
	var dialer proxy.Dialer
	return &DialerWarp{dialer: dialer}, nil
}

type DialerWarp struct {
	dialer proxy.Dialer
}

type ConnWarp struct {
	conn net.Conn
}

func (c *ConnWarp) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *ConnWarp) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *ConnWarp) Close() error {
	return c.conn.Close()
}

func (c *ConnWarp) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *ConnWarp) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *ConnWarp) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *ConnWarp) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *ConnWarp) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (d *DialerWarp) Dial(network, addr string) (c net.Conn, err error) {
	conn, err := d.dialer.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return &ConnWarp{conn: conn}, nil
}
