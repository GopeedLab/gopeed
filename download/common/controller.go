package common

import (
	"golang.org/x/net/proxy"
	"net"
	"os"
	"time"
)

// 下载请求
type Request struct {
	// 下载链接
	URL string
	// 附加信息
	Extra interface{}
}

// 资源信息
type Resource struct {
	Req *Request
	// 资源总大小
	Size int64
	// 是否支持断点下载
	Range bool
	// 资源所包含的文件列表
	Files []*FileInfo
}

type Options struct {
	// 保存文件名
	Name string
	// 保存目录
	Path string
	// 并发连接数
	Connections int
}

type FileInfo struct {
	Name       string
	Path       string
	Size       int64
	Downloaded int64
}

type Process interface {
	Start() error
	Pause() error
	Continue() error
	Delete() error
}

type Controller struct {
	Files map[string]*os.File
}

func (c *Controller) Touch(name string, size int64) error {
	if size > 0 {
		return os.Truncate(name, size)
	}
	_, err := os.Create(name)
	return err
}

func (c *Controller) Write(name string, offset int64, buf []byte) (int, error) {
	file, ok := c.Files[name]
	if !ok {
		var err error
		file, err = os.OpenFile(name, os.O_RDWR, 0666)
		if err != nil {
			return 0, err
		}
		c.Files[name] = file
	}
	return file.WriteAt(buf, offset)
}

func (c *Controller) ContextDialer() (proxy.Dialer, error) {
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
