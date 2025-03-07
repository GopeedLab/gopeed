//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func install(updateChannel, packagePath, destDir string) (bool, error) {
	return true, installByDmg(packagePath, destDir)
}

// installByDmg handles macOS dmg doUpdateion and installation
func installByDmg(packagePath, destDir string) error {
	output, err := exec.Command("hdiutil", "attach", packagePath, "-nobrowse", "-quiet").Output()
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

	matches, err := filepath.Glob(filepath.Join(mountPoint, "*.app"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no .app found in dmg")
	}

	// Remove existing app if present
	existingApp, _ := filepath.Glob(filepath.Join(destDir, "*.app"))
	if len(existingApp) > 0 {
		if err := exec.Command("rm", "-rf", existingApp[0]).Run(); err != nil {
			return fmt.Errorf("failed to remove existing app: %w", err)
		}
	}

	// Copy the new app to the destination
	if err := exec.Command("cp", "-Rf", matches[0], destDir).Run(); err != nil {
		return err
	}

	// Detach the mounted DMG
	return exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()
}
