[CmdletBinding()]
param(
  [string]$HostAlias = "aheapi-itdun",
  [string]$SshConfig = "ops/ssh/config.local",
  [string]$RemoteAppDir = "/opt/new-api/app",
  [string]$RemoteTmpDir = "/opt/new-api/tmp",
  [string]$ImageName = "new-api-local:prod",
  [string]$PublicHealthUrl = "https://aheapi.com/api/status",
  [string]$BuildOutput = "new-api-linux-amd64",
  [ValidateSet("blue", "green")]
  [string]$FirstIdleColor = "blue",
  [int]$HealthAttempts = 30,
  [int]$HealthSleepSec = 2,
  [int]$DrainTimeoutSec = 120,
  [switch]$SkipBuild,
  [switch]$SkipFrontendBuild,
  [switch]$ForceFrontendBuild,
  [switch]$SkipSourceSync,
  [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$SshConfigPath = Join-Path $RootDir $SshConfig
$BuildScript = Join-Path $PSScriptRoot "build-linux-release.ps1"
$BuildOutputPath = Join-Path $RootDir $BuildOutput
$DeployTmpDir = Join-Path $RootDir "codex-tmp\deploy"
$Stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

function Assert-Command {
  param([Parameter(Mandatory = $true)][string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required command not found: $Name"
  }
}

function Assert-ElfBinary {
  param([Parameter(Mandatory = $true)][string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) {
    throw "Missing Linux binary: $Path"
  }

  $stream = [System.IO.File]::OpenRead($Path)
  try {
    $bytes = [byte[]]::new(4)
    $read = $stream.Read($bytes, 0, 4)
    if ($read -lt 4 -or $bytes[0] -ne 0x7F -or $bytes[1] -ne 0x45 -or $bytes[2] -ne 0x4C -or $bytes[3] -ne 0x46) {
      $header = '{0:X2} {1:X2} {2:X2} {3:X2}' -f $bytes[0], $bytes[1], $bytes[2], $bytes[3]
      throw "Build output is not a Linux ELF binary. Header bytes: $header"
    }
  }
  finally {
    $stream.Dispose()
  }
}

function Invoke-Checked {
  param(
    [Parameter(Mandatory = $true)][scriptblock]$Command,
    [Parameter(Mandatory = $true)][string]$Description
  )

  Write-Host "==> $Description"
  & $Command
  if ($LASTEXITCODE -ne 0) {
    throw "$Description failed with exit code $LASTEXITCODE"
  }
}

if ($SkipFrontendBuild -and $ForceFrontendBuild) {
  throw "Use either -SkipFrontendBuild or -ForceFrontendBuild, not both."
}

Assert-Command git
Assert-Command ssh
Assert-Command scp

if (-not (Test-Path -LiteralPath $SshConfigPath)) {
  throw "Missing SSH config: $SshConfigPath"
}

$shortSha = (& git -C $RootDir rev-parse --short HEAD).Trim()
if ([string]::IsNullOrWhiteSpace($shortSha)) {
  throw "Failed to read the current git short SHA."
}

if ($DryRun) {
  Write-Host "==> Dry run: no build, upload, remote compose, nginx, or container changes will be made."
  Write-Host "Root: $RootDir"
  Write-Host "Version: $shortSha"
  Write-Host "Remote app dir: $RemoteAppDir"
  Write-Host "Remote compose command will always include: docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml -f docker-compose.prod.bluegreen.yml"
  Write-Host "Blue/green ports: blue=127.0.0.1:3001 green=127.0.0.1:3002"
  Write-Host "Public smoke test: $PublicHealthUrl"
  exit 0
}

if (-not $SkipBuild) {
  $buildArgs = @{
    Version = $shortSha
    Output  = $BuildOutput
  }
  if ($SkipFrontendBuild) {
    $buildArgs.SkipFrontendBuild = $true
  }
  elseif (-not $ForceFrontendBuild) {
    $buildArgs.AutoSkipFrontendBuild = $true
  }

  Invoke-Checked -Description "Build Linux release binary" -Command {
    & $BuildScript @buildArgs
  }
}
else {
  Write-Host "==> Skipping local build"
}

Assert-ElfBinary -Path $BuildOutputPath

$remoteArchive = ""
if (-not $SkipSourceSync) {
  $status = (& git -C $RootDir status --porcelain)
  if (-not [string]::IsNullOrWhiteSpace(($status -join "`n"))) {
    throw "Working tree has uncommitted changes. Commit them before source sync, or use -SkipSourceSync to deploy only the binary."
  }

  New-Item -ItemType Directory -Force -Path $DeployTmpDir | Out-Null
  $localArchive = Join-Path $DeployTmpDir "new-api-deploy-$shortSha.tar.gz"
  $remoteArchive = "$RemoteTmpDir/new-api-deploy-$shortSha.tar.gz"

  Invoke-Checked -Description "Create source archive from HEAD" -Command {
    & git -C $RootDir archive --format=tar.gz -o $localArchive HEAD
  }
}

Invoke-Checked -Description "Ensure remote temp directory exists" -Command {
  & ssh -F $SshConfigPath $HostAlias "mkdir -p '$RemoteTmpDir'"
}

if (-not $SkipSourceSync) {
  Invoke-Checked -Description "Upload source archive" -Command {
    & scp -F $SshConfigPath $localArchive "${HostAlias}:$remoteArchive"
  }
}

$remoteBinary = "$RemoteTmpDir/new-api-linux-amd64-$shortSha"
Invoke-Checked -Description "Upload Linux binary" -Command {
  & scp -F $SshConfigPath $BuildOutputPath "${HostAlias}:$remoteBinary"
}

$sourceSyncValue = if ($SkipSourceSync) { "0" } else { "1" }
$remoteArchiveArg = if ([string]::IsNullOrEmpty($remoteArchive)) { "-" } else { $remoteArchive }
$remoteScript = @'
set -euo pipefail

app_dir="$1"
tmp_dir="$2"
image="$3"
archive="$4"
binary="$5"
short_sha="$6"
source_sync="$7"
health_attempts="$8"
health_sleep="$9"
drain_timeout="${10}"
public_health_url="${11}"
first_idle_color="${12}"

active_file="$app_dir/runtime-prod/active-color"
upstream_file="/etc/nginx/conf.d/new-api-bluegreen-upstream.conf"
site_files=(
  "/etc/nginx/sites-enabled/new-api.conf"
  "/etc/nginx/sites-enabled/pbroe-redirect.conf"
)
compose=(docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml -f docker-compose.prod.bluegreen.yml)

color_port() {
  case "$1" in
    blue) echo "3001" ;;
    green) echo "3002" ;;
    legacy) echo "3000" ;;
    *) echo "unknown color: $1" >&2; exit 1 ;;
  esac
}

