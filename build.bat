@echo off
echo ============================================================
echo   Tiny11 Builder - Build Script (Miku Edition)
echo   Unified Version - Single Executable
echo ============================================================
echo.

echo [1/4] Checking Go environment...
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go compiler not found
    echo Please install Go: https://golang.org/dl/
    pause
    exit /b 1
)
go version
echo.

echo [2/4] Downloading dependencies...
go mod tidy
if errorlevel 1 (
    echo [ERROR] Failed to download dependencies
    pause
    exit /b 1
)
echo [OK] Dependencies ready
echo.

echo [3/4] Creating output directory...
if not exist "bin" mkdir bin
echo [OK] Output directory: bin\
echo.

echo [4/4] Building unified version...
echo.

echo   Building tiny11builder.exe (unified)...
go build -ldflags="-s -w" -o bin/tiny11builder.exe ./cmd/tiny11builder
if errorlevel 1 (
    echo [ERROR] Build failed
    pause
    exit /b 1
)
echo   [OK] Done: bin\tiny11builder.exe

echo.
echo ============================================================
echo   Build Successful!
echo ============================================================
echo.
echo Executable file:
dir /B bin\*.exe
echo.
echo File size:
for %%F in (bin\*.exe) do (
    echo   %%~nxF : %%~zF bytes
)
echo.
echo Usage:
echo   Interactive: bin\tiny11builder.exe
echo   Command line: bin\tiny11builder.exe -iso E -mode standard
echo                 bin\tiny11builder.exe -iso E -mode core
echo.
pause