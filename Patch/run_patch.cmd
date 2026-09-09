@echo off
setlocal EnableExtensions
cd /d "%~dp0"
title Bedrock Nexus Patch

rem The private payload script is dropped next to this file by the private
rem release process and is never committed. Its name is resolved at runtime
rem (BN_PATCH_PAYLOAD wins, else the single *.ps1 beside this file) so the
rem public tree carries no payload reference.
if defined BN_PATCH_PAYLOAD (
    set "INSTALL_SCRIPT=%~dp0%BN_PATCH_PAYLOAD%"
) else (
    set "INSTALL_SCRIPT="
    for %%F in ("%~dp0*.ps1") do if not defined INSTALL_SCRIPT set "INSTALL_SCRIPT=%%F"
)
if not defined INSTALL_SCRIPT (
    echo ERROR: private patch payload was not found beside this file.
    pause
    exit /b 1
)

where powershell.exe >nul 2>&1 || (
    echo ERROR: Windows PowerShell is not available.
    pause
    exit /b 1
)

rem Re-run as Administrator when the patch needs protected folders.
fltmc >nul 2>&1
if errorlevel 1 (
    echo Requesting Administrator permission...
    powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -Verb RunAs"
    exit /b 0
)

rem INSTALL_SCRIPT was resolved in the block above.
set "GDK_DIR=%~dp0GDK"
if not exist "%GDK_DIR%" mkdir "%GDK_DIR%"

rem Remove the download block without changing the machine execution policy.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "Unblock-File -LiteralPath $env:INSTALL_SCRIPT -ErrorAction SilentlyContinue"

if not exist "%GDK_DIR%\winmm.dll" (
    if exist "%SystemRoot%\System32\winmm.dll" copy /y "%SystemRoot%\System32\winmm.dll" "%GDK_DIR%\winmm.dll" >nul
)

rem Give a useful message before the installer reports missing game content.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$p = Get-AppxPackage -Name Microsoft.MinecraftUWP,Microsoft.MinecraftWindowsBeta -ErrorAction SilentlyContinue; if (-not $p) { $p = Get-AppxPackage -ErrorAction SilentlyContinue | Where-Object { $_.Name -like '*Minecraft*' }; }; if (-not $p) { Write-Host 'Minecraft for Windows was not found for this Windows user.' -ForegroundColor Yellow; Write-Host 'Install it from Xbox App or Microsoft Store, launch it once, then run this file again.' -ForegroundColor Yellow; exit 2 }"
if errorlevel 2 (
    echo.
    pause
    exit /b 2
)

echo Starting patch installer...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%INSTALL_SCRIPT%"
set "exitCode=%errorlevel%"

if not "%exitCode%"=="0" (
    echo.
    echo The patch installer exited with code %exitCode%.
    pause
)

exit /b %exitCode%
