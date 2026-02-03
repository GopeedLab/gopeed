# Gopeed Script Examples

This directory contains example scripts that demonstrate how to use Gopeed's script execution feature to automatically process files after download completion.

## Overview

Gopeed can execute scripts automatically when downloads complete successfully. This is useful for:
- Moving downloaded files to different locations (e.g., from SSD to HDD)
- Backing up files to multiple locations
- Sending notifications
- Processing files (extracting archives, converting formats, etc.)
- Integration with other tools and workflows

## Environment Variables

When Gopeed executes your script, it provides the following environment variables:

- `GOPEED_EVENT`: Event type (currently always `DOWNLOAD_DONE`)
- `GOPEED_TASK_ID`: Unique task identifier
- `GOPEED_TASK_NAME`: Display name of the task
- `GOPEED_TASK_STATUS`: Task status
- `GOPEED_DOWNLOAD_DIR`: Directory where the file was downloaded
- `GOPEED_FILE_NAME`: Name of the downloaded file
- `GOPEED_FILE_PATH`: Full path to the downloaded file

## Task Data via STDIN

In addition to environment variables, the full task data is passed to your script as JSON via STDIN. This includes detailed information about the download task.

Example JSON structure:
```json
{
  "event": "DOWNLOAD_DONE",
  "time": 1706572800000,
  "payload": {
    "task": {
      "id": "abc123",
      "protocol": "http",
      "status": "done",
      "meta": {
        "req": {
          "url": "https://example.com/file.zip"
        },
        "opts": {
          "name": "file.zip",
          "path": "/downloads"
        }
      }
    }
  }
}
```

## Example Scripts

### Unix/Linux/macOS Scripts

#### 1. move_file.sh
Automatically moves downloaded files from the download directory to a target directory. This is useful for moving files from fast SSDs to larger HDDs after download.

**Usage:**
1. Edit the script and set `TARGET_DIR` to your desired location
2. Make the script executable: `chmod +x move_file.sh`
3. Configure Gopeed to use this script

#### 2. backup_file.sh
Copies downloaded files to multiple backup locations.

**Usage:**
1. Edit the script and set the `BACKUP_DIRS` array with your backup locations
2. Make the script executable: `chmod +x backup_file.sh`
3. Configure Gopeed to use this script

#### 3. notify.py
Sends notifications when downloads complete. Can be customized to send emails, push notifications, or webhook calls.

**Requirements:**
- Python 3
- `requests` library: `pip install requests`

**Usage:**
1. Edit the script and set `WEBHOOK_URL` to your notification endpoint
2. Make the script executable: `chmod +x notify.py`
3. Configure Gopeed to use this script

#### 4. process_file.js
Processes downloaded files (e.g., automatically extracts ZIP archives).

**Requirements:**
- Node.js
- `unzip` command-line tool (for ZIP extraction)

**Usage:**
1. Make the script executable: `chmod +x process_file.js`
2. Customize the processing logic as needed
3. Configure Gopeed to use this script

### Windows Scripts

#### 5. move_file.bat
Windows batch script that moves downloaded files to a target directory.

**Usage:**
1. Edit the script and set `TARGET_DIR` to your desired location (e.g., `D:\Downloads\Archive`)
2. Configure Gopeed to use this script

#### 6. move_file.ps1
PowerShell script that moves downloaded files with better error handling.

**Usage:**
1. Edit the script and set `$TARGET_DIR` to your desired location
2. Ensure PowerShell execution policy allows script execution
3. Configure Gopeed to use this script

#### 7. notify.ps1
PowerShell script that sends notifications when downloads complete.

**Usage:**
1. Edit the script and set `$WEBHOOK_URL` to your notification endpoint
2. Configure Gopeed to use this script

## Supported Script Types

Gopeed automatically detects the script type based on the file extension:

