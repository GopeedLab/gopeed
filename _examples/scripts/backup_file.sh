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

# Check if file exists
if [ ! -f "$GOPEED_FILE_PATH" ]; then
    echo "Error: File not found at $GOPEED_FILE_PATH"
    exit 1
fi

# Copy to each backup location
for DIR in "${BACKUP_DIRS[@]}"; do
    # Create directory if it doesn't exist
    if [ ! -d "$DIR" ]; then
        mkdir -p "$DIR"
    fi
    
    echo "Copying file to $DIR/"
    cp "$GOPEED_FILE_PATH" "$DIR/"
    
    if [ $? -eq 0 ]; then
        echo "File copied successfully to $DIR/$GOPEED_FILE_NAME"
    else
        echo "Warning: Failed to copy file to $DIR"
    fi
done

echo "Backup complete"
exit 0
