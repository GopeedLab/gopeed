package http

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/monkeyWie/gopeed-core/download/common"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"
)

const testDir = "./"
const buildSize = 200 * 1024 * 1024
const buildName = "build.data"
const buildFile = testDir + buildName
const downloadName = "download.data"
const downloadFile = testDir + downloadName

func TestFetcher_Resolve(t *testing.T) {
	testResolve(startTestFileServer, &common.Resource{
		Size:  buildSize,
		Range: true,
		Files: []*common.FileInfo{
			{
				Name: buildName,
				Size: buildSize,
			},
		},
	}, t)
	testResolve(startTestChunkedServer, &common.Resource{
		Size:  0,
		Range: false,
		Files: []*common.FileInfo{
			{
				Name: buildName,
				Size: 0,
			},
		},
	}, t)
}

func testResolve(startTestServer func() net.Listener, want *common.Resource, t *testing.T) {
	listener := startTestServer()
	defer listener.Close()
	fetcher := NewFetcher()
	res, err := fetcher.Resolve(&common.Request{
		URL: "http://" + listener.Addr().String() + "/" + buildName,
	})
	if err != nil {
		t.Fatal(err)
	}
	res.Req = nil
	if !reflect.DeepEqual(want, res) {
		t.Errorf("Resolve error = %v, want %v", res, want)
	}
}

func TestFetcher_DownloadNormal(t *testing.T) {
	listener := startTestFileServer()
	defer listener.Close()
	// 正常下载
	downloadNormal(listener, 1, t)
	downloadNormal(listener, 5, t)
	downloadNormal(listener, 8, t)
	downloadNormal(listener, 16, t)
}

func TestFetcher_DownloadContinue(t *testing.T) {
	listener := startTestFileServer()
	defer listener.Close()
	// 暂停继续
	downloadContinue(listener, 1, t)
	downloadContinue(listener, 5, t)
	downloadContinue(listener, 8, t)
	downloadContinue(listener, 16, t)
}

func TestFetcher_DownloadChunked(t *testing.T) {
	listener := startTestChunkedServer()
	defer listener.Close()
	// chunked编码下载
	//downloadNormal(listener, 1, t)
	downloadContinue(listener, 1, t)
}

func TestFetcher_DownloadRetry(t *testing.T) {
	listener := startTestRetryServer()
	defer listener.Close()
	// chunked编码下载
	downloadNormal(listener, 1, t)
}

func startTestFileServer() net.Listener {
	return startTestServer(func() http.Handler {
		return http.FileServer(http.Dir(testDir))
	})
}

func startTestChunkedServer() net.Listener {
	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+buildName, func(writer http.ResponseWriter, request *http.Request) {
			file, err := os.Open(buildFile)
			if err != nil {
				panic(err)
			}
			defer file.Close()
			io.Copy(writer, file)
		})
		return mux
	})
}

func startTestRetryServer() net.Listener {
	counter := 0
	return startTestServer(func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/"+buildName, func(writer http.ResponseWriter, request *http.Request) {
			counter++
			if counter != 1 && counter < 5 {
				writer.WriteHeader(500)
				return
			}
			file, err := os.Open(buildFile)
			if err != nil {
				panic(err)
			}
			defer file.Close()
			io.Copy(writer, file)
		})
		return mux
	})
}

func startTestServer(serverHandle func() http.Handler) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	file, err := os.Create(buildFile)
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
		if size+l >= buildSize {
			file.WriteAt(buf[0:buildSize-size], size)
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
		if err := ifExistAndRemove(downloadFile); err != nil {
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

func downloadReady(listener net.Listener, connections int, t *testing.T) common.Process {
	fetcher := NewFetcher()
	fetcher.InitCtl(common.NewController())
	res, err := fetcher.Resolve(&common.Request{
		URL: "http://" + listener.Addr().String() + "/" + buildName,
	})
	if err != nil {
		t.Fatal(err)
	}
	process, err := fetcher.Create(res, &common.Options{
		Name:        downloadName,
		Path:        testDir,
		Connections: connections,
	})
	if err != nil {
		t.Fatal(err)
	}
	return process

}

func downloadNormal(listener net.Listener, connections int, t *testing.T) {
	process := downloadReady(listener, connections, t)
	err := process.Start()
	if err != nil {
		t.Fatal(err)
	}
	want := fileMd5(buildFile)
	got := fileMd5(downloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
	}
}

func downloadContinue(listener net.Listener, connections int, t *testing.T) {
	process := downloadReady(listener, connections, t)
	go func() {
		err := process.Start()
		if err != nil && err != common.PauseErr {
			t.Fatal(err)
		}
	}()
	time.Sleep(time.Millisecond * 200)
	if err := process.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 200)
	if err := process.Continue(); err != nil {
		t.Fatal(err)
	}
	want := fileMd5(buildFile)
	got := fileMd5(downloadFile)
	if want != got {
		t.Errorf("Download error = %v, want %v", got, want)
	}
}

func fileMd5(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	// Tell the program to call the following function when the current function returns
	defer file.Close()

	// Open a new hash interface to write to
	hash := md5.New()

	// Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}
