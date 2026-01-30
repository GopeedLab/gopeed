#!/bin/bash

# Gopeed Script Example: Move downloaded file to another directory
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
if [ "$GOPEED_EVENT" != "DOWNLOAD_DONE" ]; then
    echo "Event is not DOWNLOAD_DONE, skipping"
    exit 0
fi

# Configuration: Set your target directory here
TARGET_DIR="/path/to/target/directory"

# Create target directory if it doesn't exist
if [ ! -d "$TARGET_DIR" ]; then
    mkdir -p "$TARGET_DIR"
fi

# Check if file exists
if [ ! -f "$GOPEED_FILE_PATH" ]; then
    echo "Error: File not found at $GOPEED_FILE_PATH"
    exit 1
fi

# Move the file
echo "Moving file from $GOPEED_FILE_PATH to $TARGET_DIR/"
mv "$GOPEED_FILE_PATH" "$TARGET_DIR/"

if [ $? -eq 0 ]; then
    echo "File moved successfully to $TARGET_DIR/$GOPEED_FILE_NAME"
else
    echo "Error: Failed to move file"
    exit 1
fi

exit 0
