//go:build windows

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func install(killSignalChan chan<- any, updateChannel, packagePath, destDir string) (bool, error) {
	switch updateChannel {
	case "windowsInstaller":
		return false, installByInstaller(killSignalChan, packagePath, destDir)
	default:
		return true, installByPortable(killSignalChan, packagePath, destDir)
	}
}

// installByInstaller extracts the installer from the zip file and runs it
func installByInstaller(killSignalChan chan<- any, packagePath, destDir string) error {
	// Create a temp directory for extraction
	tempDir, err := os.MkdirTemp("", "gopeed_update")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Extract the zip file
	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Find the installer file
	var installerPath string
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Extract file
		path := filepath.Join(tempDir, file.Name)

		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
		if err != nil {
			return err
		}

		// If this is likely an installer (.exe, .msi), save its path
		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext == ".exe" || ext == ".msi" {
			installerPath = path
		}
	}

	if installerPath == "" {
		return fmt.Errorf("no installer found in the update package")
	}

	// Run the installer
	cmd := exec.Command(installerPath)
	if err := cmd.Start(); err != nil {
		return err
	}

	killSignalChan <- nil
	return nil
}

// installByPortable extracts the portable version to the destination directory
func installByPortable(killSignalChan chan<- any, packagePath, destDir string) error {
	killSignalChan <- nil

	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
