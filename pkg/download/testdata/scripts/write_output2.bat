@echo off
setlocal

if "%GOPEED_TEST_OUTPUT_FILE_2%"=="" exit /b 2
echo Script 2 > "%GOPEED_TEST_OUTPUT_FILE_2%"
