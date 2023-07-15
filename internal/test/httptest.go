package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
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

type SlowFileServer struct {
	delay   time.Duration
	handler http.Handler
}

func (s *SlowFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(s.delay)
	s.handler.ServeHTTP(w, r)
}

func StartTestSlowFileServer(delay time.Duration) net.Listener {
	return startTestServer(func() http.Handler {
		return &SlowFileServer{
			delay:   delay,
			handler: http.FileServer(http.Dir(Dir)),
		}
	})
}

func StartTestCustomServer() net.Listener {
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
		mux.HandleFunc("/disposition", func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Disposition", "attachment; filename=\""+BuildName+"\"")
			writer.Header().Set("Content-Type", "application/octet-stream")
			writer.Header().Set("Content-Length", fmt.Sprintf("%d", BuildSize))
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

func StartTestPostServer() net.Listener {
	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+BuildName, func(writer http.ResponseWriter, request *http.Request) {
			if request.Method == "POST" && request.Header.Get("Authorization") != "" {
				var data map[string]interface{}
				if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
					panic(err)
				}
				if data["name"] == BuildName {
					file, err := os.Open(BuildFile)
					if err != nil {
						panic(err)
					}
					defer file.Close()
					io.Copy(writer, file)
				}
			}
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
	defer file.Close()
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

	return &shutdownListener{
		server:   server,
		Listener: listener,
	}
}

type shutdownListener struct {
	server *http.Server
	net.Listener
}

func (c *shutdownListener) Close() error {
	closeErr := c.server.Shutdown(context.Background())
	if err := ifExistAndRemove(BuildFile); err != nil {
		fmt.Println(err)
	}
	if err := ifExistAndRemove(DownloadFile); err != nil {
		fmt.Println(err)
	}
	return closeErr
}

func ifExistAndRemove(name string) error {
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return os.Remove(name)
	}
	return nil
}
