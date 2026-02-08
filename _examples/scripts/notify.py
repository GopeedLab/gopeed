#!/usr/bin/env python3

"""
Gopeed Script Example: Send notification after download
This script demonstrates how to send notifications (e.g., via webhook or email)
when a download completes.
"""

import os
import sys
import json
import requests

def send_notification(event, task_name, task_path, status):
    """
    Send a notification to a webhook endpoint.
    You can customize this to send emails, push notifications, etc.
    """
    
    # Configuration: Set your notification webhook URL here
    WEBHOOK_URL = "https://example.com/webhook"
    
    # Prepare notification data
    notification_data = {
        "event": event,
        "task_name": task_name,
        "task_path": task_path,
        "status": status,
        "message": f"Download '{task_name}' completed successfully"
    }
    
    try:
        # Send POST request to webhook
        response = requests.post(
            WEBHOOK_URL,
            json=notification_data,
            timeout=10
        )
        
        if response.status_code == 200:
            print(f"Notification sent successfully")
        else:
            print(f"Warning: Webhook returned status {response.status_code}")
            
    except Exception as e:
        print(f"Error sending notification: {e}")

def main():
    # Get environment variables provided by Gopeed
    event = os.getenv("GOPEED_EVENT")
    task_id = os.getenv("GOPEED_TASK_ID")
    task_name = os.getenv("GOPEED_TASK_NAME")
    status = os.getenv("GOPEED_TASK_STATUS")
    task_path = os.getenv("GOPEED_TASK_PATH")
    
    # Only process DOWNLOAD_DONE events
    if event != "DOWNLOAD_DONE":
        print(f"Event is not DOWNLOAD_DONE, skipping")
        return 0
    
    print(f"Processing download completion for: {task_name}")
    print(f"Task path: {task_path}")
    
    # Send notification
    send_notification(event, task_name, task_path, status)
    
    # You can also read the full task data from stdin
    try:
        task_data = json.loads(sys.stdin.read())
        print(f"Full task data available: {task_data.get('event')}")
    except:
        pass
    
    return 0

if __name__ == "__main__":
    sys.exit(main())
