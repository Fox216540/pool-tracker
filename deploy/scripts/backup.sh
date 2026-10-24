#!/usr/bin/env sh
set -eu

DATABASE_PATH=${DATABASE_PATH:-/data/pool-tracker.db}
BACKUP_DIR=${BACKUP_DIR:-/backups}
RETENTION_DAYS=${RETENTION_DAYS:-14}
mkdir -p "$BACKUP_DIR"
timestamp=$(date -u +%Y%m%dT%H%M%SZ)
sqlite3 "$DATABASE_PATH" ".backup '$BACKUP_DIR/pool-tracker-$timestamp.db'"
find "$BACKUP_DIR" -type f -name 'pool-tracker-*.db' -mtime +"$RETENTION_DAYS" -delete
