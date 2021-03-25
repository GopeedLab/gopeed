package test

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
)

const (
	BuildName = "build.data"
	BuildSize = 200 * 1024 * 1024
	Dir       = "./"
	BuildFile = Dir + BuildName

	DownloadName = "download.data"
	DownloadFile = Dir + DownloadName
)

func StartTestFileServer() net.Listener {
	return startTestServer(func() http.Handler {
		return http.FileServer(http.Dir(Dir))
	})
}

func StartTestChunkedServer() net.Listener {
	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+BuildName, func(writer http.ResponseWriter, request *http.Request) {
			file, err := os.Open(BuildFile)
			if err != nil {
				panic(err)
			}
			defer file.Close()
			io.Copy(writer, file)
		})
		return mux
	})
}

func StartTestRetryServer() net.Listener {
	counter := 0
	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+BuildName, func(writer http.ResponseWriter, request *http.Request) {
			counter++
			if counter != 1 && counter < 5 {
				writer.WriteHeader(500)
				return
			}
			file, err := os.Open(BuildFile)
			if err != nil {
				panic(err)
			}
			defer file.Close()
			io.Copy(writer, file)
		})
		return mux
	})
}

func StartTestErrorServer() net.Listener {
	counter := 0
	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+BuildName, func(writer http.ResponseWriter, request *http.Request) {
			counter++
			if counter != 1 {
				writer.WriteHeader(500)
				return
			}
		})
		return mux
	})
}

func startTestServer(serverHandle func() http.Handler) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	file, err := os.Create(BuildFile)
	if err != nil {
		panic(err)
	}
	// 随机生成一个文件
	l := int64(8192)
	buf := make([]byte, l)
	size := int64(0)
	for {
		_, err := rand.Read(buf)
		if err != nil {
			panic(err)
		}
		if size+l >= BuildSize {
			file.WriteAt(buf[0:BuildSize-size], size)
			break
		}
		file.WriteAt(buf, size)
		size += l
	}
	server := &http.Server{}
	server.Handler = serverHandle()
	go server.Serve(listener)

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
	defer func() {
		c.File.Close()
		if err := ifExistAndRemove(c.File.Name()); err != nil {
			fmt.Println(err)
		}
		if err := ifExistAndRemove(DownloadFile); err != nil {
			fmt.Println(err)
		}
	}()
	return c.Listener.Close()
}

func ifExistAndRemove(name string) error {
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return os.Remove(name)
	}
	return nil
}
