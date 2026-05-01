#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_PATH="${UPAY_DB_PATH:-$ROOT_DIR/DBS/upay_pro.db}"
LOG_DIR="${UPAY_LOG_DIR:-$ROOT_DIR/logs}"
BACKUP_DIR="${UPAY_BACKUP_DIR:-$ROOT_DIR/backups}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
WORK_DIR="$BACKUP_DIR/upay-pro-$TIMESTAMP"
ARCHIVE_PATH="$BACKUP_DIR/upay-pro-$TIMESTAMP.tar.gz"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

mkdir -p "$WORK_DIR"

if [[ ! -f "$DB_PATH" ]]; then
  fail "database not found: $DB_PATH"
fi

if command -v sqlite3 >/dev/null 2>&1; then
  sqlite3 "$DB_PATH" ".backup '$WORK_DIR/upay_pro.db'" || fail "sqlite backup failed"
else
  cp "$DB_PATH" "$WORK_DIR/upay_pro.db"
fi

if [[ -d "$LOG_DIR" ]]; then
  mkdir -p "$WORK_DIR/logs"
  cp -R "$LOG_DIR"/. "$WORK_DIR/logs/"
fi

cat > "$WORK_DIR/manifest.txt" <<EOF
created_at=$TIMESTAMP
db_path=$DB_PATH
log_dir=$LOG_DIR
host=$(hostname)
EOF

tar -czf "$ARCHIVE_PATH" -C "$BACKUP_DIR" "upay-pro-$TIMESTAMP"
rm -rf "$WORK_DIR"

echo "Backup created: $ARCHIVE_PATH"
