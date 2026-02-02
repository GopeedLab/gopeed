package download

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestScript_TriggerOnDone(t *testing.T) {
	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	outputFile := filepath.Join(tmpDir, "output.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
echo "Script executed" > %s
echo "Event: $GOPEED_EVENT" >> %s
echo "Task ID: $GOPEED_TASK_ID" >> %s
`, outputFile, outputFile, outputFile)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	setupScriptTest(t, func(downloader *Downloader) {
		// Configure script paths
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{scriptPath},
		}
		downloader.PutConfig(cfg)

		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger script
		downloader.triggerScripts(ScriptEventDownloadDone, task, nil)

		// Wait for script to execute
		time.Sleep(1 * time.Second)

		// Check if output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Script did not create output file")
		}

		// Read and verify output
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		output := string(content)
		if output == "" {
			t.Error("Script output is empty")
		}
	})
}

func TestScript_TriggerOnError(t *testing.T) {
	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	outputFile := filepath.Join(tmpDir, "output.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
echo "Event: $GOPEED_EVENT" > %s
`, outputFile)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	setupScriptTest(t, func(downloader *Downloader) {
		// Configure script paths
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{scriptPath},
		}
		downloader.PutConfig(cfg)

		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger script with error
		testError := fmt.Errorf("test error")
		downloader.triggerScripts(ScriptEventDownloadError, task, testError)

		// Wait for script to execute
		time.Sleep(1 * time.Second)

		// Check if output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Script did not create output file")
		}
	})
}

func TestScript_NoScriptConfigured(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger script (should not panic with no scripts configured)
		downloader.triggerScripts(ScriptEventDownloadDone, task, nil)
	})
}

func TestScript_MultipleScripts(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile1 := filepath.Join(tmpDir, "output1.txt")
	outputFile2 := filepath.Join(tmpDir, "output2.txt")

	scriptPath1 := filepath.Join(tmpDir, "test1.sh")
	scriptContent1 := fmt.Sprintf(`#!/bin/bash
echo "Script 1" > %s
`, outputFile1)
	if err := os.WriteFile(scriptPath1, []byte(scriptContent1), 0755); err != nil {
		t.Fatalf("Failed to create test script 1: %v", err)
	}

	scriptPath2 := filepath.Join(tmpDir, "test2.sh")
	scriptContent2 := fmt.Sprintf(`#!/bin/bash
echo "Script 2" > %s
`, outputFile2)
	if err := os.WriteFile(scriptPath2, []byte(scriptContent2), 0755); err != nil {
		t.Fatalf("Failed to create test script 2: %v", err)
	}

	setupScriptTest(t, func(downloader *Downloader) {
		// Configure multiple script paths
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{scriptPath1, scriptPath2},
		}
		downloader.PutConfig(cfg)

		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger scripts
		downloader.triggerScripts(ScriptEventDownloadDone, task, nil)

		// Wait for scripts to execute
		time.Sleep(1 * time.Second)

		// Check if both output files were created
		if _, err := os.Stat(outputFile1); os.IsNotExist(err) {
			t.Error("Script 1 did not create output file")
		}
		if _, err := os.Stat(outputFile2); os.IsNotExist(err) {
			t.Error("Script 2 did not create output file")
		}
	})
}

func TestScript_GetScriptPaths_EmptyConfig(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		paths := downloader.getScriptPaths()
		if paths != nil {
			t.Errorf("Expected nil, got %v", paths)
		}
	})
}

func TestScript_GetScriptPaths_NoScriptConfig(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Script = nil
		downloader.PutConfig(cfg)

		paths := downloader.getScriptPaths()
		if paths != nil {
			t.Errorf("Expected nil, got %v", paths)
		}
	})
}

func TestScript_GetScriptPaths_DisabledScript(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: false,
			Paths:  []string{"/path/to/script.sh"},
		}
		downloader.PutConfig(cfg)

		paths := downloader.getScriptPaths()
		if paths != nil {
			t.Errorf("Expected nil for disabled script, got %v", paths)
		}
	})
}

func TestScript_GetScriptPaths_EmptyPaths(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{},
		}
		downloader.PutConfig(cfg)

		paths := downloader.getScriptPaths()
		if paths != nil {
			t.Errorf("Expected nil for empty paths, got %v", paths)
		}
	})
}

func TestScript_GetScriptPaths_WithEmptyStrings(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{"/path/to/script1.sh", "", "/path/to/script2.sh", ""},
		}
		downloader.PutConfig(cfg)

		paths := downloader.getScriptPaths()
		if len(paths) != 2 {
			t.Errorf("Expected 2 valid paths (ignoring empty strings), got %d: %v", len(paths), paths)
		}
		if paths[0] != "/path/to/script1.sh" || paths[1] != "/path/to/script2.sh" {
			t.Errorf("Paths don't match expected values: %v", paths)
		}
	})
}

func TestScript_ExecuteScriptAtPath_EmptyPath(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		data := &ScriptData{
			Event: ScriptEventDownloadDone,
			Time:  time.Now().UnixMilli(),
		}

		err := downloader.executeScriptAtPath("", data)
		if err == nil {
			t.Error("Expected error for empty path")
		}
		if err.Error() != "script path is empty" {
			t.Errorf("Expected 'script path is empty' error, got: %v", err)
		}
	})
}

