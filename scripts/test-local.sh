#!/bin/bash
# Local testing script

echo "Testing Docker Compose configuration..."
docker-compose config

echo "Building image..."
docker-compose build

echo "Starting services..."
docker-compose up -d

echo "Checking logs..."
docker-compose logs -f --tail=50

echo "Stopping services..."
docker-compose down
