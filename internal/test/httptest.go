package test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/armon/go-socks5"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	BuildName = "build.data"
	BuildSize = 200 * 1024 * 1024
	Dir       = "./"
	BuildFile = Dir + BuildName

	ExternalDownloadUrl  = "https://raw.githubusercontent.com/GopeedLab/gopeed/v1.5.6/_docs/img/banner.png"
	ExternalDownloadName = "banner.png"
	ExternalDownloadSize = 26416
	//ExternalDownloadMd5 = "c67c6e3cae79a95342485676571e8a5c"

	DownloadName       = "download.data"
	DownloadRename     = "download (1).data"
	DownloadFile       = Dir + DownloadName
	DownloadRenameFile = Dir + DownloadRename
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
			if counter != 1 && counter < 2 {
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

// StartTestLimitServer connections limit server
func StartTestLimitServer(maxConnections int32, delay int64) net.Listener {
	var connections atomic.Int32

	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+BuildName, func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				connections.Add(-1)
			}()
			connections.Add(1)
			if maxConnections != 0 && connections.Load() > maxConnections {
				writer.WriteHeader(403)
				return
			}

			r := request.Header.Get("Range")
			if r == "" {
				writer.Header().Set("Content-Length", fmt.Sprintf("%d", BuildSize))
				writer.WriteHeader(200)
				file, err := os.Open(BuildFile)
				if err != nil {
					panic(err)
				}
				defer file.Close()
				slowCopy(writer, file, delay)
			} else {
				// split range
				s := strings.Split(r, "=")
				if len(s) != 2 {
					writer.WriteHeader(400)
					return
				}
				s = strings.Split(s[1], "-")
				if len(s) != 2 {
					writer.WriteHeader(400)
					return
				}
				start, err := strconv.ParseInt(s[0], 10, 64)
				if err != nil {
					writer.WriteHeader(400)
					return
				}
				end, err := strconv.ParseInt(s[1], 10, 64)
				if err != nil {
					writer.WriteHeader(400)
					return
				}
				if start < 0 || end < 0 || start > end {
					writer.WriteHeader(400)
					return
				}
				if end >= BuildSize {
					end = BuildSize - 1
				}
				writer.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
				writer.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, BuildSize))
				writer.Header().Set("Accept-Ranges", "bytes")
				writer.WriteHeader(206)
				file, err := os.Open(BuildFile)
				if err != nil {
					writer.WriteHeader(500)
					return
				}
				defer file.Close()
				file.Seek(start, 0)
				slowCopyN(writer, file, end-start+1, delay)
			}
		})
		return mux
	})
}

// slowCopyN copies n bytes from src to dst, speed limit is bytes per second
func slowCopy(dst io.Writer, src io.Reader, delay int64) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
		if delay > 0 {
			time.Sleep(time.Millisecond * time.Duration(delay))
		}
	}
	return written, err
}

func slowCopyN(dst io.Writer, src io.Reader, n int64, delay int64) (written int64, err error) {
	written, err = slowCopy(dst, io.LimitReader(src, n), delay)
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = io.EOF
	}
	return
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
	if err := ifExistAndRemove(DownloadRenameFile); err != nil {
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

func StartSocks5Server(usr, pwd string) net.Listener {
	conf := &socks5.Config{}
	if usr != "" && pwd != "" {
		conf.Credentials = socks5.StaticCredentials{
			usr: pwd,
		}
	}

	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0") // 你可以根据需要更改监听地址
	if err != nil {
		panic(err)
	}
	go server.Serve(listener)
	return listener
}

func AssertResourceEqual(want, got *base.Resource) bool {
	// Ignore ctime
	if got != nil && len(got.Files) > 0 {
		got.Files[0].Ctime = nil
	}
	return reflect.DeepEqual(want, got)
}
