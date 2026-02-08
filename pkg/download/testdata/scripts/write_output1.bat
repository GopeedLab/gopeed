@echo off
setlocal

if "%GOPEED_TEST_OUTPUT_FILE_1%"=="" exit /b 2
echo Script 1 > "%GOPEED_TEST_OUTPUT_FILE_1%"
