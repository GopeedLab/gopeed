#!/bin/bash

# Gopeed Script Example: Move downloaded file to another directory
# This script demonstrates how to automatically move downloaded files
# to a different location after download completes.

# Environment variables provided by Gopeed:
# - GOPEED_EVENT: Event type (DOWNLOAD_DONE)
# - GOPEED_TASK_ID: Task ID
# - GOPEED_TASK_NAME: Task name
# - GOPEED_TASK_STATUS: Task status
# - GOPEED_TASK_PATH: Full path to downloaded file or folder

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

# Check if file or folder exists
if [ ! -e "$GOPEED_TASK_PATH" ]; then
    echo "Error: Path not found at $GOPEED_TASK_PATH"
    exit 1
fi

# Get the base name of the file or folder
BASENAME=$(basename "$GOPEED_TASK_PATH")

# Move the file or folder
echo "Moving $GOPEED_TASK_PATH to $TARGET_DIR/"
mv "$GOPEED_TASK_PATH" "$TARGET_DIR/"

if [ $? -eq 0 ]; then
    echo "Successfully moved to $TARGET_DIR/$BASENAME"
else
    echo "Error: Failed to move"
    exit 1
fi

exit 0
