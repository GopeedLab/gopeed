package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
)

// go build -ldflags="-s -w" -o ui/flutter/assets/exec/ github.com/GopeedLab/gopeed/cmd/updater

func main() {
	pid := flag.Int("pid", 0, "PID of the process to update")
	updateChannel := flag.String("channel", "", "Update channel")
	packagePath := flag.String("asset", "", "Path to the package asset")
	exeDir := flag.String("exeDir", "", "Directory of the entry executable")
	logPath := flag.String("log", "", "Log file path")
	flag.Parse()

	if *pid == 0 {
		log.Println("Invalid PID")
		os.Exit(1)
	}
	if *updateChannel == "" {
		log.Println("Invalid update channel")
		os.Exit(1)
	}

	if *logPath != "" {
		logFile, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err == nil {
			defer logFile.Close()
			log.SetOutput(logFile)
		}
	}

	var (
		restart bool
		err     error
	)
	if restart, err = update(*pid, *updateChannel, *packagePath, *exeDir); err != nil {
		log.Printf("Update failed: %v\n", err)
		os.Exit(1)
	}

	// Restart the application
	if restart {
		browser.OpenURL("gopeed:///")
	}

	// Delete package asset
	if *packagePath != "" {
		os.Remove(*packagePath)
	}

	os.Exit(0)
}

func update(pid int, updateChannel, packagePath, exeDir string) (restart bool, err error) {
	killSignalChan := make(chan any, 1)

	go func() {
		<-killSignalChan

		if err = killProcess(pid); err != nil {
			log.Printf("Failed to kill process: %v\n", err)
		}

		if err = waitForProcessExit(pid); err != nil {
			log.Printf("Failed to wait for process exit: %v\n", err)
		}
	}()

	log.Printf("Updating process updateChannel=%s packagePath=%s exeDir=%s\n", updateChannel, packagePath, exeDir)
	if restart, err = install(killSignalChan, updateChannel, packagePath, exeDir); err != nil {
		return false, errors.Wrap(err, "failed to install package")
	}

	return
}

func waitForProcessExit(pid int) error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		process, err := os.FindProcess(pid)
		if err != nil {
			// On some systems, error is returned if process doesn't exist
			return nil
		}

		// Send null signal to test if process exists
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			// If error occurs, the process no longer exists
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("process %d still running after timeout", pid)
}

func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return process.Kill()
}
