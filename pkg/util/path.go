package util

import (
	"errors"
	"io"
	"os"
	syspath "path"
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

// SafeRemoveAll remove file and parent directories safely
func SafeRemoveAll(path string, names []string) error {
	for _, name := range names {
		err := SafeRemove(syspath.Join(path, name))
		if err != nil {
			return err
		}
		if err := safeRemoveParent(path, name); err != nil {
			return err
		}
	}
	return nil
}

func safeRemoveParent(path string, subPath string) error {
	currPath := syspath.Dir(subPath)
	if currPath == "." {
		return nil
	}
	// if directory is empty, remove it
	dir := syspath.Join(path, syspath.Dir(subPath))
	empty, err := isEmpty(dir)
	if err != nil {
		return err
	}
	if empty {
		err = SafeRemove(dir)
		if err != nil {
			return err
		}
		return safeRemoveParent(path, currPath)
	}
	return nil
}

// check directory is empty
func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
