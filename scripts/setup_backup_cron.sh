#!/bin/bash

# Setup cron job for PostgreSQL backup - Ishchi Bot

SCRIPT_DIR="/home/adminuser/ishchi-bot/scripts"
BACKUP_SCRIPT="$SCRIPT_DIR/backup_postgres.sh"
MONITOR_SCRIPT="$SCRIPT_DIR/monitor_backup.sh"

# Create scripts directory
sudo mkdir -p "$SCRIPT_DIR"
sudo mkdir -p /home/adminuser/logs
sudo mkdir -p /home/adminuser/backups/ishchi-bot

# Copy backup scripts
sudo cp backup_postgres.sh "$BACKUP_SCRIPT"
sudo cp monitor_backup.sh "$MONITOR_SCRIPT"
sudo chmod +x "$BACKUP_SCRIPT"
sudo chmod +x "$MONITOR_SCRIPT"

# Create log file
sudo touch /home/adminuser/logs/ishchi-bot-backup.log
sudo chmod 666 /home/adminuser/logs/ishchi-bot-backup.log

# Add cron job (hourly at minute 0)
echo "0 * * * * $BACKUP_SCRIPT >> /home/adminuser/logs/ishchi-bot-backup.log 2>&1" | sudo crontab -

echo "✅ Backup cron job installed successfully!"
echo "📁 Backup script: $BACKUP_SCRIPT"
echo "📁 Monitor script: $MONITOR_SCRIPT"
echo "📄 Log file: /home/adminuser/logs/ishchi-bot-backup.log"
echo "⏰ Schedule: Every hour (0 * * * *)"
echo ""
echo "Backup retention policy:"
echo "  • Hourly: 24 backups (1 day)"
echo "  • Daily: 30 backups (1 month)"
echo "  • Weekly: 12 backups (3 months)"
echo ""
echo "Useful commands:"
echo "  Check cron job:     sudo crontab -l"
echo "  Monitor backups:    $MONITOR_SCRIPT"
echo "  View logs:          tail -f /home/adminuser/logs/ishchi-bot-backup.log"
echo "  Restore backup:     ./restore_backup.sh <backup_file>"
