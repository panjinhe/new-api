#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_NAME="prod"
PRINT_PATH=0
DB_BACKEND=""
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-0}"
BACKUP_NGINX_CONFIG="${BACKUP_NGINX_CONFIG:-1}"
NGINX_BACKUP_STATUS="not_run"

usage() {
  cat <<'EOF'
Usage: ./backup.sh [--env-name prod|dev] [--print-path] [--retention-days N]
EOF
}

while (($# > 0)); do
  case "$1" in
    --env-name)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --env-name" >&2
        exit 1
      }
      ENV_NAME="$2"
      shift 2
      ;;
    --print-path)
      PRINT_PATH=1
      shift
      ;;
    --retention-days)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --retention-days" >&2
        exit 1
      }
      RETENTION_DAYS="$2"
      shift 2
      ;;
    --db)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --db" >&2
        exit 1
      }
      DB_BACKEND="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ ! "$RETENTION_DAYS" =~ ^[0-9]+$ ]]; then
  echo "Invalid retention days: $RETENTION_DAYS" >&2
  exit 1
fi

case "$BACKUP_NGINX_CONFIG" in
  0|1)
    ;;
  *)
    echo "Invalid BACKUP_NGINX_CONFIG: $BACKUP_NGINX_CONFIG (expected 0 or 1)" >&2
    exit 1
    ;;
esac

case "$ENV_NAME" in
  prod)
    DATA_DIR="${DATA_DIR:-$ROOT_DIR/data-prod}"
    LOG_DIR="${LOG_DIR:-$ROOT_DIR/logs-prod}"
    ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.prod}"
    COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.prod.yml}"
    POSTGRES_COMPOSE_FILE="${POSTGRES_COMPOSE_FILE:-$ROOT_DIR/docker-compose.prod.postgres.yml}"
    ;;
  dev)
    DATA_DIR="${DATA_DIR:-$ROOT_DIR/data-dev}"
    LOG_DIR="${LOG_DIR:-$ROOT_DIR/logs-dev}"
    ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.dev}"
    COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.dev.yml}"
    POSTGRES_COMPOSE_FILE="${POSTGRES_COMPOSE_FILE:-$ROOT_DIR/docker-compose.dev.postgres.yml}"
    ;;
  *)
    echo "Unsupported env name: $ENV_NAME" >&2
    exit 1
    ;;
esac

load_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck disable=SC1090
    . "$ENV_FILE"
    set +a
  fi
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "Docker Compose is not installed." >&2
    exit 1
  fi
}

compose_with_files() {
  compose "${COMPOSE_ARGS[@]}" "$@"
}

BACKUP_ROOT="${BACKUP_ROOT:-$ROOT_DIR/backups/$ENV_NAME}"
SQLITE_SOURCE="${SQLITE_SOURCE:-$DATA_DIR/one-api.db}"
SQLITE_SOURCE="${SQLITE_SOURCE%%\?*}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
SNAPSHOT_DIR="$BACKUP_ROOT/$TIMESTAMP"
SNAPSHOT_DB="$SNAPSHOT_DIR/one-api.db"
POSTGRES_DUMP="$SNAPSHOT_DIR/postgres.dump"

mkdir -p "$DATA_DIR" "$LOG_DIR" "$BACKUP_ROOT" "$SNAPSHOT_DIR"

load_env_file

if [[ -z "$DB_BACKEND" ]]; then
  DB_BACKEND="${DATABASE_BACKEND:-postgres}"
fi

case "$DB_BACKEND" in
  sqlite)
    COMPOSE_ARGS=(-f "$COMPOSE_FILE")
    ;;
  postgres)
    COMPOSE_ARGS=(-f "$COMPOSE_FILE" -f "$POSTGRES_COMPOSE_FILE")
    ;;
  *)
    echo "Unsupported database backend: $DB_BACKEND" >&2
    exit 1
    ;;
esac

copy_sqlite_backup() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$SQLITE_SOURCE" "$SNAPSHOT_DB" <<'PY'
import sqlite3
import sys

src = sys.argv[1].split("?", 1)[0]
dst = sys.argv[2].split("?", 1)[0]

source = sqlite3.connect(f"file:{src}?mode=ro", uri=True)
target = sqlite3.connect(dst)
with target:
    source.backup(target)
target.close()
source.close()
PY
  else
    cp "$SQLITE_SOURCE" "$SNAPSHOT_DB"
  fi
}

copy_postgres_backup() {
  compose_with_files exec -T postgres sh -lc 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "$POSTGRES_DUMP"
}

