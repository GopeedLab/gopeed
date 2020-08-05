package http

import (
	"encoding/json"
	"fmt"
	"github.com/monkeyWie/gopeed/download/common"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func startMockServer() net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	file, err := os.Create("./test.data")
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 1024)
	for i := 0; i < 20*1024; i++ {
		_, err := rand.Read(buf)
		if err != nil {
			panic(err)
		}
		file.WriteAt(buf, int64(i*1024))
	}

	go func() {
		http.Serve(listener, http.FileServer(http.Dir("./")))
	}()
	return &delFileListener{
		File:     file,
		Listener: listener,
	}
}

type delFileListener struct {
	*os.File
	net.Listener
}

func (c *delFileListener) Close() error {
	c.File.Close()
	err := os.Remove(c.File.Name())
	if err != nil {
		fmt.Println(err)
		return err
	}
	return c.Listener.Close()
}

func TestFetcher_Resolve(t *testing.T) {
	listener := startMockServer()
	defer listener.Close()
	fetcher := &Fetcher{}
	resolve, err := fetcher.Resolve(&common.Request{
		URL: "http://" + listener.Addr().String() + "/test.data",
	})
	if err != nil {
		t.Fatal(err)
	}
	buf, err := json.Marshal(resolve)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", string(buf))
}

func Test2(t *testing.T) {
	socks5, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := socks5.Dial("tcp", "www.baidu.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte("GET"))
}

func Test3(t *testing.T) {
	var u User
	fmt.Println((&u).Say1())
	fmt.Println(u.Say2())
}

func Test4(t *testing.T) {
	resp, err := http.Get("http://192.168.200.163:8088/go1.12.7.linux-amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	go func() {
		time.Sleep(time.Second * 3)
		err2 := resp.Body.Close()
		if err2 != nil {
			fmt.Println("close err2:" + err2.Error())
		}
	}()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func Test5(t *testing.T) {
	name := "E:\\test\\gopeed\\http\\test.txt"
	err := os.Truncate(name, 10)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(name, os.O_RDWR, 0666)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteAt([]byte("test"), 5)
	if err != nil {
		t.Fatal(err)
	}
}

type User struct {
	name string
}

func (u *User) Say1() string {
	return "hello1 " + u.name
}

func (u User) Say2() string {
	return "hello2 " + u.name
}
