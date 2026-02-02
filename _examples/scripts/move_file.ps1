# Gopeed Script Example: Move downloaded file to another directory (PowerShell)
# This script demonstrates how to automatically move downloaded files
# to a different location after download completes.

# Environment variables provided by Gopeed:
# - GOPEED_EVENT: Event type (DOWNLOAD_DONE or DOWNLOAD_ERROR)
# - GOPEED_TASK_ID: Task ID
# - GOPEED_TASK_NAME: Task name
# - GOPEED_TASK_STATUS: Task status
# - GOPEED_DOWNLOAD_DIR: Download directory
# - GOPEED_FILE_NAME: Downloaded file name
# - GOPEED_FILE_PATH: Full path to downloaded file

# Exit if not a download done event
if ($env:GOPEED_EVENT -ne "DOWNLOAD_DONE") {
    Write-Host "Event is not DOWNLOAD_DONE, skipping"
    exit 0
}

# Configuration: Set your target directory here
$TARGET_DIR = "D:\Downloads\Archive"

# Create target directory if it doesn't exist
if (-not (Test-Path $TARGET_DIR)) {
    New-Item -Path $TARGET_DIR -ItemType Directory -Force | Out-Null
    Write-Host "Created target directory: $TARGET_DIR"
}

# Check if file exists
if (-not (Test-Path $env:GOPEED_FILE_PATH)) {
    Write-Host "Error: File not found at $env:GOPEED_FILE_PATH"
    exit 1
}

# Move the file
try {
    Write-Host "Moving file from $env:GOPEED_FILE_PATH to $TARGET_DIR\"
    Move-Item -Path $env:GOPEED_FILE_PATH -Destination $TARGET_DIR -Force
    Write-Host "File moved successfully to $TARGET_DIR\$env:GOPEED_FILE_NAME"
    exit 0
}
catch {
    Write-Host "Error: Failed to move file - $_"
    exit 1
}
