# Minecraft PaperMC - Docker + AWS + CI/CD

## Quick start (Docker)

```bash
# Build and start
docker compose up -d --build

# Follow logs
docker compose logs -f papermc

# Stop
docker compose down
```

Default ports: 25565 (Minecraft), 25575 (RCON disabled by default).

## Configuration

Edit `docker-compose.yml`:
- Environment
  - `MINECRAFT_VERSION` (default: 1.21.4)
  - `PAPER_BUILD` (default: latest)
  - `MEMORY_SIZE` (default: 2G)
  - `EULA` (true/false)
  - Optional server property overrides like `SERVER_NAME`, `MOTD`, etc.
- Volumes (persist data)
  - `minecraft-data:/minecraft/world`
  - `minecraft-plugins:/minecraft/plugins`
  - `minecraft-logs:/minecraft/logs`

To load plugins from your project folder instead of a named volume, change:
```yaml
to:
  volumes:
    - ./plugins:/minecraft/plugins
    - minecraft-data:/minecraft/world
    - minecraft-logs:/minecraft/logs
```
Then put your plugin JARs into `./plugins/` and restart.

## Startup behavior (`start.sh`)
- Ensures `eula.txt` exists
- Creates `server.properties` from `server.properties.template` if missing
- Creates `plugins`, `logs`, `worlds` folders
- Downloads PaperMC JAR (version/build controlled by env)
- Minimal JVM args: `-Xms/-Xmx MEM_SIZE -XX:+UseG1GC`
- Drops privileges to user `minecraft` when available

## CI/CD (GitHub Actions)
One workflow in `.github/workflows/`:
- `dockerhub.yml`
  - Builds and pushes Docker images to Docker Hub
  - Runs on push to `main` and `cicd-pipeline` branches
  - Multi-platform builds with caching
  - Test PR workflow

Required repo secrets for Docker Hub:
- `DOCKER_USERNAME`: Your Docker Hub username
- `DOCKER_PASSWORD`: Your Docker Hub password or access token

## Local Deployment with Ngrok (Go)
Run your Minecraft server locally and expose it via ngrok:

```bash
# Windows
start-ngrok-minecraft.bat

# Linux/Mac
./start-ngrok-minecraft.sh

# Or run directly
go run ngrok-minecraft.go
```

This will:
1. Start your Minecraft Docker container on port 25565
2. Create an ngrok tunnel to expose it to the internet
3. Give you a public address to share with friends

## AWS provisioning (Go)
Provision infrastructure and get a public address:

```powershell
# Build & run
go run upload2AWS.go
```

What it does (idempotent where possible):
- Creates or reuses VPC `minecraft-vpc`
- Internet Gateway `minecraft-igw`
- Subnet `minecraft-subnet`
- Route table `minecraft-rt` with default route
- Security Group `minecraft-sg` (SSH 22 and Minecraft 25565; attempts to restrict to your current public IP when detected)
- Launches/uses an EC2 instance tagged `minecraft-ec2`
- Allocates and associates an Elastic IP (EIP) to the instance
- Prints the public address for Minecraft and SSH

Cleanup to avoid charges:
```powershell
go run cleanup.go
```
Cleanup order (implemented): release EIP → terminate instance (wait) → delete route table associations and default route → delete route tables → delete subnets → delete SG → detach+delete IGW → delete VPC.

## Security tips
- Never commit private keys (.pem). Add to `.gitignore`:
  ```
  *.pem
  *.key
  id_*.pem
  ```
- If a key leaked, rotate it and purge history.
- Prefer a non-root SSH user with `docker` group membership for deploys.

## Useful commands
```bash
# Check compose config
docker compose config

# See Java args printed by start.sh
docker compose logs papermc | grep -i "JVM Args"

# Inspect Java cmdline when ps is absent
docker compose exec papermc sh -c "tr '\0' ' ' </proc/1/cmdline"
```

## Troubleshooting
- Read-only `server.properties`: ensure you are not bind-mounting it as `:ro`.
- Permissions/log4j errors: container currently runs as root to fix startup, then `start.sh` chowns and drops to `minecraft` if present.
- World not persisted: make sure volume maps `/minecraft/world` (note singular) and not `/minecraft/worlds`.
- Slow downloads: the script uses the PaperMC API. Network hiccups are retried.

## SSH example
```powershell
ssh -i "C:\path\to\minecraft-key.pem" ec2-user@ELASTIC_IP
```

Replace the path and `ELASTIC_IP` with your values.
