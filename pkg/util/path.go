package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	syspath "path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
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

const (
	// MaxFilenameLength is the maximum length in bytes for a filename
	MaxFilenameLength = 100
	// maxExtensionLength is the maximum length in bytes for a file extension
	// to be treated as a valid extension. Extensions longer than this are
	// treated as part of the filename to avoid edge cases.
	maxExtensionLength = 20
)

// SafeFilename sanitizes a filename by replacing invalid characters and truncating to a safe length.
// It performs two operations:
// 1. Replaces invalid path characters (platform-specific) with underscores
// 2. Truncates filename to MaxFilenameLength bytes while preserving the file extension
// The function handles UTF-8 multi-byte characters correctly by truncating at valid boundaries.
func SafeFilename(filename string) string {
	if filename == "" {
		return ""
	}
	
	// Step 1: Replace invalid characters
	for _, char := range invalidPathChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	
	// Step 2: Truncate if needed
	if len(filename) <= MaxFilenameLength {
		return filename
	}

	// Find the extension (last dot in filename)
	ext := ""
	lastDot := strings.LastIndex(filename, ".")
	
	// Only treat as extension if:
	// 1. There is a dot
	// 2. The dot is not at the start (not a hidden file like .gitignore)
	// 3. The extension is reasonable length (< maxExtensionLength bytes) to avoid edge cases
	if lastDot > 0 && lastDot < len(filename)-1 && len(filename)-lastDot < maxExtensionLength {
		ext = filename[lastDot:]
		filename = filename[:lastDot]
	}

	// Calculate how much space we have for the base name
	availableLength := MaxFilenameLength - len(ext)
	
	// Ensure we have at least some space for the base name
	if availableLength < 1 {
		// Extension itself is too long or no room, just truncate everything at byte boundary
		return truncateAtValidUTF8Boundary(filename+ext, MaxFilenameLength)
	}

	// Truncate the base name at a valid UTF-8 boundary
	truncatedBase := truncateAtValidUTF8Boundary(filename, availableLength)
	
	return truncatedBase + ext
}

// ReplaceInvalidFilename replace invalid path characters
// Deprecated: Use SafeFilename instead which also handles length truncation
func ReplaceInvalidFilename(path string) string {
	if path == "" {
		return ""
	}
	for _, char := range invalidPathChars {
		path = strings.ReplaceAll(path, char, "_")
	}
	return path
}

// TruncateFilename truncates a filename to a maximum byte length while preserving the extension.
// Deprecated: Use SafeFilename instead which also handles invalid character replacement
func TruncateFilename(filename string, maxLength int) string {
	// If already short enough, return as-is
	if len(filename) <= maxLength {
		return filename
	}

	// Find the extension (last dot in filename)
	ext := ""
	lastDot := strings.LastIndex(filename, ".")
	
	// Only treat as extension if:
	// 1. There is a dot
	// 2. The dot is not at the start (not a hidden file like .gitignore)
	// 3. The extension is reasonable length (< maxExtensionLength bytes) to avoid edge cases
	if lastDot > 0 && lastDot < len(filename)-1 && len(filename)-lastDot < maxExtensionLength {
		ext = filename[lastDot:]
		filename = filename[:lastDot]
	}

	// Calculate how much space we have for the base name
	availableLength := maxLength - len(ext)
	
	// Ensure we have at least some space for the base name
	if availableLength < 1 {
		// Extension itself is too long or no room, just truncate everything at byte boundary
		return truncateAtValidUTF8Boundary(filename+ext, maxLength)
	}

	// Truncate the base name at a valid UTF-8 boundary
	truncatedBase := truncateAtValidUTF8Boundary(filename, availableLength)
	
	return truncatedBase + ext
}

// truncateAtValidUTF8Boundary truncates a string to at most maxBytes,
// ensuring we don't cut in the middle of a UTF-8 character
func truncateAtValidUTF8Boundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	
	// Truncate at byte position
	truncated := s[:maxBytes]
	
	// Find the last valid UTF-8 character boundary
	// Walk backwards to find where the last complete character ends
	for len(truncated) > 0 {
		// Check if this is a valid UTF-8 string
		if utf8.ValidString(truncated) {
			return truncated
		}
		// Remove one byte and try again
		truncated = truncated[:len(truncated)-1]
	}
	
	return truncated
}

// ReplacePathPlaceholders replaces date placeholders in a path with actual values
// Supported placeholders:
//   - %year%  - Current year (e.g., 2025)
//   - %month% - Current month (01-12)
//   - %day%   - Current day (01-31)
//   - %date%  - Full date format (2025-01-01)
func ReplacePathPlaceholders(path string) string {
	if path == "" {
		return ""
	}

	now := time.Now()
	year := fmt.Sprintf("%d", now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())
	date := fmt.Sprintf("%s-%s-%s", year, month, day)

	path = strings.ReplaceAll(path, "%year%", year)
	path = strings.ReplaceAll(path, "%month%", month)
	path = strings.ReplaceAll(path, "%day%", day)
	path = strings.ReplaceAll(path, "%date%", date)

	return path
}
