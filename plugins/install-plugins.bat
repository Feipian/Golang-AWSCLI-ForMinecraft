@echo off
REM Plugin Installation Script for PaperMC (Windows)
REM This script downloads and installs popular Minecraft plugins

setlocal enabledelayedexpansion

REM Plugin URLs and information
set "PLUGINS[WorldEdit]=https://dev.bukkit.org/projects/worldedit/files/latest"
set "PLUGINS[EssentialsX]=https://github.com/EssentialsX/Essentials/releases/latest/download/EssentialsX-2.20.1.jar"
set "PLUGINS[LuckPerms]=https://github.com/lucko/LuckPerms/releases/latest/download/LuckPerms-Bukkit-5.4.101.jar"
set "PLUGINS[Vault]=https://github.com/MilkBowl/Vault/releases/latest/download/Vault-1.7.3.jar"
set "PLUGINS[WorldGuard]=https://dev.bukkit.org/projects/worldguard/files/latest"
set "PLUGINS[Dynmap]=https://github.com/webbukkit/dynmap/releases/latest/download/dynmap-3.4.2-bukkit.jar"
set "PLUGINS[GriefPrevention]=https://github.com/TechFortress/GriefPrevention/releases/latest/download/GriefPrevention-16.18.jar"
set "PLUGINS[CoreProtect]=https://github.com/PlayPro/CoreProtect/releases/latest/download/CoreProtect-22.2.jar"

REM Function to download a plugin
:download_plugin
set "plugin_name=%~1"
set "plugin_url=%~2"
set "plugin_file=plugins\%plugin_name%.jar"

if exist "%plugin_file%" (
    echo [WARNING] %plugin_name% is already installed. Skipping...
    goto :eof
)

echo [INFO] Downloading %plugin_name%...

REM Create plugins directory if it doesn't exist
if not exist "plugins" mkdir plugins

REM Download the plugin using PowerShell
powershell -Command "Invoke-WebRequest -Uri '%plugin_url%' -OutFile '%plugin_file%'"

if exist "%plugin_file%" (
    echo [INFO] %plugin_name% downloaded successfully!
) else (
    echo [ERROR] Failed to download %plugin_name%
    exit /b 1
)
goto :eof

REM Function to install a specific plugin
:install_plugin
set "plugin_name=%~1"

REM Check if plugin exists in our list
set "plugin_found=false"
for /f "tokens=1,2 delims==" %%a in ('set PLUGINS[') do (
    if "%%a"=="PLUGINS[%plugin_name%]" (
        set "plugin_found=true"
        call :download_plugin "%plugin_name%" "%%b"
        goto :eof
    )
)

if "%plugin_found%"=="false" (
    echo [ERROR] Plugin '%plugin_name%' not found in the plugin list
    echo [INFO] Available plugins: WorldEdit, EssentialsX, LuckPerms, Vault, WorldGuard, Dynmap, GriefPrevention, CoreProtect
    exit /b 1
)
goto :eof

REM Function to install all plugins
:install_all
echo [INFO] Installing all available plugins...

call :install_plugin "WorldEdit"
call :install_plugin "EssentialsX"
call :install_plugin "LuckPerms"
call :install_plugin "Vault"
call :install_plugin "WorldGuard"
call :install_plugin "Dynmap"
call :install_plugin "GriefPrevention"
call :install_plugin "CoreProtect"

echo [INFO] All plugins installed successfully!
goto :eof

REM Function to list available plugins
:list_plugins
echo [INFO] Available plugins:
echo   - WorldEdit
echo   - EssentialsX
echo   - LuckPerms
echo   - Vault
echo   - WorldGuard
echo   - Dynmap
echo   - GriefPrevention
echo   - CoreProtect
goto :eof

REM Function to show help
:show_help
echo PaperMC Plugin Installation Script (Windows)
echo.
echo Usage: %0 [COMMAND] [PLUGIN_NAME]
echo.
echo Commands:
echo   install ^<plugin^>  Install a specific plugin
echo   install-all       Install all available plugins
echo   list              List all available plugins
echo   help              Show this help message
echo.
echo Examples:
echo   %0 install WorldEdit
echo   %0 install-all
echo   %0 list
goto :eof

REM Main script logic
if "%1"=="install" (
    if "%2"=="" (
        echo [ERROR] Please specify a plugin name
        call :show_help
        exit /b 1
    )
    call :install_plugin "%2"
) else if "%1"=="install-all" (
    call :install_all
) else if "%1"=="list" (
    call :list_plugins
) else if "%1"=="help" (
    call :show_help
) else if "%1"=="--help" (
    call :show_help
) else if "%1"=="-h" (
    call :show_help
) else if "%1"=="" (
    call :show_help
) else (
    echo [ERROR] Unknown command: %1
    call :show_help
    exit /b 1
)
