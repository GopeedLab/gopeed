//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func install(killSignalChan chan<- any, updateChannel, packagePath, destDir string) (bool, error) {
	killSignalChan <- nil
	switch updateChannel {
	case "linuxDeb":
		return true, installByDeb(packagePath)
	case "linuxFlathub":
		return true, installByFlathub()
	case "linuxSnap":
		return true, installBySnap()
	default:
		return false, fmt.Errorf("unsupported update channel for Linux: %s", updateChannel)
	}
}

// executeInTerminal tries to execute a command in one of several common terminal emulators
func executeInTerminal(command string) error {
	terminals := []string{
		"gnome-terminal", // GNOME
		"konsole",        // KDE
		"xfce4-terminal", // XFCE
		"xterm",          // X11
	}

	command = fmt.Sprintf(`echo "Starting update..." && echo "[CMD] %s" && %s`, command, command)

	for _, term := range terminals {
		if _, err := exec.LookPath(term); err == nil {
			var cmd *exec.Cmd
			switch term {
			case "gnome-terminal", "xfce4-terminal":
				cmd = exec.Command(term, "--", "bash", "-c", command)
			case "konsole":
				cmd = exec.Command(term, "-e", "bash", "-c", command)
			case "xterm":
				cmd = exec.Command(term, "-e", command)
			}

			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable terminal emulator found. Please install gnome-terminal, konsole, xfce4-terminal or xterm")
}

// installByDeb installs the .deb package
func installByDeb(packagePath string) error {
	command := fmt.Sprintf(`sudo dpkg -i "%s"`, packagePath)
	return executeInTerminal(command)
}

// installByFlathub updates the application via Flathub
func installByFlathub() error {
	command := "flatpak update com.gopeed.Gopeed -y"
	return executeInTerminal(command)
}

// installBySnap updates the application via Snap
func installBySnap() error {
	command := "sudo snap refresh gopeed"
	return executeInTerminal(command)
}
