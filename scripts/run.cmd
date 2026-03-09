@echo off
setlocal enabledelayedexpansion
REM Detect architecture and run the appropriate save-transcript binary.
REM Downloads the binary from GitHub Releases on first run if not present.
REM This shim is invoked by the Claude Code PreCompact hook on Windows.

set "SCRIPT_DIR=%~dp0"
set "PLUGIN_ROOT=%SCRIPT_DIR%.."
set "BIN_DIR=%PLUGIN_ROOT%\bin"
set "REPO=IDisposable/claude-transcript-plugin"

REM Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else (
    set "ARCH=amd64"
)

set "FILENAME=save-transcript-windows-%ARCH%.exe"
set "BINARY=%BIN_DIR%\%FILENAME%"

REM Bootstrap: download binary from GitHub Release if not present
if not exist "%BINARY%" (
    REM Parse version from plugin.json
    set "VERSION="
    for /f "tokens=2 delims=:" %%a in ('findstr /c:"\"version\"" "%PLUGIN_ROOT%\.claude-plugin\plugin.json"') do (
        set "RAW=%%a"
    )
    if defined RAW (
        REM Strip quotes, spaces, and commas
        set "VERSION=!RAW: =!"
        set "VERSION=!VERSION:"=!"
        set "VERSION=!VERSION:,=!"
    )

    if not defined VERSION (
        echo transcript-saver: cannot determine version from plugin.json 1>&2
        exit /b 0
    )

    set "URL=https://github.com/%REPO%/releases/download/v!VERSION!/%FILENAME%"

    echo transcript-saver: downloading binary for windows/%ARCH% ^(v!VERSION!^)... 1>&2
    if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

    REM Try curl first (available on Windows 10+), then PowerShell as fallback
    where curl >nul 2>&1
    if !errorlevel! equ 0 (
        curl -fsSL -o "%BINARY%" "!URL!"
        if !errorlevel! neq 0 (
            echo transcript-saver: download failed from !URL! 1>&2
            if exist "%BINARY%" del "%BINARY%"
            exit /b 0
        )
    ) else (
        powershell -NoProfile -Command "try { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri '!URL!' -OutFile '%BINARY%' -UseBasicParsing } catch { Write-Error $_.Exception.Message; exit 1 }"
        if !errorlevel! neq 0 (
            echo transcript-saver: download failed from !URL! 1>&2
            if exist "%BINARY%" del "%BINARY%"
            exit /b 0
        )
    )
)

"%BINARY%" %*
