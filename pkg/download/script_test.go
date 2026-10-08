package download

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

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

func TestScript_TriggerOnError(t *testing.T) {
	scriptPath := getTestScriptPath(t, envDumpScriptName())
	ensureScriptExecutable(t, scriptPath)
	outputFile := filepath.Join(t.TempDir(), "env_output.txt")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	setupScriptTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Script = &base.ScriptConfig{
			Enable: true,
			Paths:  []string{scriptPath},
		}
		downloader.PutConfig(cfg)

		t.Setenv("GOPEED_TEST_OUTPUT_FILE", outputFile)
		id, err := downloader.CreateDirect(&base.Request{URL: server.URL + "/error.bin"}, &base.Options{
			Path: t.TempDir(),
			Name: "error.bin",
		})
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		waitForTaskStatus(t, downloader, id, base.DownloadStatusError, 10*time.Second)

		output := waitForFileContains(t, outputFile, "GOPEED_TASK_PATH=", 5*time.Second)
		for _, want := range []string{
			"GOPEED_EVENT=DOWNLOAD_ERROR",
			"GOPEED_TASK_ID=" + id,
			"GOPEED_TASK_NAME=error.bin",
			"GOPEED_TASK_STATUS=" + string(base.DownloadStatusError),
		} {
			if !strings.Contains(output, want) {
				t.Errorf("Expected %q in output, got: %s", want, output)
			}
		}
	})
}

func TestScript_TriggerOnError_ResolvedTask(t *testing.T) {
	scriptPath := getTestScriptPath(t, envDumpScriptName())
	ensureScriptExecutable(t, scriptPath)
	outputFile := filepath.Join(t.TempDir(), "env_output.txt")

	manager := &generationTestManager{holdOpen: true}
	downloader := NewDownloader(&DownloaderConfig{
		FetchManagers: []fetcher.FetcherManager{manager},
		Storage:       NewMemStorage(),
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Clear()

	cfg, _ := downloader.GetConfig()
	cfg.Script = &base.ScriptConfig{
		Enable: true,
		Paths:  []string{scriptPath},
	}
	downloader.PutConfig(cfg)

	t.Setenv("GOPEED_TEST_OUTPUT_FILE", outputFile)
	id, err := downloader.CreateDirect(&base.Request{URL: "generation://script-error"}, &base.Options{
		Path: t.TempDir(),
		Name: "failed.bin",
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for manager.starts.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Timeout waiting for the task to start")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// The task is resolved and started, so the script gets the same path it gets on DOWNLOAD_DONE.
	task := downloader.GetTask(id)
	taskPath := task.Meta.SingleFilepath()
	task.fetcher.(*generationTestFetcher).done <- errors.New("download failed mid-way")
	waitForTaskStatus(t, downloader, id, base.DownloadStatusError, 2*time.Second)

	output := waitForFileContains(t, outputFile, "GOPEED_TASK_PATH=", 5*time.Second)
	for _, want := range []string{
		"GOPEED_EVENT=DOWNLOAD_ERROR",
		"GOPEED_TASK_ID=" + id,
		"GOPEED_TASK_NAME=failed.bin",
		"GOPEED_TASK_STATUS=" + string(base.DownloadStatusError),
		"GOPEED_TASK_PATH=" + taskPath,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Expected %q in output, got: %s", want, output)
		}
	}
}

func createDownloadDoneTask(t *testing.T, downloadDir, fileName string) (*Task, string) {
	t.Helper()
	content := []byte("downloaded file")
	task := NewTask()
	task.Protocol = "http"
	task.Status = base.DownloadStatusDone
	task.Meta = &fetcher.FetcherMeta{
		Req: &base.Request{
			URL: "https://example.com/" + fileName,
		},
		Opts: &base.Options{
			Name: fileName,
			Path: filepath.ToSlash(downloadDir),
		},
		Res: &base.Resource{
			Size: int64(len(content)),
			Files: []*base.FileInfo{
				{Name: fileName, Size: int64(len(content))},
			},
		},
	}

	filePath := task.Meta.SingleFilepath()
	filePathOS := filepath.FromSlash(filePath)
	if err := os.MkdirAll(filepath.Dir(filePathOS), 0755); err != nil {
		t.Fatalf("Failed to create download dir: %v", err)
	}
	if err := os.WriteFile(filePathOS, content, 0644); err != nil {
		t.Fatalf("Failed to create download file: %v", err)
	}
	return task, filePath
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for file: %s", path)
}

// waitForFileContains waits until the file contains want and returns its content.
// Test scripts write the file line by line, so waiting for the file to exist is not enough.
func waitForFileContains(t *testing.T, path, want string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if content, err := os.ReadFile(path); err == nil && strings.Contains(string(content), want) {
			return string(content)
		}
		time.Sleep(50 * time.Millisecond)
	}
	content, _ := os.ReadFile(path)
	t.Fatalf("Timeout waiting for %q in file %s, got: %q", want, path, content)
	return ""
}

// envDumpScriptName returns the test script that dumps the GOPEED_* variables
// into the file named by GOPEED_TEST_OUTPUT_FILE.
func envDumpScriptName() string {
	if runtime.GOOS == "windows" {
		return "env_dump.bat"
	}
	return "env_dump.sh"
}

func getTestScriptPath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", "scripts", name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Missing test script %s: %v", path, err)
	}
	return path
}

func ensureScriptExecutable(t *testing.T, scriptPath string) {
	t.Helper()
	if filepath.Ext(scriptPath) != ".sh" {
		return
	}
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Fatalf("Failed to chmod script: %v", err)
	}
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
