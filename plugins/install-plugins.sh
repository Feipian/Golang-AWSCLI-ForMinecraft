#!/bin/bash

# Plugin Installation Script for PaperMC
# This script downloads and installs popular Minecraft plugins

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Plugin URLs and information
declare -A PLUGINS=(
    ["WorldEdit"]="https://dev.bukkit.org/projects/worldedit/files/latest"
    ["EssentialsX"]="https://github.com/EssentialsX/Essentials/releases/latest/download/EssentialsX-2.20.1.jar"
    ["LuckPerms"]="https://github.com/lucko/LuckPerms/releases/latest/download/LuckPerms-Bukkit-5.4.101.jar"
    ["Vault"]="https://github.com/MilkBowl/Vault/releases/latest/download/Vault-1.7.3.jar"
    ["WorldGuard"]="https://dev.bukkit.org/projects/worldguard/files/latest"
    ["Dynmap"]="https://github.com/webbukkit/dynmap/releases/latest/download/dynmap-3.4.2-bukkit.jar"
    ["GriefPrevention"]="https://github.com/TechFortress/GriefPrevention/releases/latest/download/GriefPrevention-16.18.jar"
    ["CoreProtect"]="https://github.com/PlayPro/CoreProtect/releases/latest/download/CoreProtect-22.2.jar"
)

# Function to download a plugin
download_plugin() {
    local plugin_name="$1"
    local plugin_url="$2"
    local plugin_file="plugins/${plugin_name}.jar"
    
    if [ -f "$plugin_file" ]; then
        print_warning "$plugin_name is already installed. Skipping..."
        return 0
    fi
    
    print_status "Downloading $plugin_name..."
    
    # Create plugins directory if it doesn't exist
    mkdir -p plugins
    
    # Download the plugin
    if curl -L -o "$plugin_file" "$plugin_url"; then
        print_status "$plugin_name downloaded successfully!"
    else
        print_error "Failed to download $plugin_name"
        return 1
    fi
}

# Function to install a specific plugin
install_plugin() {
    local plugin_name="$1"
    
    if [ -z "${PLUGINS[$plugin_name]}" ]; then
        print_error "Plugin '$plugin_name' not found in the plugin list"
        print_status "Available plugins: ${!PLUGINS[*]}"
        return 1
    fi
    
    download_plugin "$plugin_name" "${PLUGINS[$plugin_name]}"
}

# Function to install all plugins
install_all() {
    print_status "Installing all available plugins..."
    
    for plugin_name in "${!PLUGINS[@]}"; do
        install_plugin "$plugin_name"
    done
    
    print_status "All plugins installed successfully!"
}

# Function to list available plugins
list_plugins() {
    print_status "Available plugins:"
    for plugin_name in "${!PLUGINS[@]}"; do
        echo "  - $plugin_name"
    done
}

# Function to show help
show_help() {
    echo "PaperMC Plugin Installation Script"
    echo ""
    echo "Usage: $0 [COMMAND] [PLUGIN_NAME]"
    echo ""
    echo "Commands:"
    echo "  install <plugin>  Install a specific plugin"
    echo "  install-all       Install all available plugins"
    echo "  list              List all available plugins"
    echo "  help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 install WorldEdit"
    echo "  $0 install-all"
    echo "  $0 list"
}

# Main script logic
case "${1:-help}" in
    install)
        if [ -z "$2" ]; then
            print_error "Please specify a plugin name"
            show_help
            exit 1
        fi
        install_plugin "$2"
        ;;
    install-all)
        install_all
        ;;
    list)
        list_plugins
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