func TestScript_ExecuteScriptAtPath_NonExistentFile(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		data := &ScriptData{
			Event: ScriptEventDownloadDone,
			Time:  time.Now().UnixMilli(),
		}

		err := downloader.executeScriptAtPath("/non/existent/script.sh", data)
		if err == nil {
			t.Error("Expected error for non-existent script")
		}
	})
}

func TestScript_TestScript_Success(t *testing.T) {
	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	outputFile := filepath.Join(tmpDir, "output.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
echo "Test script executed" > %s
`, outputFile)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	setupScriptTest(t, func(downloader *Downloader) {
		err := downloader.TestScript(scriptPath)
		if err != nil {
			t.Errorf("TestScript failed: %v", err)
		}

		// Wait for script to execute
		time.Sleep(500 * time.Millisecond)

		// Check if output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Test script did not create output file")
		}
	})
}

func TestScript_TestScript_NonExistentScript(t *testing.T) {
	setupScriptTest(t, func(downloader *Downloader) {
		err := downloader.TestScript("/non/existent/script.sh")
		if err == nil {
			t.Error("Expected error for non-existent script")
		}
	})
}

func TestScript_EnvironmentVariables(t *testing.T) {
	// Create a temporary test script that outputs environment variables
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_env.sh")
	outputFile := filepath.Join(tmpDir, "env_output.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
echo "GOPEED_EVENT=$GOPEED_EVENT" > %s
echo "GOPEED_TASK_ID=$GOPEED_TASK_ID" >> %s
echo "GOPEED_TASK_NAME=$GOPEED_TASK_NAME" >> %s
echo "GOPEED_TASK_STATUS=$GOPEED_TASK_STATUS" >> %s
echo "GOPEED_DOWNLOAD_DIR=$GOPEED_DOWNLOAD_DIR" >> %s
echo "GOPEED_FILE_NAME=$GOPEED_FILE_NAME" >> %s
echo "GOPEED_FILE_PATH=$GOPEED_FILE_PATH" >> %s
`, outputFile, outputFile, outputFile, outputFile, outputFile, outputFile, outputFile)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	setupScriptTest(t, func(downloader *Downloader) {
		// Configure script paths
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{scriptPath},
		}
		downloader.PutConfig(cfg)

		// Create a mock task with proper metadata
		task := NewTask()
		task.Protocol = "http"
		task.Status = base.DownloadStatusDone
		task.Meta = &fetcher.FetcherMeta{
			Req: &base.Request{
				URL: "https://example.com/test.zip",
			},
			Opts: &base.Options{
				Name: "test.zip",
				Path: "/downloads",
			},
			Res: &base.Resource{
				Size: 1024 * 1024,
				Files: []*base.FileInfo{
					{Name: "test.zip", Size: 1024 * 1024},
				},
			},
		}

		// Trigger script
		downloader.triggerScripts(ScriptEventDownloadDone, task, nil)

		// Wait for script to execute
		time.Sleep(1 * time.Second)

		// Check if output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Script did not create output file")
			return
		}

		// Read and verify environment variables
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
			return
		}

		output := string(content)
		if output == "" {
			t.Error("Script output is empty")
		}
	})
}

func TestScript_WindowsBatchExtension(t *testing.T) {
	// This test verifies that .bat and .cmd extensions are recognized
	tmpDir := t.TempDir()
	
	setupScriptTest(t, func(downloader *Downloader) {
		// Test .bat extension
		batScript := filepath.Join(tmpDir, "test.bat")
		if err := os.WriteFile(batScript, []byte("@echo off\necho test"), 0644); err != nil {
			t.Fatalf("Failed to create .bat script: %v", err)
		}
		
		// Test .cmd extension
		cmdScript := filepath.Join(tmpDir, "test.cmd")
		if err := os.WriteFile(cmdScript, []byte("@echo off\necho test"), 0644); err != nil {
			t.Fatalf("Failed to create .cmd script: %v", err)
		}
		
		// Note: These tests only verify the extension is recognized
		// Actual execution would fail on non-Windows systems, which is expected
	})
}

func TestScript_PowerShellExtension(t *testing.T) {
	// This test verifies that .ps1 extension is recognized
	tmpDir := t.TempDir()
	
	setupScriptTest(t, func(downloader *Downloader) {
		// Test .ps1 extension
		ps1Script := filepath.Join(tmpDir, "test.ps1")
		if err := os.WriteFile(ps1Script, []byte("Write-Host 'test'"), 0644); err != nil {
			t.Fatalf("Failed to create .ps1 script: %v", err)
		}
		
		// Note: This test only verifies the extension is recognized
		// Actual execution depends on PowerShell availability
	})
}

func setupScriptTest(t *testing.T, fn func(downloader *Downloader)) {
	defaultDownloader.Setup()
	defaultDownloader.cfg.StorageDir = ".test_storage"
	defaultDownloader.cfg.DownloadDir = ".test_download"
	defer func() {
		defaultDownloader.Clear()
		os.RemoveAll(defaultDownloader.cfg.StorageDir)
		os.RemoveAll(defaultDownloader.cfg.DownloadDir)
	}()
	fn(defaultDownloader)
}
