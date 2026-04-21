#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="${APP_DIR:-/opt/new-api/app}"
ENV_NAME="${ENV_NAME:-prod}"
SERVICE_NAME="${SERVICE_NAME:-new-api-backup}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
ON_CALENDAR="${BACKUP_ON_CALENDAR:-*-*-* 04:20:00}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"

usage() {
  cat <<'EOF'
Usage: sudo ./scripts/install-prod-backup-timer.sh [--app-dir PATH] [--env-name prod|dev] [--service-name NAME] [--retention-days N] [--on-calendar "*-*-* 04:20:00"]
EOF
}

while (($# > 0)); do
  case "$1" in
    --app-dir)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --app-dir" >&2
        exit 1
      }
      APP_DIR="$2"
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
    --service-name)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --service-name" >&2
        exit 1
      }
      SERVICE_NAME="$2"
      shift 2
      ;;
    --retention-days)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --retention-days" >&2
        exit 1
      }
      RETENTION_DAYS="$2"
      shift 2
      ;;
    --on-calendar)
      [[ $# -ge 2 ]] || {
        echo "Missing value for --on-calendar" >&2
        exit 1
      }
      ON_CALENDAR="$2"
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

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "Please run this script with sudo or as root." >&2
  exit 1
fi

if [[ ! "$RETENTION_DAYS" =~ ^[0-9]+$ ]]; then
  echo "Invalid retention days: $RETENTION_DAYS" >&2
  exit 1
fi

SERVICE_TEMPLATE="$ROOT_DIR/deploy/systemd/new-api-backup.service.example"
TIMER_TEMPLATE="$ROOT_DIR/deploy/systemd/new-api-backup.timer.example"
SERVICE_FILE="$SYSTEMD_DIR/$SERVICE_NAME.service"
TIMER_FILE="$SYSTEMD_DIR/$SERVICE_NAME.timer"

if [[ ! -f "$SERVICE_TEMPLATE" || ! -f "$TIMER_TEMPLATE" ]]; then
  echo "Systemd template files are missing." >&2
  exit 1
fi

escaped_app_dir=$(printf '%s' "$APP_DIR" | sed 's/[\/&]/\\&/g')
escaped_env_name=$(printf '%s' "$ENV_NAME" | sed 's/[\/&]/\\&/g')
escaped_service_name=$(printf '%s' "$SERVICE_NAME" | sed 's/[\/&]/\\&/g')
escaped_retention_days=$(printf '%s' "$RETENTION_DAYS" | sed 's/[\/&]/\\&/g')
escaped_on_calendar=$(printf '%s' "$ON_CALENDAR" | sed 's/[\/&]/\\&/g')

sed \
  -e "s/__APP_DIR__/$escaped_app_dir/g" \
  -e "s/__ENV_NAME__/$escaped_env_name/g" \
  -e "s/__RETENTION_DAYS__/$escaped_retention_days/g" \
  "$SERVICE_TEMPLATE" > "$SERVICE_FILE"

sed \
  -e "s/__SERVICE_NAME__/$escaped_service_name/g" \
  -e "s/__ON_CALENDAR__/$escaped_on_calendar/g" \
  "$TIMER_TEMPLATE" > "$TIMER_FILE"

systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME.timer"

echo "Installed $SERVICE_FILE"
echo "Installed $TIMER_FILE"
systemctl status "$SERVICE_NAME.timer" --no-pager
