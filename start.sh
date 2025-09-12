#!/bin/bash

# PaperMC Server Startup Script
set -e

echo "Starting PaperMC Server..."

# Set default values
MINECRAFT_VERSION=${MINECRAFT_VERSION:-"1.21.4"}
PAPER_BUILD=${PAPER_BUILD:-"latest"}
MEMORY_SIZE=${MEMORY_SIZE:-"2G"}
EULA=${EULA:-"false"}

# Download PaperMC if not exists
if [ ! -f "paper-${MINECRAFT_VERSION}-${PAPER_BUILD}.jar" ]; then
    echo "Downloading PaperMC ${MINECRAFT_VERSION}-${PAPER_BUILD} from fixed URL..."
    curl -L -o "paper-${MINECRAFT_VERSION}-${PAPER_BUILD}.jar" \
        "https://fill-data.papermc.io/v1/objects/5ee4f542f628a14c644410b08c94ea42e772ef4d29fe92973636b6813d4eaffc/paper-1.21.4-232.jar"
    echo "PaperMC downloaded successfully!"
fi

# Create eula.txt if it doesn't exist
if [ ! -f "eula.txt" ]; then
    echo "Creating eula.txt..."
    echo "eula=${EULA}" > eula.txt
fi

# Copy server.properties template if server.properties doesn't exist
if [ ! -f "server.properties" ]; then
    echo "Creating server.properties from template..."
    cp server.properties.template server.properties
fi

# Ensure required directories exist
mkdir -p plugins logs worlds

# Download and install plugins if PLUGINS environment variable is set
if [ ! -z "$PLUGINS" ]; then
    echo "Installing plugins..."
    IFS=',' read -ra PLUGIN_ARRAY <<< "$PLUGINS"
    for plugin in "${PLUGIN_ARRAY[@]}"; do
        plugin=$(echo $plugin | xargs) # trim whitespace
        if [ ! -z "$plugin" ]; then
            echo "Installing plugin: $plugin"
            # This is a placeholder - you can implement plugin download logic here
            # For now, we'll just create a placeholder file
            touch "plugins/${plugin}.jar"
        fi
    done
fi

# Set JVM arguments (simplified for reliable startup)
JVM_ARGS="-Xms${MEMORY_SIZE} -Xmx${MEMORY_SIZE} -XX:+UseG1GC"

# Start the server
echo "Starting PaperMC server with ${MEMORY_SIZE} memory..."
echo "JVM Args: ${JVM_ARGS}"
echo "Server JAR: paper-${MINECRAFT_VERSION}-${PAPER_BUILD}.jar"

# If minecraft user exists, fix permissions then drop privileges to run the server
if id minecraft >/dev/null 2>&1; then
    echo "Adjusting ownership to minecraft..."
    chown -R minecraft:minecraft /minecraft || true
    echo "Launching server as minecraft user..."
    exec su -s /bin/sh -c "exec java ${JVM_ARGS} -jar 'paper-${MINECRAFT_VERSION}-${PAPER_BUILD}.jar' nogui" minecraft
fi

# Fallback: run as current user
exec java ${JVM_ARGS} -jar "paper-${MINECRAFT_VERSION}-${PAPER_BUILD}.jar" nogui
