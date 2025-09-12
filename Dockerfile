# Use OpenJDK 21 as base image (required for PaperMC)
FROM openjdk:21-jdk-slim

# Set working directory
WORKDIR /minecraft

RUN apt-get update && apt-get install -y procps && rm -rf /var/lib/apt/lists/*

# Install necessary packages
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Create minecraft user for security
RUN useradd -r -s /bin/false minecraft && \
    chown -R minecraft:minecraft /minecraft

# Set environment variables
ENV MINECRAFT_VERSION=1.21.4
ENV PAPER_BUILD=latest
ENV MEMORY_SIZE=2G
ENV EULA=true

# Create directories
RUN mkdir -p /minecraft/plugins /minecraft/worlds /minecraft/logs

# Copy startup script
COPY start.sh /minecraft/start.sh
RUN chmod +x /minecraft/start.sh

# Copy server properties template
COPY server.properties.template /minecraft/server.properties.template

# Run as root to avoid permission issues with mounted volumes
USER root

# Expose Minecraft port
EXPOSE 25565

# Expose RCON port (optional)
EXPOSE 25575

# Set volume for persistent data
VOLUME ["/minecraft/worlds", "/minecraft/plugins", "/minecraft/logs"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:25565 || exit 1

# Start the server
CMD ["/minecraft/start.sh"]
