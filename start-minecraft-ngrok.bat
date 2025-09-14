@echo off
echo 🎮 Starting Minecraft Server with Ngrok...
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

REM Run the deployment tool
go run ngrok-minecraft.go

pause
