#!/bin/bash

# PaperMC Docker Management Script
# This script helps you manage your PaperMC Docker container

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Function to create necessary directories
create_directories() {
    print_status "Creating necessary directories..."
    mkdir -p data plugins logs
    print_status "Directories created successfully!"
}

# Function to build the Docker image
build_image() {
    print_header "Building PaperMC Docker Image"
    check_docker
    create_directories
    
    print_status "Building Docker image..."
    docker build -t papermc-server .
    print_status "Docker image built successfully!"
}

# Function to start the server
start_server() {
    print_header "Starting PaperMC Server"
    check_docker
    
    if [ ! -f "docker-compose.yml" ]; then
        print_error "docker-compose.yml not found!"
        exit 1
    fi
    
    print_status "Starting PaperMC server with Docker Compose..."
    docker-compose up -d
    print_status "Server started successfully!"
    print_status "Server is running on port 25565"
    print_status "Use 'docker-compose logs -f' to view server logs"
}

# Function to stop the server
stop_server() {
    print_header "Stopping PaperMC Server"
    check_docker
    
    print_status "Stopping PaperMC server..."
    docker-compose down
    print_status "Server stopped successfully!"
}

# Function to restart the server
restart_server() {
    print_header "Restarting PaperMC Server"
    stop_server
    start_server
}

# Function to view logs
view_logs() {
    print_header "PaperMC Server Logs"
    check_docker
    
    if [ "$1" = "-f" ]; then
        print_status "Following logs (Ctrl+C to exit)..."
        docker-compose logs -f
    else
        docker-compose logs --tail=50
    fi
}

# Function to backup server data
backup_data() {
    print_header "Backing Up Server Data"
    
    BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    print_status "Creating backup in $BACKUP_DIR..."
    
    if [ -d "data" ]; then
        cp -r data "$BACKUP_DIR/"
        print_status "World data backed up"
    fi
    
    if [ -d "plugins" ]; then
        cp -r plugins "$BACKUP_DIR/"
        print_status "Plugins backed up"
    fi
    
    if [ -f "server.properties" ]; then
        cp server.properties "$BACKUP_DIR/"
        print_status "Server properties backed up"
    fi
    
    print_status "Backup completed: $BACKUP_DIR"
}

# Function to show server status
show_status() {
    print_header "PaperMC Server Status"
    check_docker
    
    if docker-compose ps | grep -q "Up"; then
        print_status "Server is running"
        docker-compose ps
    else
        print_warning "Server is not running"
    fi
}

# Function to show help
show_help() {
    echo "PaperMC Docker Management Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  build     Build the Docker image"
    echo "  start     Start the PaperMC server"
    echo "  stop      Stop the PaperMC server"
    echo "  restart   Restart the PaperMC server"
    echo "  logs      View server logs (add -f to follow)"
    echo "  status    Show server status"
    echo "  backup    Backup server data"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 build"
    echo "  $0 start"
    echo "  $0 logs -f"
    echo "  $0 backup"
}

# Main script logic
case "${1:-help}" in
    build)
        build_image
        ;;
    start)
        start_server
        ;;
    stop)
        stop_server
        ;;
    restart)
        restart_server
        ;;
    logs)
        view_logs "$2"
        ;;
    status)
        show_status
        ;;
    backup)
        backup_data
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
