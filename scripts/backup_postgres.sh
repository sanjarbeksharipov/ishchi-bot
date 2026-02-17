#!/bin/bash

# PostgreSQL Backup Script for Ishchi Bot
# Automated backup with hourly, daily, and weekly retention

# Configuration
BACKUP_DIR="/home/adminuser/backups/ishchi-bot"
DB_NAME="${DB_NAME:-telegram_bot}"
DB_USER="${DB_USER:-postgres}"
DB_HOST="localhost"
DB_PORT="5432"
CONTAINER_NAME="ishchi-bot-postgres"
LAST_BACKUP_FILE=""
# Retention settings
KEEP_HOURLY=24         # Keep last 24 hours
KEEP_DAILY=30          # Keep last 30 days
KEEP_WEEKLY=12         # Keep last 12 weeks

# Logging
LOG_FILE="/home/adminuser/logs/ishchi-bot-backup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function with proper timestamp
log() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "[$timestamp] $1" | tee -a "$LOG_FILE"
}

# Error handling
error_exit() {
    log "${RED}ERROR: $1${NC}"
    exit 1
}

# Create backup directory with proper permissions
create_backup_dirs() {
    log "${YELLOW}1. Creating backup directories... ${NC}"
    
    # Create backup directories for different retention periods
    for dir in hourly daily weekly; do
        if ! mkdir -p "$BACKUP_DIR"/$dir; then
            error_exit "Failed to create backup directory: $dir"
        fi
        chmod 755 "$BACKUP_DIR"/$dir
    done
    
    # Set proper permissions
    chmod 755 "$BACKUP_DIR"
    
    log "${GREEN}====== Backup directories created successfully =======${NC}"
}

# Check if PostgreSQL container is running
check_postgres() {
    log "${YELLOW}2. Checking PostgreSQL container status...${NC}"
    
    # Check if container exists and is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        # Try to find any postgres container
        local postgres_containers=$(docker ps --format "table {{.Names}}" | grep postgres)
        if [ -n "$postgres_containers" ]; then
            log "Available postgres containers: $postgres_containers"
        fi
        error_exit "PostgreSQL container '$CONTAINER_NAME' is not running"
    fi
    
    # Check if database is accessible
    log "Testing database connection..."
    if ! docker exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
        error_exit "PostgreSQL database is not ready or accessible"
    fi
    
    log "${GREEN}======= PostgreSQL container is running and accessible ========${NC}"
}

# Take backup function - FIXED
take_backup() {
    local backup_type=$1
    local backup_subdir=$2
    local filename_format=$3
    
    local timestamp=$(date +"$filename_format")
    local backup_filename="ishchi_backup_${backup_type}_${timestamp}.sql"
    local backup_file="$BACKUP_DIR/$backup_subdir/$backup_filename"
    
    log "${YELLOW}3. Starting $backup_type backup...${NC}"
    log "Backup file will be: $backup_file"
    
    # Ensure backup subdirectory exists
    mkdir -p "$BACKUP_DIR/$backup_subdir"
    
    # Take the backup with proper error handling
    log "Executing pg_dump..."
    if docker exec "$CONTAINER_NAME" pg_dump \
        -U "$DB_USER" \
        -d "$DB_NAME" > "$backup_file" 2>/dev/null; then
        
        # Check if backup file was created and has content
        if [ ! -f "$backup_file" ]; then
            error_exit "Backup file was not created: $backup_file"
        fi
        
        local file_size=$(stat -c%s "$backup_file" 2>/dev/null || stat -f%z "$backup_file" 2>/dev/null)
        if [ "$file_size" -lt 1000 ]; then  # Less than 1KB indicates empty/failed backup
            log "Backup file content preview:"
            head -10 "$backup_file" | tee -a "$LOG_FILE"
            rm -f "$backup_file"
            error_exit "$backup_type backup file is too small ($file_size bytes) - likely failed"
        fi
        
        # Compress the backup
        log "Compressing backup file..."
        if gzip "$backup_file"; then
            backup_file="${backup_file}.gz"
            local compressed_size=$(stat -c%s "$backup_file" 2>/dev/null || stat -f%z "$backup_file" 2>/dev/null)
            log "${GREEN}✅ $backup_type backup completed successfully: $(basename "$backup_file") (${compressed_size} bytes compressed)${NC}"
            LAST_BACKUP_FILE="$backup_file"
        else
            rm -f "$backup_file"
            error_exit "Failed to compress backup file"
        fi
    else
        rm -f "$backup_file"
        error_exit "Failed to create $backup_type backup using pg_dump"
    fi
}

# Clean old backups - IMPROVED
cleanup_old_backups() {
    local backup_dir=$1
    local keep_count=$2
    local backup_type=$3
    
    log "${YELLOW}5. Cleaning old $backup_type backups (keeping last $keep_count)...${NC}"
    
    # Check if directory exists
    if [ ! -d "$BACKUP_DIR/$backup_dir" ]; then
        log "Backup directory does not exist: $BACKUP_DIR/$backup_dir"
        return
    fi
    
    # Count current backups
    local current_count=$(find "$BACKUP_DIR/$backup_dir" -name "ishchi_backup_*.sql.gz" 2>/dev/null | wc -l)
    
    if [ "$current_count" -gt "$keep_count" ]; then
        local delete_count=$((current_count - keep_count))
        
        log "Found $current_count backups, will delete $delete_count oldest"
        
        # Delete oldest backups
        find "$BACKUP_DIR/$backup_dir" -name "ishchi_backup_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | \
        sort -n | \
        head -n "$delete_count" | \
        cut -d' ' -f2- | \
        while IFS= read -r file; do
            if [ -f "$file" ] && rm -f "$file"; then
                log "🗑️  Deleted old backup: $(basename "$file")"
            else
                log "${RED}Failed to delete: $(basename "$file")${NC}"
            fi
        done
        
        log "${GREEN}Cleaned $delete_count old $backup_type backups${NC}"
    else
        log "No cleanup needed for $backup_type backups ($current_count/$keep_count)"
    fi
}