copy_nginx_config_backup() {
  if [[ "$BACKUP_NGINX_CONFIG" != "1" ]]; then
    NGINX_BACKUP_STATUS="disabled"
    return
  fi

  local nginx_backup_dir="$SNAPSHOT_DIR/nginx"
  local copied=0
  local missing_or_unreadable=0
  mkdir -p "$nginx_backup_dir/sites-available" "$nginx_backup_dir/sites-enabled"

  local source_file
  for source_file in \
    /etc/nginx/nginx.conf \
    /etc/nginx/sites-available/new-api.conf \
    /etc/nginx/sites-enabled/new-api.conf
  do
    if [[ ! -e "$source_file" ]]; then
      missing_or_unreadable=1
      printf 'missing: %s\n' "$source_file" >> "$nginx_backup_dir/README.txt"
      continue
    fi

    if [[ ! -r "$source_file" ]]; then
      missing_or_unreadable=1
      printf 'unreadable: %s\n' "$source_file" >> "$nginx_backup_dir/README.txt"
      continue
    fi

    local relative_path="${source_file#/etc/nginx/}"
    local target_file="$nginx_backup_dir/$relative_path"
    mkdir -p "$(dirname "$target_file")"
    if cp -a "$source_file" "$target_file"; then
      copied=1
    else
      missing_or_unreadable=1
      printf 'copy_failed: %s\n' "$source_file" >> "$nginx_backup_dir/README.txt"
    fi
  done

  if command -v nginx >/dev/null 2>&1; then
    if nginx -T > "$nginx_backup_dir/nginx-T.txt" 2> "$nginx_backup_dir/nginx-T.stderr.txt"; then
      copied=1
    else
      missing_or_unreadable=1
      printf 'nginx -T failed; see nginx-T.stderr.txt\n' >> "$nginx_backup_dir/README.txt"
    fi
  else
    missing_or_unreadable=1
    printf 'nginx command not found\n' >> "$nginx_backup_dir/README.txt"
  fi

  if [[ "$copied" -eq 1 && "$missing_or_unreadable" -eq 1 ]]; then
    NGINX_BACKUP_STATUS="partial"
  elif [[ "$copied" -eq 1 ]]; then
    NGINX_BACKUP_STATUS="saved"
  else
    NGINX_BACKUP_STATUS="skipped_no_readable_config"
  fi
}

cleanup_old_backups() {
  if [[ "$RETENTION_DAYS" -le 0 ]]; then
    return
  fi

  if [[ ! -d "$BACKUP_ROOT" ]]; then
    return
  fi

  local cutoff_days=$((RETENTION_DAYS - 1))
  if [[ "$cutoff_days" -lt 0 ]]; then
    cutoff_days=0
  fi

  while IFS= read -r -d '' old_dir; do
    rm -rf -- "$old_dir"
    echo "Pruned old backup: $old_dir"
  done < <(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -mtime "+$cutoff_days" -print0)
}

if [[ "$DB_BACKEND" == "postgres" ]]; then
  copy_postgres_backup
elif [[ -f "$SQLITE_SOURCE" ]]; then
  copy_sqlite_backup
fi

if [[ -d "$DATA_DIR" ]]; then
  tar -czf "$SNAPSHOT_DIR/data-extra.tar.gz" --exclude='./one-api.db' -C "$DATA_DIR" .
fi

if [[ -f "$ENV_FILE" ]]; then
  cp "$ENV_FILE" "$SNAPSHOT_DIR/$(basename "$ENV_FILE").backup"
fi

if [[ -f "$COMPOSE_FILE" ]]; then
  cp "$COMPOSE_FILE" "$SNAPSHOT_DIR/$(basename "$COMPOSE_FILE")"
fi

if [[ -f "$POSTGRES_COMPOSE_FILE" ]]; then
  cp "$POSTGRES_COMPOSE_FILE" "$SNAPSHOT_DIR/$(basename "$POSTGRES_COMPOSE_FILE")"
fi

copy_nginx_config_backup

{
  echo "env_name=$ENV_NAME"
  echo "created_at=$(date -Iseconds)"
  echo "hostname=$(hostname)"
  echo "db_backend=$DB_BACKEND"
  echo "retention_days=$RETENTION_DAYS"
  echo "nginx_backup_enabled=$BACKUP_NGINX_CONFIG"
  echo "nginx_backup_status=$NGINX_BACKUP_STATUS"
  echo "data_dir=$DATA_DIR"
  echo "sqlite_source=$SQLITE_SOURCE"
  if command -v git >/dev/null 2>&1 && git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "git_commit=$(git -C "$ROOT_DIR" rev-parse HEAD)"
  fi
} > "$SNAPSHOT_DIR/metadata.txt"

if (( PRINT_PATH )); then
  echo "$SNAPSHOT_DIR"
  exit 0
fi

echo "Backup created: $SNAPSHOT_DIR"
if [[ -f "$POSTGRES_DUMP" ]]; then
  echo "PostgreSQL snapshot: $POSTGRES_DUMP"
elif [[ -f "$SNAPSHOT_DB" ]]; then
  echo "SQLite snapshot: $SNAPSHOT_DB"
else
  echo "Database snapshot skipped."
fi

cleanup_old_backups
