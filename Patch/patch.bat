@echo off
title Minecraft Full Unlock - System DLL Replacer

:: فعال‌سازی کدهای رنگی ANSI در محیط خط فرمان
for /f "tokens=1,2 delims=#" %%a in ('"prompt #$H#$E# & echo on & for %%b in (1) do rem"') do set "ESC=%%b"
set "Cyan=%ESC%[96m"
set "Green=%ESC%[92m"
set "Red=%ESC%[91m"
set "Yellow=%ESC%[93m"
set "White=%ESC%[97m"
set "Reset=%ESC%[0m"

:: تغییر مسیر به پوشه فعلی فایل
cd /d "%~dp0"

:: بررسی دسترسی ادمین
echo %Yellow%Checking for Administrator privileges...%Reset%
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo %Red%[!] Requesting Administrator privileges...%Reset%
    powershell -Command "Start-Process cmd -ArgumentList '/c \"%~f0\"' -Verb RunAs"
    exit /b
)

:main
cls
echo %Cyan%=====================================================================%Reset%
echo  MINECRAFT FULL UNLOCK - SYSTEM DLL REPLACER
echo  Developed by: Aster
echo %Cyan%=====================================================================%Reset%
echo.
echo [%Green%OK%Reset%] Administrator access confirmed.

:: تعریف مسیرها
set "Source32=System32\Windows.ApplicationModel.Store.dll"
set "SourceWow=SysWOW64\Windows.ApplicationModel.Store.dll"
set "Target32=%SystemRoot%\System32\Windows.ApplicationModel.Store.dll"
set "TargetWow=%SystemRoot%\SysWOW64\Windows.ApplicationModel.Store.dll"
set "Backup32=%SystemRoot%\System32\Windows.ApplicationModel.Store.dll.bak"
set "BackupWow=%SystemRoot%\SysWOW64\Windows.ApplicationModel.Store.dll.bak"

:: بررسی فایل‌های سورس
echo.
echo %White%[1/6] Checking source files...%Reset%
if not exist "%Source32%" (
    echo %Red%[X] Source file not found: %Source32%%Reset%
    goto error_exit
)
if not exist "%SourceWow%" (
    echo %Red%[X] Source file not found: %SourceWow%%Reset%
    goto error_exit
)
echo [%Green%OK%Reset%] Source files found.

:: گرفتن مالکیت فایل‌ها
echo.
echo %White%[2/6] Taking ownership of System32 file...%Reset%
if exist "%Target32%" (
    takeown /F "%Target32%" /A >nul 2>&1
    icacls "%Target32%" /grant Administrators:F >nul 2>&1
)

echo %White%[3/6] Taking ownership of SysWOW64 file...%Reset%
if exist "%TargetWow%" (
    takeown /F "%TargetWow%" /A >nul 2>&1
    icacls "%TargetWow%" /grant Administrators:F >nul 2>&1
)
echo [%Green%OK%Reset%] Ownership and permissions updated.

:: بکاپ‌گیری
echo.
echo %White%[4/6] Creating backup of original files...%Reset%
if exist "%Target32%" (
    copy /y "%Target32%" "%Backup32%" >nul
    echo [%Green%OK%Reset%] Backup created: %Backup32%
)
if exist "%TargetWow%" (
    copy /y "%TargetWow%" "%BackupWow%" >nul
    echo [%Green%OK%Reset%] Backup created: %BackupWow%
)

:: جایگزینی فایل‌ها
echo.
echo %White%[5/6] Replacing system files...%Reset%
copy /y "%Source32%" "%Target32%" >nul
echo [%Green%OK%Reset%] System32 file replaced.

copy /y "%SourceWow%" "%TargetWow%" >nul
echo [%Green%OK%Reset%] SysWOW64 file replaced.

:: پایان کار
echo.
echo %Cyan%=====================================================================%Reset%
echo %Green%Operation Completed Successfully!%Reset%
echo %Cyan%=====================================================================%Reset%
echo.
echo %Yellow%Backup files location:%Reset%
echo  -%White% %Backup32%%Reset%
echo  -%White% %BackupWow%%Reset%
echo.
echo %Red%Please Restart your computer now to apply changes.%Reset%
echo.
exit /b

:error_exit
echo.
echo %Red%[X] Operation failed. Please check the files and try again.%Reset%
echo.
exit /b