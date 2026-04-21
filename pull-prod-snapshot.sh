#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_NAME="prod"
REMOTE="${REMOTE:-}"
REMOTE_APP_DIR="${REMOTE_APP_DIR:-/opt/new-api/app}"
LOCAL_SNAPSHOT_ROOT="${LOCAL_SNAPSHOT_ROOT:-$ROOT_DIR/data-prod-snapshot}"

usage() {
  cat <<'EOF'
Usage: REMOTE=user@host ./pull-prod-snapshot.sh [--remote user@host] [--remote-app-dir /opt/new-api/app] [--env-name prod|dev]
EOF
}

while (($# > 0)); do
  case "$1" in
    --remote)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --remote" >&2
        exit 1
      }
      REMOTE="$2"
      shift 2
      ;;
    --remote-app-dir)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --remote-app-dir" >&2
        exit 1
      }
      REMOTE_APP_DIR="$2"
      shift 2
      ;;
    --env-name)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --env-name" >&2
        exit 1
      }
      ENV_NAME="$2"
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

if [[ -z "$REMOTE" ]]; then
  echo "REMOTE is required, for example: REMOTE=ubuntu@server ./pull-prod-snapshot.sh" >&2
  exit 1
fi

mkdir -p "$LOCAL_SNAPSHOT_ROOT"

REMOTE_BACKUP_PATH="$(
  ssh "$REMOTE" "cd '$REMOTE_APP_DIR' && ./backup.sh --env-name '$ENV_NAME' --print-path"
)"

if [[ -z "$REMOTE_BACKUP_PATH" ]]; then
  echo "Remote backup path is empty." >&2
  exit 1
fi

scp -r "$REMOTE:$REMOTE_BACKUP_PATH" "$LOCAL_SNAPSHOT_ROOT/"

echo "Snapshot downloaded to $LOCAL_SNAPSHOT_ROOT/$(basename "$REMOTE_BACKUP_PATH")"
