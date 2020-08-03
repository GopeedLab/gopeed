package http

import (
	"encoding/json"
	"fmt"
	"github.com/monkeyWie/gopeed/download/common"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestFetcher_Resolve(t *testing.T) {
	fetcher := &Fetcher{}
	resolve, err := fetcher.Resolve(&common.Request{
		// URL: "https://github.com/monkeyWie/github-actions-demo/releases/download/v1.0.6/simple-v1.0.6-windows-x64.zip",
		URL: "http://192.168.200.163:8088/go1.12.7.linux-amd64.tar.gz",
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
