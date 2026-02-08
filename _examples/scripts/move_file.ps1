# Gopeed Script Example: Move downloaded file to another directory (PowerShell)
# This script demonstrates how to automatically move downloaded files
# to a different location after download completes.

# Environment variables provided by Gopeed:
# - GOPEED_EVENT: Event type (DOWNLOAD_DONE)
# - GOPEED_TASK_ID: Task ID
# - GOPEED_TASK_NAME: Task name
# - GOPEED_TASK_STATUS: Task status
# - GOPEED_TASK_PATH: Full path to downloaded file or folder

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

# Check if file or folder exists
if (-not (Test-Path $env:GOPEED_TASK_PATH)) {
    Write-Host "Error: Path not found at $env:GOPEED_TASK_PATH"
    exit 1
}

# Move the file or folder
try {
    Write-Host "Moving $env:GOPEED_TASK_PATH to $TARGET_DIR\"
    Move-Item -Path $env:GOPEED_TASK_PATH -Destination $TARGET_DIR -Force
    Write-Host "Successfully moved to $TARGET_DIR\"
    exit 0
}
catch {
    Write-Host "Error: Failed to move - $_"
    exit 1
}
