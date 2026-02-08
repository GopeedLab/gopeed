package download

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// ScriptEvent represents the type of script event
type ScriptEvent string

const (
	ScriptEventDownloadDone  ScriptEvent = "DOWNLOAD_DONE"
	ScriptEventDownloadError ScriptEvent = "DOWNLOAD_ERROR"
)

// ScriptData is the internal data structure for passing script information
type ScriptData struct {
	Event   ScriptEvent
	Time    int64 // Unix timestamp in milliseconds
	Payload *ScriptPayload
}

// ScriptPayload contains the task data
type ScriptPayload struct {
	Task *Task
}

// getScriptPaths extracts script paths from config
func (d *Downloader) getScriptPaths() []string {
	cfg := d.cfg.DownloaderStoreConfig
	if cfg == nil {
		return nil
	}

	// Check new script config
	if cfg.Script != nil && cfg.Script.Enable && len(cfg.Script.Paths) > 0 {
		paths := make([]string, 0, len(cfg.Script.Paths))
		for _, path := range cfg.Script.Paths {
			if path != "" {
				paths = append(paths, path)
			}
		}
		if len(paths) > 0 {
			return paths
		}
	}

	return nil
}

// executeScriptAtPath executes a single script with the given data
// Returns any error that occurred during execution
func (d *Downloader) executeScriptAtPath(scriptPath string, data *ScriptData) error {
	if scriptPath == "" {
		return fmt.Errorf("script path is empty")
	}

	// Check if script file exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script file does not exist: %s", scriptPath)
	}

	// Determine the script interpreter based on file extension
	var cmd *exec.Cmd
	ext := filepath.Ext(scriptPath)
	
	switch ext {
	case ".sh", ".bash":
		cmd = exec.Command("bash", scriptPath)
	case ".py":
		cmd = exec.Command("python3", scriptPath)
	case ".js":
		cmd = exec.Command("node", scriptPath)
	case ".bat", ".cmd":
		// Windows batch files
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", scriptPath)
		} else {
			// Batch files are Windows-specific
			return fmt.Errorf("batch files (.bat/.cmd) are only supported on Windows")
		}
	case ".ps1":
		// PowerShell scripts
		if runtime.GOOS == "windows" {
			cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
		} else {
			// Try pwsh (PowerShell Core) on non-Windows systems
			cmd = exec.Command("pwsh", "-File", scriptPath)
		}
	case "":
		// No extension, try to execute directly (assumes shebang or executable)
		cmd = exec.Command(scriptPath)
	default:
		// Unknown extension, try to execute directly
		cmd = exec.Command(scriptPath)
	}

	// Set environment variables with task information
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOPEED_EVENT=%s", data.Event),
		fmt.Sprintf("GOPEED_TASK_ID=%s", data.Payload.Task.ID),
		fmt.Sprintf("GOPEED_TASK_NAME=%s", data.Payload.Task.Name()),
		fmt.Sprintf("GOPEED_TASK_STATUS=%s", data.Payload.Task.Status),
	)

	// Add task path using the same logic as task deletion
	if data.Payload.Task.Meta != nil && data.Payload.Task.Meta.Res != nil {
		var taskPath string
		if data.Payload.Task.Meta.Res.Name != "" {
			// Multi-file task (folder)
			taskPath = data.Payload.Task.Meta.FolderPath()
		} else {
			// Single file task
			taskPath = data.Payload.Task.Meta.SingleFilepath()
		}
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("GOPEED_TASK_PATH=%s", taskPath),
		)
	}

	// Start and wait for the command to complete (no timeout)
	return cmd.Run()
}

// triggerScripts executes all configured scripts
func (d *Downloader) triggerScripts(event ScriptEvent, task *Task, err error) {
	paths := d.getScriptPaths()
	if len(paths) == 0 {
		return
	}

	data := &ScriptData{
		Event: event,
		Time:  time.Now().UnixMilli(),
		Payload: &ScriptPayload{
			Task: task.clone(),
		},
	}

	go d.executeScripts(paths, data)
}

func (d *Downloader) executeScripts(paths []string, data *ScriptData) {
	for _, path := range paths {
		if path == "" {
			continue
		}
		go func(scriptPath string) {
			err := d.executeScriptAtPath(scriptPath, data)
			if err != nil {
				d.Logger.Warn().Err(err).Str("path", scriptPath).Msg("script: failed to execute")
				return
			}
			d.Logger.Debug().Str("path", scriptPath).Msg("script: executed successfully")
		}(path)
	}
}