| Extension | Interpreter | Platform |
|-----------|-------------|----------|
| `.sh`, `.bash` | bash | Unix/Linux/macOS |
| `.py` | python3 | All platforms |
| `.js` | node | All platforms |
| `.bat`, `.cmd` | cmd.exe | Windows only |
| `.ps1` | PowerShell | Windows (pwsh on other platforms) |
| No extension | Direct execution | All platforms (requires executable bit or shebang) |

## Configuration

To configure Gopeed to use these scripts:

### Via API
```bash
# Update configuration
curl -X PUT http://localhost:9999/api/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "script": {
      "enable": true,
      "paths": [
        "/path/to/move_file.sh",
        "/path/to/notify.py"
      ]
    }
  }'
```

### Via Configuration File
Edit your Gopeed configuration file and add:
```json
{
  "script": {
    "enable": true,
    "paths": [
      "/path/to/move_file.sh",
      "/path/to/notify.py"
    ]
  }
}
```

## Creating Your Own Scripts

### Bash Script Template
```bash
#!/bin/bash

# Exit if not a download done event
if [ "$GOPEED_EVENT" != "DOWNLOAD_DONE" ]; then
    exit 0
fi

# Your custom logic here
echo "Processing: $GOPEED_FILE_NAME"
echo "Location: $GOPEED_FILE_PATH"

# Read full task data from stdin (optional)
# TASK_DATA=$(cat)
# echo "$TASK_DATA" | jq '.payload.task.id'

exit 0
```

### Python Script Template
```python
#!/usr/bin/env python3

import os
import sys
import json

# Get environment variables
event = os.getenv("GOPEED_EVENT")
file_path = os.getenv("GOPEED_FILE_PATH")

# Exit if not a download done event
if event != "DOWNLOAD_DONE":
    sys.exit(0)

# Your custom logic here
print(f"Processing: {file_path}")

# Read full task data from stdin (optional)
try:
    task_data = json.loads(sys.stdin.read())
    print(f"Task ID: {task_data['payload']['task']['id']}")
except:
    pass

sys.exit(0)
```

### Windows Batch Script Template
```batch
@echo off
REM Exit if not a download done event
if not "%GOPEED_EVENT%"=="DOWNLOAD_DONE" exit /b 0

REM Your custom logic here
echo Processing: %GOPEED_FILE_NAME%
echo Location: %GOPEED_FILE_PATH%

exit /b 0
```

### Windows PowerShell Script Template
```powershell
# Exit if not a download done event
if ($env:GOPEED_EVENT -ne "DOWNLOAD_DONE") { exit 0 }

# Your custom logic here
Write-Host "Processing: $env:GOPEED_FILE_NAME"
Write-Host "Location: $env:GOPEED_FILE_PATH"

# Read full task data from stdin (optional)
try {
    $input = [System.Console]::In.ReadToEnd()
    if ($input) {
        $taskData = $input | ConvertFrom-Json
        Write-Host "Task ID: $($taskData.payload.task.id)"
    }
} catch {
    # Ignore parsing errors
}

exit 0
```

## Security Considerations

- Scripts are executed with the same permissions as the Gopeed process
- Always validate and sanitize file paths before processing
- Be cautious with scripts that accept external input
- Only use scripts from trusted sources
- Scripts run without timeout limitation, so ensure they complete properly
- **Windows users**: Be aware of PowerShell execution policies

## Troubleshooting

### Script not executing
- Check that the script file exists and is executable (`chmod +x script.sh` on Unix)
- Verify the script path in Gopeed configuration is correct
- Check Gopeed logs for error messages
- **Windows**: Ensure PowerShell execution policy allows running scripts

### Permission errors
- Ensure Gopeed has permission to execute the script
- Ensure the script has permission to access/modify the target directories
- Check file and directory ownership and permissions
- **Windows**: Run Gopeed with appropriate privileges if accessing protected folders

### Windows-specific issues
- **PowerShell execution policy**: Run `Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser` to allow script execution
- **Batch file errors**: Check that paths with spaces are properly quoted
- **Path format**: Use Windows path format (e.g., `C:\path\to\file` or `C:/path/to/file`)

## License

These examples are provided under the same license as Gopeed.