color_service() {
  case "$1" in
    blue|green) echo "new-api-$1" ;;
    legacy) echo "new-api" ;;
    *) echo "unknown color: $1" >&2; exit 1 ;;
  esac
}

color_container() {
  case "$1" in
    blue|green) echo "new-api-$1" ;;
    legacy) echo "new-api-prod" ;;
    *) echo "unknown color: $1" >&2; exit 1 ;;
  esac
}

detect_active_color() {
  if [ -f "$active_file" ]; then
    color="$(tr -d '[:space:]' < "$active_file" || true)"
    if [ "$color" = "blue" ] || [ "$color" = "green" ]; then
      echo "$color"
      return
    fi
  fi
  if [ -f "$upstream_file" ]; then
    if grep -q "127.0.0.1:3001" "$upstream_file"; then
      echo "blue"
      return
    fi
    if grep -q "127.0.0.1:3002" "$upstream_file"; then
      echo "green"
      return
    fi
  fi
  if docker inspect -f '{{.State.Running}}' new-api-prod >/dev/null 2>&1; then
    if [ "$(docker inspect -f '{{.State.Running}}' new-api-prod)" = "true" ]; then
      echo "legacy"
      return
    fi
  fi
  echo "legacy"
}

write_active_color() {
  mkdir -p "$(dirname "$active_file")"
  tmp="$(mktemp "$active_file.XXXXXX")"
  printf '%s\n' "$1" > "$tmp"
  mv "$tmp" "$active_file"
}

write_upstream() {
  port="$1"
  tmp="$(mktemp "$upstream_file.XXXXXX")"
  cat > "$tmp" <<EOF
upstream new_api_bluegreen {
    server 127.0.0.1:$port;
    keepalive 32;
}
EOF
  mv "$tmp" "$upstream_file"
}

patch_nginx_sites() {
  patched=0
  ts="$(date +%Y%m%d-%H%M%S)"
  backup_dir="/etc/nginx/bluegreen-backups/$ts"
  mkdir -p "$backup_dir"
  for file in "${site_files[@]}"; do
    if [ ! -f "$file" ]; then
      continue
    fi
    cp "$file" "$backup_dir/$(basename "$file")"
    sed -i -E \
      -e 's#proxy_pass[[:space:]]+http://127\.0\.0\.1:3000;#proxy_pass http://new_api_bluegreen;#g' \
      -e 's#proxy_pass[[:space:]]+http://127\.0\.0\.1:3001;#proxy_pass http://new_api_bluegreen;#g' \
      -e 's#proxy_pass[[:space:]]+http://127\.0\.0\.1:3002;#proxy_pass http://new_api_bluegreen;#g' \
      "$file"
    if grep -q "proxy_pass http://new_api_bluegreen;" "$file"; then
      patched=$((patched + 1))
    fi
  done
  if [ "$patched" -eq 0 ]; then
    echo "No nginx site contains proxy_pass http://new_api_bluegreen; after patching." >&2
    return 1
  fi
}

