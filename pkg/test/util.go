package test

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
)

func FileMd5(filePath string) string {
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

func ToJson(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}
