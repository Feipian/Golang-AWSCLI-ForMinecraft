
```
You have to create one key pair for your ec2 instance, for ssh login
```

# Use the downloaded .pem file
ssh -i "C:\path\to\minecraft-key.pem" ec2-user@PUBLIC_IP

```
Replace the path and publicIP
```


# Cleanup.go
It can clean all resource our create, avoid forget close ec2 to getting charge
- [ ] maybe need to change remove rank(order) to avoid dependent. so cant delete some resource

# sshConnect.go

```
  go run sshConnect.go --key "C:\path\to\minecraft-key.pem" --run
```

- [X] VPC
- [X] Internet GateWay


- [X] bash for install paper mc
- [ ] how to load plugin to my minecraft server using s3??
    - [ ] or using docker package my paper mc using some special feature run server?? 

# Docker Setup for PaperMC

This project now includes a complete Docker setup for running PaperMC server. Here's how to use it:

## Quick Start

1. **Build and start the server:**
   ```bash
   # Build the Docker image
   docker build -t papermc-server .
   
   # Start the server using Docker Compose
   docker-compose up -d
   ```

2. **Or use the management script:**
   ```bash
   # On Linux/Mac
   ./docker-run.sh build
   ./docker-run.sh start
   
   # On Windows
   docker-run.sh build
   docker-run.sh start
   ```

## Configuration

### Environment Variables
You can customize your server by modifying the environment variables in `docker-compose.yml`:

- `MINECRAFT_VERSION`: Minecraft version (default: 1.20.4)
- `MEMORY_SIZE`: JVM memory allocation (default: 2G)
- `EULA`: Accept Minecraft EULA (default: true)
- `MAX_PLAYERS`: Maximum players (default: 20)
- `GAMEMODE`: Default gamemode (default: survival)
- `DIFFICULTY`: World difficulty (default: easy)

### Server Properties
The server uses `server.properties.template` as a base. You can:
- Modify the template file directly
- Mount a custom `server.properties` file in docker-compose.yml

### Plugin Management
Install plugins using the provided scripts:

```bash
# Install specific plugin
./plugins/install-plugins.sh install WorldEdit

# Install all available plugins
./plugins/install-plugins.sh install-all

# List available plugins
./plugins/install-plugins.sh list
```

Available plugins:
- WorldEdit
- EssentialsX
- LuckPerms
- Vault
- WorldGuard
- Dynmap
- GriefPrevention
- CoreProtect

## Management Commands

```bash
# View server logs
docker-compose logs -f

# Stop the server
docker-compose down

# Restart the server
docker-compose restart

# Backup server data
./docker-run.sh backup

# Check server status
./docker-run.sh status
```

## Data Persistence

The following directories are mounted as volumes for data persistence:
- `./data` - World data
- `./plugins` - Plugin files
- `./logs` - Server logs

## Integration with AWS

This Docker setup can be easily integrated with your existing AWS deployment:

1. **Deploy to EC2:**
   ```bash
   # Copy Docker files to your EC2 instance
   scp -i "minecraft-key.pem" -r . ec2-user@YOUR_EC2_IP:/home/ec2-user/minecraft/
   
   # SSH into EC2 and start the server
   ssh -i "minecraft-key.pem" ec2-user@YOUR_EC2_IP
   cd minecraft
   docker-compose up -d
   ```

2. **Use with S3 for plugin storage:**
   - Upload plugins to S3 bucket
   - Modify startup script to download from S3
   - Use AWS CLI in container to sync plugins

## Troubleshooting

- **Server won't start:** Check if port 25565 is available
- **Out of memory:** Increase `MEMORY_SIZE` in docker-compose.yml
- **Plugin issues:** Check plugin compatibility with your Minecraft version
- **Connection issues:** Verify firewall settings and port forwarding

# Waitting to do:

- [ ] make a app can control open minecraft server
- [X] Docker setup for PaperMC server
- [X] Plugin management system
- [ ] Integration with S3 for plugin storage
- [ ] Web-based admin panel
- [ ] How to read my costom plugin i made