reload_nginx_for_port() {
  port="$1"
  write_upstream "$port"
  patch_nginx_sites
  nginx -t
  nginx -s reload
}

wait_health() {
  url="$1"
  label="$2"
  echo "Waiting for $label health check: $url"
  for i in $(seq 1 "$health_attempts"); do
    if curl --silent --fail "$url" | grep -Eq '"success":[[:space:]]*true'; then
      echo "$label is healthy."
      return 0
    fi
    sleep "$health_sleep"
  done
  echo "$label health check failed: $url" >&2
  return 1
}

mkdir -p "$app_dir" "$tmp_dir" "$app_dir/runtime-prod" "$app_dir/logs-prod/blue" "$app_dir/logs-prod/green" /opt/new-api/backups/bin
cd "$app_dir"

if [ "$source_sync" = "1" ]; then
  tar -xzf "$archive" -C "$app_dir"
  chmod +x "$app_dir/deploy.sh" "$app_dir/backup.sh" "$app_dir/pull-prod-snapshot.sh" 2>/dev/null || true
fi

if [ ! -f .env.prod ]; then
  echo "Missing .env.prod in $app_dir" >&2
  exit 1
fi

echo "==> Compose SQL_DSN preview"
"${compose[@]}" config | grep 'SQL_DSN:' || {
  echo "Unable to preview SQL_DSN from compose config" >&2
  exit 1
}
if "${compose[@]}" config | grep -q 'change-me@postgres'; then
  echo "Refusing to deploy: compose config expanded SQL_DSN with change-me." >&2
  exit 1
fi

active="$(detect_active_color)"
if [ "$active" = "blue" ]; then
  idle="green"
elif [ "$active" = "green" ]; then
  idle="blue"
else
  idle="$first_idle_color"
fi
old_port="$(color_port "$active")"
new_port="$(color_port "$idle")"
idle_service="$(color_service "$idle")"
idle_container="$(color_container "$idle")"
old_container="$(color_container "$active")"

echo "Active color: $active ($old_port)"
echo "Idle color: $idle ($new_port)"

if [ ! -f "$active_file" ]; then
  write_active_color "$active"
fi

chmod 755 "$binary"
cp "$binary" "$app_dir/new-api-linux-amd64"

echo "==> Start idle $idle_service"
"${compose[@]}" up -d --no-deps --force-recreate "$idle_service"
docker cp "$binary" "$idle_container:/tmp/new-api.next"
docker exec "$idle_container" sh -c 'chmod 755 /tmp/new-api.next && mv /tmp/new-api.next /new-api'
docker restart --time "$drain_timeout" "$idle_container" >/dev/null
wait_health "http://127.0.0.1:$new_port/api/status" "$idle"

echo "==> Commit idle image tag $image"
docker commit "$idle_container" "$image" >/dev/null

previous_color="$active"
previous_port="$old_port"

echo "==> Switch nginx upstream to $idle"
if ! reload_nginx_for_port "$new_port"; then
  echo "Nginx switch failed; rolling back upstream." >&2
  reload_nginx_for_port "$previous_port" || true
  exit 1
fi

if ! wait_health "$public_health_url" "public endpoint"; then
  echo "Public smoke test failed; rolling back upstream." >&2
  reload_nginx_for_port "$previous_port" || true
  exit 1
fi

if [ "$active" = "legacy" ]; then
  if docker inspect "$old_container" >/dev/null 2>&1; then
    echo "==> Drain and stop legacy container $old_container"
    docker stop --time "$drain_timeout" "$old_container" >/dev/null || true
  fi
  write_active_color "$idle"
else
  write_active_color "$idle"
  if [ "$old_container" != "$idle_container" ] && docker inspect "$old_container" >/dev/null 2>&1; then
    echo "==> Drain and stop old container $old_container"
    docker stop --time "$drain_timeout" "$old_container" >/dev/null || true
  fi
fi

echo "Blue-green deployment succeeded: $idle ($short_sha)."
docker inspect -f '{{.Name}} {{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' "$idle_container"
'@

Write-Host "==> Run remote blue-green deployment"
$sshArgs = @(
  "-F", $SshConfigPath,
  $HostAlias,
  "bash", "-s", "--",
  $RemoteAppDir,
  $RemoteTmpDir,
  $ImageName,
  $remoteArchiveArg,
  $remoteBinary,
  $shortSha,
  $sourceSyncValue,
  "$HealthAttempts",
  "$HealthSleepSec",
  "$DrainTimeoutSec",
  $PublicHealthUrl,
  $FirstIdleColor
)

$remoteScript = $remoteScript -replace "`r`n", "`n"
$remoteScript | & ssh @sshArgs
if ($LASTEXITCODE -ne 0) {
  throw "Remote blue-green deployment failed with exit code $LASTEXITCODE"
}

$Stopwatch.Stop()
Write-Host ("Done in {0:n1}s." -f $Stopwatch.Elapsed.TotalSeconds)
