//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func install(killSignalChan chan<- any, updateChannel, packagePath, destDir string) (bool, error) {
	return true, installByDmg(killSignalChan, packagePath, destDir)
}

// installByDmg handles macOS dmg package installation
func installByDmg(killSignalChan chan<- any, packagePath, destDir string) error {
	// /Applications/Gopeed.app/Contents/MacOS -> /Applications
	appPath := getParentDir(getParentDir(getParentDir(destDir)))
	output, err := exec.Command("hdiutil", "attach", packagePath, "-nobrowse").Output()
	if err != nil {
		return err
	}

	mountPoint := ""
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "/Volumes/") {
			fields := strings.Fields(line)
			if len(fields) > 2 {
				mountPoint = fields[len(fields)-1]
				break
			}
		}
	}

	if mountPoint == "" {
		return fmt.Errorf("failed to get mount point")
	}

	// Detach the mounted DMG
	defer exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()

	matches, err := filepath.Glob(filepath.Join(mountPoint, "*.app"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no .app found in dmg")
	}

	killSignalChan <- nil

	// Copy the new app to the destination
	// cp -Rf /Volumes/GoPeed/GoPeed.app /Applications
	if err := exec.Command("cp", "-Rf", matches[0], appPath).Run(); err != nil {
		return err
	}

	return nil
}

// Get parent directory safely, handling trailing separators
func getParentDir(path string) string {
	// Remove trailing separators if they exist
	path = strings.TrimRight(path, string(filepath.Separator))
	// Now get the parent directory
	return filepath.Dir(path)
}
