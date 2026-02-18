#!/bin/bash

# Deployment script for Ishchi Bot
# Makes manual deployment safer and faster

set -e  # Exit on error

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}   Ishchi Bot Deployment Script${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}Error: docker-compose.yml not found!${NC}"
    echo "Please run this script from the project root directory"
    exit 1
fi

# Check if .env exists
if [ ! -f ".env" ]; then
    echo -e "${RED}Error: .env file not found!${NC}"
    echo "Please create .env file from .env.example"
    exit 1
fi

# Function to check if service is healthy
wait_for_health() {
    local service=$1
    local max_attempts=30
    local attempt=0
    
    echo -e "${YELLOW}Waiting for $service to be healthy...${NC}"
    
    while [ $attempt -lt $max_attempts ]; do
        if docker compose ps | grep $service | grep -q "healthy\|running"; then
            echo -e "${GREEN}✓ $service is healthy${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
        echo -n "."
    done
    
    echo -e "${RED}✗ $service failed to become healthy${NC}"
    return 1
}

# Confirm deployment
echo -e "${YELLOW}This will:${NC}"
echo "  1. Pull latest code from git"
echo "  2. Stop current services"
echo "  3. Rebuild Docker images"
echo "  4. Start services"
echo "  5. Run health checks"
echo ""
read -p "Continue with deployment? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Deployment cancelled${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}Step 1/6: Pulling latest code...${NC}"
git pull origin main || {
    echo -e "${RED}Failed to pull from git${NC}"
    exit 1
}

echo ""
echo -e "${BLUE}Step 2/6: Creating backup...${NC}"
if [ -f "/home/adminuser/ishchi-bot/scripts/backup_postgres.sh" ]; then
    sudo /home/adminuser/ishchi-bot/scripts/backup_postgres.sh || {
        echo -e "${YELLOW}Warning: Backup failed but continuing...${NC}"
    }
else
    echo -e "${YELLOW}Backup script not found, skipping...${NC}"
fi

echo ""
echo -e "${BLUE}Step 3/6: Stopping current services...${NC}"
docker compose down || {
    echo -e "${RED}Failed to stop services${NC}"
    exit 1
}

echo ""
echo -e "${BLUE}Step 4/6: Building images...${NC}"
docker compose build --no-cache || {
    echo -e "${RED}Failed to build images${NC}"
    exit 1
}

echo ""
echo -e "${BLUE}Step 5/6: Starting services...${NC}"
docker compose up -d || {
    echo -e "${RED}Failed to start services${NC}"
    exit 1
}

echo ""
echo -e "${BLUE}Step 6/6: Health checks...${NC}"

# Wait for database
wait_for_health "postgres" || {
    echo -e "${RED}Database health check failed!${NC}"
    echo "Showing database logs:"
    docker logs ishchi-bot-postgres --tail 50
    exit 1
}

# Wait for bot
sleep 5
wait_for_health "bot" || {
    echo -e "${YELLOW}Bot health check unclear, checking logs...${NC}"
    docker logs ishchi-bot --tail 30
}

echo ""
echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}   ✓ Deployment Complete!${NC}"
echo -e "${GREEN}================================================${NC}"
echo ""

# Show status
echo -e "${BLUE}Service Status:${NC}"
docker compose ps

echo ""
echo -e "${BLUE}Recent Bot Logs:${NC}"
docker logs ishchi-bot --tail 20

echo ""
echo -e "${YELLOW}Useful commands:${NC}"
echo "  View logs:    docker logs ishchi-bot -f"
echo "  Check status: docker compose ps"
echo "  Stop all:     docker compose down"
echo "  Restart bot:  docker compose restart bot"

echo ""
echo -e "${GREEN}Deployment successful! 🚀${NC}"
