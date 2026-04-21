#!/bin/sh
set -eu

log() {
  echo "[entrypoint] $*"
}

TARGET_DATA_DIR="${TARGET_DATA_DIR:-/data}"
TARGET_DB_PATH="${SQLITE_PATH:-${TARGET_DATA_DIR}/one-api.db}"
TARGET_DB_PATH="${TARGET_DB_PATH%%\?*}"
LEGACY_SEED_ROOT="${LEGACY_SEED_ROOT:-/seed/workspace}"
LEGACY_DB_PATH="${LEGACY_DB_PATH:-${LEGACY_SEED_ROOT}/one-api.db}"
LEGACY_DATA_DIR="${LEGACY_DATA_DIR:-${LEGACY_SEED_ROOT}/data}"
ENABLE_LEGACY_SEED="${ENABLE_LEGACY_SEED:-true}"

mkdir -p "$TARGET_DATA_DIR"

if [ "$ENABLE_LEGACY_SEED" = "true" ]; then
  if [ ! -f "$TARGET_DB_PATH" ] && [ -f "$LEGACY_DB_PATH" ]; then
    log "Seeding SQLite database from legacy source: $LEGACY_DB_PATH -> $TARGET_DB_PATH"
    cp "$LEGACY_DB_PATH" "$TARGET_DB_PATH"
  fi

  if [ -d "$LEGACY_DATA_DIR" ]; then
    log "Merging legacy data directory into target data directory (existing files are preserved)."
    cp -an "$LEGACY_DATA_DIR/." "$TARGET_DATA_DIR/" 2>/dev/null || true
  fi
fi

exec /new-api "$@"
