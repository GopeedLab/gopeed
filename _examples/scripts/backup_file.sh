#!/bin/bash

# Gopeed Script Example: Copy downloaded file to multiple locations
# This script demonstrates how to copy downloaded files to multiple
# backup locations after download completes.

# Exit if not a download done event
if [ "$GOPEED_EVENT" != "DOWNLOAD_DONE" ]; then
    echo "Event is not DOWNLOAD_DONE, skipping"
    exit 0
fi

# Configuration: Set your backup directories here
BACKUP_DIRS=(
    "/path/to/backup1"
    "/path/to/backup2"
    "/path/to/backup3"
)

# Check if file or folder exists
if [ ! -e "$GOPEED_TASK_PATH" ]; then
    echo "Error: Path not found at $GOPEED_TASK_PATH"
    exit 1
fi

# Get the base name
BASENAME=$(basename "$GOPEED_TASK_PATH")

# Copy to each backup location
for DIR in "${BACKUP_DIRS[@]}"; do
    # Create directory if it doesn't exist
    if [ ! -d "$DIR" ]; then
        mkdir -p "$DIR"
    fi
    
    echo "Copying to $DIR/"
    cp -r "$GOPEED_TASK_PATH" "$DIR/"
    
    if [ $? -eq 0 ]; then
        echo "Successfully copied to $DIR/$BASENAME"
    else
        echo "Warning: Failed to copy to $DIR"
    fi
done

echo "Backup complete"
exit 0
