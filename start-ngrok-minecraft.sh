#!/bin/bash

echo "🎮 Starting Minecraft Server with Ngrok..."
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# Check if Docker is running
if ! docker version &> /dev/null; then
    echo "❌ Docker is not running or not installed"
    echo "Please start Docker"
    exit 1
fi

# Run the deployment tool
go run ngrok-minecraft.go
