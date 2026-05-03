[CmdletBinding()]
param(
  [string]$HostAlias = "aheapi-itdun",
  [string]$SshConfig = "ops/ssh/config.local",
  [string]$RemoteAppDir = "/opt/new-api/app",
  [string]$HealthUrl = "https://aheapi.com/api/status",
  [int]$HealthAttempts = 30,
  [int]$HealthSleepSec = 2,
  [switch]$ForceRecreate
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$SshConfigPath = Join-Path $RootDir $SshConfig

if (-not (Get-Command ssh -ErrorAction SilentlyContinue)) {
  throw "Required command not found: ssh"
}

if (-not (Test-Path -LiteralPath $SshConfigPath)) {
  throw "Missing SSH config: $SshConfigPath"
}

$recreateFlag = if ($ForceRecreate) { " --force-recreate" } else { "" }
$remoteScript = @"
set -euo pipefail

cd "$RemoteAppDir"

if [ ! -f .env.prod ]; then
  echo "Missing .env.prod in $RemoteAppDir" >&2
  exit 1
fi

echo "==> Compose SQL_DSN preview"
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml config | grep 'SQL_DSN:' || {
  echo "Unable to preview SQL_DSN from compose config" >&2
  exit 1
}

if docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml config | grep -q 'change-me@postgres'; then
  echo "Refusing to restart: compose config expanded SQL_DSN with change-me." >&2
  exit 1
fi

echo "==> Restart production app container"
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-deps$recreateFlag new-api

echo "==> Runtime SQL_DSN"
docker inspect new-api-prod --format '{{range .Config.Env}}{{println .}}{{end}}' | grep '^SQL_DSN=' || {
  echo "Runtime SQL_DSN missing" >&2
  exit 1
}

if docker inspect new-api-prod --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -q '^SQL_DSN=.*change-me@postgres'; then
  echo "Refusing to continue: runtime SQL_DSN uses change-me." >&2
  docker logs --tail 80 new-api-prod >&2 || true
  exit 1
fi

echo "==> Wait for health check"
for i in `$(seq 1 $HealthAttempts); do
  if curl --silent --fail "$HealthUrl" | grep -Eq '"success":[[:space:]]*true'; then
    docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml ps
    echo "Production app is healthy."
    exit 0
  fi
  sleep $HealthSleepSec
done

echo "Health check failed. Recent logs:" >&2
docker logs --tail 120 new-api-prod >&2 || true
exit 1
"@

$remoteScript = $remoteScript -replace "`r`n", "`n"
$remoteScript | ssh -F $SshConfigPath $HostAlias "bash -s"
if ($LASTEXITCODE -ne 0) {
  throw "Production app restart failed with exit code $LASTEXITCODE"
}
