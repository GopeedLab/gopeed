package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"time"
)

var (
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procSetFileValidData = modkernel32.NewProc("SetFileValidData")
)

func main() {
	var err error
	name := "d:/test.data"
	var size int64 = 1024 * 1024 * 1024 * 4

	err = os.Truncate(name, size)
	if err != nil {
		panic(err)
	}

	var hwnd HWND

	err = ShellExecute(hwnd,
		"runas",
		"fsutil.exe",
		"file setValidData d:\\test.data 4294967296",
		"d:\\",
		syscall.SW_HIDE)

	//err = exec.Command("fsutil", "file", "setValidData", "d:\\test.data", "4294967296").Run()
	if err != nil {
		panic(err)
	}

	file, _ := os.OpenFile(name, os.O_RDWR, 0644)
	defer os.Remove(name)
	defer file.Close()
	defer func() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("输入回车结束")
		reader.ReadString('\n')
	}()

	/*process, err := syscall.GetCurrentProcess()
	if err != nil {
		panic(err)
	}
	var token syscall.Token
	err = syscall.OpenProcessToken(process, syscall.TOKEN_ADJUST_PRIVILEGES|syscall.TOKEN_QUERY, &token)
	if err != nil {
		panic(err)
	}


	_, _, en := syscall.Syscall(procSetFileValidData.Addr(), 2, file.Fd(), uintptr(size), 0)
	if en != 0 {
		fmt.Println(en)
		return
	}*/

	buf := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	var (
		tm int64
	)

	tm = time.Now().UnixNano()
	_, err = file.WriteAt(buf, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("write use:%d\n", toMill(time.Now().UnixNano()-tm))

	tm = time.Now().UnixNano()
	_, err = file.WriteAt(buf, 1024*1024*1024*2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("write use:%d\n", toMill(time.Now().UnixNano()-tm))
}

func toMill(nano int64) int64 {
	return nano / int64(time.Millisecond)
}
