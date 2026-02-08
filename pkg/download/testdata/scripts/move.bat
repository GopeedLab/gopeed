@echo off
setlocal

if "%GOPEED_TEST_DEST_DIR%"=="" exit /b 2
set "SRC=%GOPEED_TASK_PATH:/=\%"
set "DEST=%GOPEED_TEST_DEST_DIR%"
if not exist "%DEST%" mkdir "%DEST%"
move /Y "%SRC%" "%DEST%\" >nul
