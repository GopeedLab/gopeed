package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type User struct {
	Age  uint32
	Sex  byte
	Name [20]byte
}

type Buffer struct {
	buf   []byte
	limit int
}

func NewBuffer(buf []byte) *Buffer {
	return &Buffer{
		buf:   buf,
		limit: 0,
	}
}

func main() {
	user := &User{1, 2, [20]byte{}}
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, user)
	fmt.Println(buf.Bytes())
}

/*func (buffer Buffer) ReadByte() (byte, error) {

}

func (buffer Buffer) ReadByte() (byte, error)()  {

}
*/
