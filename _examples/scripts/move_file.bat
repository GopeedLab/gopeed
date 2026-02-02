@echo off
REM Gopeed Script Example: Move downloaded file to another directory (Windows)
REM This script demonstrates how to automatically move downloaded files
REM to a different location after download completes.

REM Environment variables provided by Gopeed:
REM - GOPEED_EVENT: Event type (DOWNLOAD_DONE or DOWNLOAD_ERROR)
REM - GOPEED_TASK_ID: Task ID
REM - GOPEED_TASK_NAME: Task name
REM - GOPEED_TASK_STATUS: Task status
REM - GOPEED_DOWNLOAD_DIR: Download directory
REM - GOPEED_FILE_NAME: Downloaded file name
REM - GOPEED_FILE_PATH: Full path to downloaded file

REM Exit if not a download done event
if not "%GOPEED_EVENT%"=="DOWNLOAD_DONE" (
    echo Event is not DOWNLOAD_DONE, skipping
    exit /b 0
)

REM Configuration: Set your target directory here
set TARGET_DIR=D:\Downloads\Archive

REM Create target directory if it doesn't exist
if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%"

REM Check if file exists
if not exist "%GOPEED_FILE_PATH%" (
    echo Error: File not found at %GOPEED_FILE_PATH%
    exit /b 1
)

REM Move the file
echo Moving file from %GOPEED_FILE_PATH% to %TARGET_DIR%\
move "%GOPEED_FILE_PATH%" "%TARGET_DIR%\"

if %ERRORLEVEL% equ 0 (
    echo File moved successfully to %TARGET_DIR%\%GOPEED_FILE_NAME%
) else (
    echo Error: Failed to move file
    exit /b 1
)

exit /b 0
