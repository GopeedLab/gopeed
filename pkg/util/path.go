package util

import (
	"errors"
	"fmt"
	"os"
	syspath "path"
	"strings"
)

func Dir(path string) string {
	dir := syspath.Dir(path)
	if dir == "." {
		return ""
	}
	return dir
}

func Filepath(path string, originName string, customName string) string {
	if customName == "" {
		customName = originName
	}
	return syspath.Join(path, customName)
}

// SafeRemove remove file safely, ignoring errors if the path does not exist.
func SafeRemove(name string) error {
	if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// CheckDuplicateAndRename rename duplicate file, add suffix (1) (2) ...
// if file name is a.txt, rename to a (1).txt
// if directory name is a, rename to a (1)
// return new name
func CheckDuplicateAndRename(path string) (string, error) {
	dir := syspath.Dir(path)
	name := syspath.Base(path)

	// if file not exists, return directly
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return name, nil
		}
		return "", err
	}

	index := strings.LastIndex(name, ".")
	var nameTpl string
	if index == -1 {
		nameTpl = name + " (%d)"
	} else {
		nameTpl = name[:index] + " (%d)" + name[index:]
	}
	for i := 1; ; i++ {
		newName := fmt.Sprintf(nameTpl, i)
		newPath := syspath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newName, nil
		}
	}
}

// GetSingleDir get the top level single folder name,if not exist, return empty string
func GetSingleDir(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	split := strings.Split(paths[0], "/")
	if len(split) == 0 || split[0] == "" {
		return ""
	}
	dir := split[0]
	for i := 1; i < len(paths); i++ {
		if !strings.HasPrefix(paths[i], dir) {
			return ""
		}
	}
	return dir
}

func CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o777)
	}
	return nil
}

// IsExistsFile check file exists and is a file
func IsExistsFile(path string) bool {
	info, err := os.Stat(path)
	// 判断路径是否存在，并且是一个文件
	if err == nil && !info.IsDir() {
		return true
	}
	return false
}
