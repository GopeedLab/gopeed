# Gopeed Script Example: Send notification after download (PowerShell)
# This script demonstrates how to send notifications when a download completes.

# Exit if not a download done event
if ($env:GOPEED_EVENT -ne "DOWNLOAD_DONE") {
    Write-Host "Event is not DOWNLOAD_DONE, skipping"
    exit 0
}

# Configuration: Set your notification webhook URL here
$WEBHOOK_URL = "https://example.com/webhook"

# Prepare notification data
$notificationData = @{
    event = $env:GOPEED_EVENT
    task_name = $env:GOPEED_TASK_NAME
    task_path = $env:GOPEED_TASK_PATH
    status = $env:GOPEED_TASK_STATUS
    message = "Download '$env:GOPEED_TASK_NAME' completed successfully"
} | ConvertTo-Json

Write-Host "Processing download completion for: $env:GOPEED_TASK_NAME"
Write-Host "Task path: $env:GOPEED_TASK_PATH"

# Send notification via webhook
try {
    $response = Invoke-RestMethod -Uri $WEBHOOK_URL -Method Post -Body $notificationData -ContentType "application/json" -TimeoutSec 10
    Write-Host "Notification sent successfully"
}
catch {
    Write-Host "Error sending notification: $_"
}

# You can also read the full task data from stdin
try {
    $input = [System.Console]::In.ReadToEnd()
    if ($input) {
        $taskData = $input | ConvertFrom-Json
        Write-Host "Full task data available: $($taskData.event)"
    }
}
catch {
    # Ignore parsing errors
}

exit 0
