[CmdletBinding()]
param(
  [string]$HostAlias = "aheapi-itdun",
  [string]$SshConfig = "ops/ssh/config.local",
  [string]$RemoteAppDir = "/opt/new-api/app",
  [string]$RemoteTmpDir = "/opt/new-api/tmp",
  [string]$ContainerName = "new-api-prod",
  [string]$ImageName = "new-api-local:prod",
  [string]$HealthUrl = "http://127.0.0.1:3000/api/status",
  [string]$BuildOutput = "new-api-linux-amd64",
  [int]$RestartTimeoutSec = 120,
  [switch]$SkipBuild,
  [switch]$SkipFrontendBuild,
  [switch]$ForceFrontendBuild,
  [switch]$SkipSourceSync
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
  param(
    [Parameter(Mandatory = $true)]
    [string]$Name
  )

  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required command not found: $Name"
  }
}

function Assert-ElfBinary {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

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
    [Parameter(Mandatory = $true)]
    [scriptblock]$Command,
    [Parameter(Mandatory = $true)]
    [string]$Description
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

if (-not $SkipBuild) {
  $shortSha = (& git -C $RootDir rev-parse --short HEAD).Trim()
  if ([string]::IsNullOrWhiteSpace($shortSha)) {
    throw "Failed to read the current git short SHA."
  }

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

if (-not (Get-Variable -Name shortSha -Scope Local -ErrorAction SilentlyContinue)) {
  $shortSha = (& git -C $RootDir rev-parse --short HEAD).Trim()
  if ([string]::IsNullOrWhiteSpace($shortSha)) {
    throw "Failed to read the current git short SHA."
  }
}

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
container="$3"
image="$4"
archive="$5"
binary="$6"
health_url="$7"
restart_timeout="$8"
source_sync="$9"

mkdir -p "$app_dir" "$tmp_dir" /opt/new-api/backups/bin

if [ "$source_sync" = "1" ]; then
  tar -xzf "$archive" -C "$app_dir"
  chmod +x "$app_dir/deploy.sh" "$app_dir/backup.sh" "$app_dir/pull-prod-snapshot.sh" 2>/dev/null || true
fi

ts="$(date +%Y%m%d-%H%M%S)"
docker cp "$container:/new-api" "/opt/new-api/backups/bin/new-api-$ts" >/dev/null 2>&1 || true

chmod 755 "$binary"
cp "$binary" "$app_dir/new-api-linux-amd64"
docker cp "$binary" "$container:/tmp/new-api.next"
docker exec "$container" sh -c 'chmod 755 /tmp/new-api.next && mv /tmp/new-api.next /new-api'
docker commit "$container" "$image" >/dev/null
docker restart --time "$restart_timeout" "$container" >/dev/null

echo "Waiting for health check: $health_url"
for i in $(seq 1 30); do
  if command -v curl >/dev/null 2>&1; then
    if curl --silent --fail "$health_url" | grep -Eq '"success":[[:space:]]*true'; then
      echo "Deployment succeeded."
      docker inspect -f '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' "$container"
      exit 0
    fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -q -O - "$health_url" | grep -Eq '"success":[[:space:]]*true'; then
      echo "Deployment succeeded."
      docker inspect -f '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' "$container"
      exit 0
    fi
  else
    echo "Deployment finished; no curl/wget available for health check."
    exit 0
  fi
  sleep 2
done

echo "Health check failed. Recent logs:" >&2
docker logs --tail 100 "$container" >&2 || true
exit 1
'@

Write-Host "==> Replace binary and restart remote container"
$sshArgs = @(
  "-F", $SshConfigPath,
  $HostAlias,
  "bash", "-s", "--",
  $RemoteAppDir,
  $RemoteTmpDir,
  $ContainerName,
  $ImageName,
  $remoteArchiveArg,
  $remoteBinary,
  $HealthUrl,
  "$RestartTimeoutSec",
  $sourceSyncValue
)

$remoteScript = $remoteScript -replace "`r`n", "`n"
$remoteScript | & ssh @sshArgs
if ($LASTEXITCODE -ne 0) {
  throw "Remote fast deployment failed with exit code $LASTEXITCODE"
}

$Stopwatch.Stop()
Write-Host ("Done in {0:n1}s." -f $Stopwatch.Elapsed.TotalSeconds)
