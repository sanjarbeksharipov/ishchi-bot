#!/bin/bash

# Monitor backup status for Ishchi Bot

BACKUP_DIR="/home/adminuser/backups/ishchi-bot"
LOG_FILE="/home/adminuser/logs/ishchi-bot-backup.log"

echo "🔍 Ishchi Bot Backup Status"
echo "=================================="

# Check if backup directories exist
for dir in hourly daily weekly; do
    backup_count=$(ls -1 "$BACKUP_DIR/$dir"/*.sql.gz 2>/dev/null | wc -l)
    latest_backup=$(ls -1t "$BACKUP_DIR/$dir"/*.sql.gz 2>/dev/null | head -1)
    
    if [ "$backup_count" -gt 0 ]; then
        latest_date=$(stat -f%m "$latest_backup" 2>/dev/null || stat -c%Y "$latest_backup" 2>/dev/null)
        latest_human=$(date -r "$latest_date" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -d "@$latest_date" '+%Y-%m-%d %H:%M:%S')
        echo "📁 $dir: $backup_count backups, latest: $latest_human"
    else
        echo "📁 $dir: No backups found"
    fi
done

echo ""
echo "💾 Disk Usage:"
du -sh "$BACKUP_DIR"/* 2>/dev/null

echo ""
echo "📄 Recent log entries:"
tail -18 "$LOG_FILE"
