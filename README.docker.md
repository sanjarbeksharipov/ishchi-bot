# Docker Deployment Guide

This guide explains how to deploy the Ishchi Bot using Docker and Docker Compose.

## Prerequisites

- Docker (version 20.10 or higher)
- Docker Compose (version 2.0 or higher)

## Quick Start

1. **Clone the repository and navigate to the project directory:**
   ```bash
   cd /path/to/ishchi-bot
   ```

2. **Create your `.env` file from the example:**
   ```bash
   cp .env.example .env
   ```

3. **Edit the `.env` file and add your configuration:**
   ```bash
   nano .env  # or use your preferred editor
   ```
   
   Required variables:
   - `BOT_TOKEN`: Your Telegram bot token from [@BotFather](https://t.me/botfather)
   - `BOT_ADMIN_IDS`: Comma-separated list of admin Telegram user IDs
   - `DB_PASSWORD`: Secure password for PostgreSQL

4. **Build and start the services:**
   ```bash
   docker-compose up -d
   ```

5. **Check the logs:**
   ```bash
   docker-compose logs -f bot
   ```

## Docker Commands

### Build and Start
```bash
# Build and start all services in detached mode
docker-compose up -d

# Build and start with rebuild
docker-compose up -d --build

# Start without building
docker-compose start
```

### Stop and Remove
```bash
# Stop services
docker-compose stop

# Stop and remove containers
docker-compose down

# Stop, remove containers and volumes (WARNING: deletes database data)
docker-compose down -v
```

### View Logs
```bash
# View all logs
docker-compose logs

# Follow logs in real-time
docker-compose logs -f

# View specific service logs
docker-compose logs -f bot
docker-compose logs -f postgres
```

### Database Management
```bash
# Access PostgreSQL shell
docker-compose exec postgres psql -U postgres -d telegram_bot

# Create database backup
docker-compose exec postgres pg_dump -U postgres telegram_bot > backup.sql

# Restore database from backup
docker-compose exec -T postgres psql -U postgres telegram_bot < backup.sql
```

### Maintenance
```bash
# Restart a service
docker-compose restart bot

# Rebuild and restart after code changes
docker-compose up -d --build bot

# View running containers
docker-compose ps

# View resource usage
docker stats
```

## Environment Variables

### Bot Configuration
- `BOT_TOKEN` (required): Telegram bot token
- `BOT_VERBOSE`: Enable verbose logging (default: false)
- `BOT_POLLER`: Polling interval (default: 10s)
- `BOT_CHANNEL_ID`: Channel ID for notifications
- `BOT_ADMIN_IDS`: Comma-separated admin user IDs
- `BOT_ADMIN_GROUP_ID`: Admin group ID for payment approvals
- `BOT_USERNAME`: Bot username

### Database Configuration
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (required in production)
- `DB_NAME`: Database name (default: telegram_bot)
- `DB_PORT`: Database port (default: 5432)
- `DB_MAX_CONNECTIONS`: Max database connections (default: 25)

### App Configuration
- `APP_ENV`: Environment (development/production)
- `LOG_LEVEL`: Logging level (debug/info/warn/error)

## Troubleshooting

### Bot is not starting
1. Check logs: `docker-compose logs bot`
2. Verify BOT_TOKEN is correct
3. Ensure database is healthy: `docker-compose ps postgres`

### Database connection errors
1. Wait for database to be ready (can take 10-30 seconds on first start)
2. Check database health: `docker-compose exec postgres pg_isready`
3. Verify credentials in `.env` file

### Port conflicts
If port 5432 is already in use:
1. Change `DB_PORT` in `.env` file
2. Update port mapping in docker-compose.yml

### Permission issues
```bash
# Fix log directory permissions
sudo chown -R $USER:$USER ./logs
```

## Production Deployment

For production deployment:

1. **Use strong passwords**
2. **Set `APP_ENV=production`**
3. **Configure proper `LOG_LEVEL`**
4. **Set up regular backups**
5. **Use Docker secrets for sensitive data** (optional):
   ```yaml
   secrets:
     bot_token:
       file: ./secrets/bot_token.txt
   ```

## Updating the Bot

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose up -d --build

# Check if update was successful
docker-compose logs -f bot
```

## Health Checks

The bot includes automatic health checks:
- Database health check runs every 10 seconds
- Bot will wait for database to be healthy before starting

## Data Persistence

Data is persisted in Docker volumes:
- `postgres_data`: Database files
- `./logs`: Application logs (mounted from host)

To backup the database volume:
```bash
docker run --rm -v ishchi-bot_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres-backup.tar.gz -C /data .
```

## Free Hosting Options (See below for recommendations)
