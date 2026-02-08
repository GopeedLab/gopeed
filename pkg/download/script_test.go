package download

import (
	"os"
	"path/filepath"
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
