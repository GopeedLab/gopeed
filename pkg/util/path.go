package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	syspath "path"
	"path/filepath"
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

	ext := syspath.Ext(name)
	var nameTpl string
	// Special case: if the extension is the entire filename (like .gitignore),
	// or if index of last dot is 0 (starts with dot), treat it as no extension
	if ext == "" || ext == name || (len(ext) > 0 && strings.LastIndex(name, ".") == 0) {
		// No extension or hidden file without extension
		nameTpl = name + " (%d)"
	} else {
		// Has extension
		nameWithoutExt := name[:len(name)-len(ext)]
		nameTpl = nameWithoutExt + " (%d)" + ext
	}
	for i := 1; ; i++ {
		newName := fmt.Sprintf(nameTpl, i)
		newPath := syspath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newName, nil
		}
	}
}

// CopyDir Copy all files to the target directory, if the file already exists, it will be overwritten.
// Remove target file if the source file is not exist.
func CopyDir(source string, target string, excludeDir ...string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	if err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if len(excludeDir) > 0 {
				for _, dir := range excludeDir {
					if info.IsDir() && info.Name() == dir {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(target, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		if err := copyForce(path, targetPath); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	if err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if len(excludeDir) > 0 {
				for _, dir := range excludeDir {
					if info.IsDir() && info.Name() == dir {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}
		relPath, err := filepath.Rel(target, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(target, relPath)
		sourcePath := filepath.Join(source, relPath)
		// if source file is not exist, remove target file
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			if err := SafeRemove(targetPath); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// RmAndMkDirAll remove and create directory, if the directory already exists, it will be overwritten.
func RmAndMkDirAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	return nil
}

// copy file, if the target file already exists, it will be overwritten.
func copyForce(source string, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
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
	// if file exists and is a file
	if err == nil && !info.IsDir() {
		return true
	}
	return false
}

// ReplaceInvalidFilename replace invalid path characters
func ReplaceInvalidFilename(path string) string {
	if path == "" {
		return ""
	}
	for _, char := range invalidPathChars {
		path = strings.ReplaceAll(path, char, "_")
	}
	return path
}
