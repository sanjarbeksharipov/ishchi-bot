#!/bin/bash

# Restore PostgreSQL backup for Ishchi Bot

BACKUP_DIR="/home/adminuser/backups/ishchi-bot"
CONTAINER_NAME="ishchi-bot-postgres"
DB_NAME="${DB_NAME:-telegram_bot}"
DB_USER="${DB_USER:-postgres}"

if [ $# -eq 0 ]; then
    echo "Usage: $0 <backup_file>"
    echo ""
    echo "Available backups:"
    find "$BACKUP_DIR" -name "*.sql.gz" -exec basename {} \; | sort
    exit 1
fi

BACKUP_FILE="$1"

# Find the backup file
if [ ! -f "$BACKUP_FILE" ]; then
    FULL_PATH=$(find "$BACKUP_DIR" -name "*$BACKUP_FILE*" | head -1)
    if [ -z "$FULL_PATH" ]; then
        echo "❌ Backup file not found: $BACKUP_FILE"
        exit 1
    fi
    BACKUP_FILE="$FULL_PATH"
fi

echo "🔄 Restoring backup: $(basename "$BACKUP_FILE")"
echo "⚠️  This will overwrite the current database!"
read -p "Are you sure? (y/N): " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Restore cancelled"
    exit 1
fi

# Stop the bot
echo "🛑 Stopping ishchi bot..."
docker compose stop bot

# Restore database
echo "📥 Restoring database..."
# Disable foreign key constraints during restore
echo "🔧 Temporarily disabling foreign key constraints..."
docker exec -i "$CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -c "SET session_replication_role = 'replica';" 2>/dev/null

if zcat "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" 2>/dev/null; then
    # Re-enable foreign key constraints
    echo "🔧 Re-enabling foreign key constraints..."
    docker exec -i "$CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -c "SET session_replication_role = 'origin';" 2>/dev/null
    echo "✅ Database restored successfully"
    
    # Start the bot
    echo "🚀 Starting ishchi bot..."
    docker compose start bot
    
    echo "✅ Restore completed successfully!"
else
    # Re-enable foreign key constraints even on failure
    echo "🔧 Re-enabling foreign key constraints..."
    docker exec -i "$CONTAINER_NAME" psql -d "$DB_NAME" -U "$DB_USER" -c "SET session_replication_role = 'origin';" 2>/dev/null
    echo "❌ Restore failed!"
    exit 1
fi
