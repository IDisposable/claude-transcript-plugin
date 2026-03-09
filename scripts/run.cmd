@echo off
REM Detect architecture and run the appropriate save-transcript binary.
REM This shim is invoked by the Claude Code PreCompact hook on Windows.

set "SCRIPT_DIR=%~dp0"
set "PLUGIN_ROOT=%SCRIPT_DIR%.."
set "BIN_DIR=%PLUGIN_ROOT%\bin"

REM Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else (
    set "ARCH=amd64"
)

set "BINARY=%BIN_DIR%\save-transcript-windows-%ARCH%.exe"

if not exist "%BINARY%" (
    echo transcript-saver: binary not found: %BINARY% 1>&2
    exit /b 0
)

"%BINARY%" %*