# Verify backup integrity - IMPROVED
verify_backup() {
    local backup_file=$1
    local backup_type=$2
    
    log "${YELLOW}4. Verifying $backup_type backup integrity...${NC}"
    
    # Check if file exists and is readable
    if [ ! -f "$backup_file" ]; then
        error_exit "Backup file does not exist: $backup_file"
    fi
    
    if [ ! -r "$backup_file" ]; then
        error_exit "Backup file is not readable: $backup_file"
    fi
    
    # Check file size
    local file_size=$(stat -c%s "$backup_file" 2>/dev/null || stat -f%z "$backup_file" 2>/dev/null)
    if [ "$file_size" -lt 100 ]; then
        error_exit "Backup file is too small: $file_size bytes"
    fi
    
    # Check if gzip file is valid
    if ! gzip -t "$backup_file" 2>/dev/null; then
        error_exit "Backup file is corrupted (gzip test failed): $backup_file"
    fi
    
    # Check if SQL content is valid (basic check)
    if ! zcat "$backup_file" 2>/dev/null | head -20 | grep -q "PostgreSQL database dump"; then
        log "Backup file header content:"
        zcat "$backup_file" 2>/dev/null | head -5 | tee -a "$LOG_FILE"
        error_exit "Backup file does not contain valid PostgreSQL dump header"
    fi
    
    log "${GREEN}✅ Backup verification passed (${file_size} bytes)${NC}"
}

# Get disk usage
check_disk_usage() {
    if [ -d "$BACKUP_DIR" ]; then
        local backup_dir_usage=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
        local available_space=$(df -h "$BACKUP_DIR" 2>/dev/null | awk 'NR==2 {print $4}')
        
        log "${YELLOW}6. 💾 Backup directory usage: $backup_dir_usage, Available space: $available_space${NC}"
        
        # Check if less than 1GB available
        local available_kb=$(df "$BACKUP_DIR" 2>/dev/null | awk 'NR==2 {print $4}')
        if [ "$available_kb" -lt 1048576 ]; then  # Less than 1GB
            log "${RED}⚠️  Warning: Low disk space available (less than 1GB)${NC}"
        fi
    fi
}

# Determine backup type based on current time
get_backup_type() {
    local hour=$(date +%H)
    local day=$(date +%u)  # 1=Monday, 7=Sunday
    
    # Weekly backup on Sunday at midnight
    if [ "$day" -eq 7 ] && [ "$hour" -eq 0 ]; then
        echo "weekly"
    # Daily backup at midnight
    elif [ "$hour" -eq 0 ]; then
        echo "daily"
    # Hourly backup
    else
        echo "hourly"
    fi
}

# Main backup function - IMPROVED
main() {
    log "${GREEN}=== Starting Ishchi Bot PostgreSQL Backup ===${NC}"
    
    # Initialize
    create_backup_dirs
    check_postgres
    
    # Determine backup type
    local backup_type=$(get_backup_type)
    log "Backup type: $backup_type"
    
    case $backup_type in
        hourly)
            take_backup "hourly" "hourly" "%Y%m%d_%H" || error_exit "Failed to create hourly backup"
            verify_backup "$LAST_BACKUP_FILE" "hourly"
            cleanup_old_backups "hourly" "$KEEP_HOURLY" "hourly"
            ;;
        daily)
            take_backup "daily" "daily" "%Y%m%d" || error_exit "Failed to create daily backup"
            verify_backup "$LAST_BACKUP_FILE" "daily"
            cleanup_old_backups "daily" "$KEEP_DAILY" "daily"
            # Also take hourly backup
            take_backup "hourly" "hourly" "%Y%m%d_%H" || error_exit "Failed to create hourly backup"
            cleanup_old_backups "hourly" "$KEEP_HOURLY" "hourly"
            ;;
        weekly)
            take_backup "weekly" "weekly" "%Y_week%U" || error_exit "Failed to create weekly backup"
            verify_backup "$LAST_BACKUP_FILE" "weekly"
            cleanup_old_backups "weekly" "$KEEP_WEEKLY" "weekly"
            # Also take daily and hourly backups
            take_backup "daily" "daily" "%Y%m%d" || error_exit "Failed to create daily backup"
            cleanup_old_backups "daily" "$KEEP_DAILY" "daily"
            take_backup "hourly" "hourly" "%Y%m%d_%H" || error_exit "Failed to create hourly backup"
            cleanup_old_backups "hourly" "$KEEP_HOURLY" "hourly"
            ;;
    esac
  
    # Check disk usage
    check_disk_usage
    
    log "${GREEN}=== Backup Process Completed Successfully ===${NC}"
}

# Run main function
main "$@"
