@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"

echo ========================================
echo          NetMap Windows Build
echo ========================================

call :build netmap-windows-amd64 amd64
if errorlevel 1 goto :error

call :build netmap-windows-arm64 arm64
if errorlevel 1 goto :error

call :build netmap-windows-386 386
if errorlevel 1 goto :error

echo.
echo ========================================
echo All builds succeeded!
echo Output: dist\
echo ========================================
exit /b 0


:build
set "PLATFORM=%~1"
set "ARCH=%~2"
set "OUTPUT_DIR=dist\%PLATFORM%"

echo.
echo ----------------------------------------
echo Building %PLATFORM%...
echo ----------------------------------------

if exist "%OUTPUT_DIR%" rmdir /s /q "%OUTPUT_DIR%"
if exist "%OUTPUT_DIR%" exit /b 1

mkdir "%OUTPUT_DIR%"
if errorlevel 1 exit /b 1

set "GOOS=windows"
set "GOARCH=%ARCH%"
set "CGO_ENABLED=0"

go build -trimpath -ldflags="-H windowsgui" -o "%OUTPUT_DIR%\netmap.exe" .\cmd\netmap
if errorlevel 1 exit /b 1

REM 复制配置目录
if exist "configs" (
    xcopy "configs" "%OUTPUT_DIR%\configs\" /E /I /Y /Q >nul
    if errorlevel 1 exit /b 1
) else (
    echo [WARN] configs directory not found, skipped.
)

REM 复制资源目录 assets
if exist "assets" (
    xcopy "assets" "%OUTPUT_DIR%\assets\" /E /I /Y /Q >nul
    if errorlevel 1 exit /b 1
) else (
    echo [WARN] assets directory not found, skipped.
)

echo [OK] %PLATFORM%
exit /b 0


:error
echo.
echo ========================================
echo [ERROR] Build failed. Check the output above.
echo ========================================
exit /b 1
