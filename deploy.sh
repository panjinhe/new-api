#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_NAME="prod"
RUN_BACKUP=1
RUN_GIT_PULL=0
DB_BACKEND=""

usage() {
  cat <<'EOF'
Usage: ./deploy.sh [--env-name prod|dev] [--skip-backup] [--git-pull]

Production deploys must be run from the local workstation with:
  pwsh ./scripts/deploy-fast-prod.ps1

Server-side production builds are intentionally disabled.
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
    --skip-backup)
      RUN_BACKUP=0
      shift
      ;;
    --git-pull)
      RUN_GIT_PULL=1
      shift
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

case "$ENV_NAME" in
  prod)
    echo "Server-side production builds are disabled." >&2
    echo "Use the local fast deploy flow instead:" >&2
    echo "  pwsh ./scripts/deploy-fast-prod.ps1" >&2
    exit 2
    ;;
  dev)
    DATA_DIR="$ROOT_DIR/data-dev"
    LOG_DIR="$ROOT_DIR/logs-dev"
    ENV_FILE="$ROOT_DIR/.env.dev"
    COMPOSE_FILE="$ROOT_DIR/docker-compose.dev.yml"
    HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:3000/api/status}"
    POSTGRES_COMPOSE_FILE="$ROOT_DIR/docker-compose.dev.postgres.yml"
    ;;
  *)
    echo "Unsupported env name: $ENV_NAME" >&2
    exit 1
    ;;
esac

load_env_file() {
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
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

probe_health() {
  local url="$1"

  if command -v curl >/dev/null 2>&1; then
    curl --silent --fail "$url" | grep -Eq '"success":[[:space:]]*true'
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -q -O - "$url" | grep -Eq '"success":[[:space:]]*true'
    return
  fi

  return 0
}

seed_from_legacy_layout() {
  local legacy_db="$ROOT_DIR/one-api.db"
  local legacy_data_dir="$ROOT_DIR/data"

  if [[ "$ENV_NAME" != "prod" || "$DB_BACKEND" != "sqlite" ]]; then
    return
  fi

  if [[ ! -f "$DATA_DIR/one-api.db" && -f "$legacy_db" ]]; then
    echo "Seeding $DATA_DIR/one-api.db from legacy root database."
    cp "$legacy_db" "$DATA_DIR/one-api.db"
  fi

  if [[ -d "$legacy_data_dir" ]]; then
    echo "Copying legacy ./data files into $DATA_DIR (existing files are kept)."
    cp -Rn "$legacy_data_dir/." "$DATA_DIR/" 2>/dev/null || true
  fi
}

mkdir -p "$DATA_DIR" "$LOG_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE" >&2
  echo "Create it from $(basename "$ENV_FILE").example before deploying." >&2
  exit 1
fi

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

if [[ "$RUN_GIT_PULL" -eq 1 ]]; then
  if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$ROOT_DIR" pull --ff-only
  else
    echo "Current directory is not a git worktree." >&2
    exit 1
  fi
fi

seed_from_legacy_layout

if [[ "$RUN_BACKUP" -eq 1 ]]; then
  "$ROOT_DIR/backup.sh" --env-name "$ENV_NAME"
fi

compose_with_files up -d --build --remove-orphans

echo "Waiting for health check: $HEALTH_URL"
for _ in $(seq 1 30); do
  if probe_health "$HEALTH_URL"; then
    echo "Deployment succeeded."
    compose_with_files ps
    exit 0
  fi
  sleep 2
done

echo "Health check failed. Recent container status:" >&2
compose_with_files ps >&2 || true
compose_with_files logs --tail 100 >&2 || true
exit 1
