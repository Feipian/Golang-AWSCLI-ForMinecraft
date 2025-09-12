# PowerShell version of test script

Write-Host "Testing Docker Compose configuration..."
docker-compose config

Write-Host "Building image..."
docker-compose build

Write-Host "Starting services..."
docker-compose up -d

Write-Host "Checking logs..."
docker-compose logs -f --tail=50

Write-Host "Stopping services..."
docker-compose down
