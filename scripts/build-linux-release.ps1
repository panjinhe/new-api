[CmdletBinding()]
param(
  [string]$Version = "",
  [string]$Output = "new-api-linux-amd64",
  [switch]$SkipFrontendBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$WebDir = Join-Path $RootDir "web"
$OutputPath = Join-Path $RootDir $Output

function Get-ShortGitSha {
  $sha = (& git -C $RootDir rev-parse --short HEAD).Trim()
  if ([string]::IsNullOrWhiteSpace($sha)) {
    throw "Failed to read the current git short SHA."
  }
  return $sha
}

function Assert-ElfBinary {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  $bytes = [System.IO.File]::ReadAllBytes($Path)
  if ($bytes.Length -lt 4) {
    throw "Build artifact is too small to identify its file type: $Path"
  }

  $isElf = $bytes[0] -eq 0x7F -and $bytes[1] -eq 0x45 -and $bytes[2] -eq 0x4C -and $bytes[3] -eq 0x46
  if (-not $isElf) {
    $header = '{0:X2} {1:X2} {2:X2} {3:X2}' -f $bytes[0], $bytes[1], $bytes[2], $bytes[3]
    throw "Build output is not a Linux ELF binary. Header bytes: $header. Do not deploy a wrong-platform artifact into a Linux container."
  }
}

if ([string]::IsNullOrWhiteSpace($Version)) {
  $Version = Get-ShortGitSha
}

Write-Host "Root: $RootDir"
Write-Host "Version: $Version"
Write-Host "Output: $OutputPath"

if (-not $SkipFrontendBuild) {
  Write-Host "==> Rebuilding frontend with bun"
  Push-Location $WebDir
  try {
    & bun run build
  }
  finally {
    Pop-Location
  }
}

$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
$oldCgoEnabled = $env:CGO_ENABLED
$oldGoexperiment = $env:GOEXPERIMENT

try {
  $env:GOOS = "linux"
  $env:GOARCH = "amd64"
  $env:CGO_ENABLED = "0"
  $env:GOEXPERIMENT = "greenteagc"

  Write-Host "==> Building Linux amd64 binary"
  & go build `
    -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$Version" `
    -o $OutputPath
}
finally {
  if ($null -ne $oldGoos) { $env:GOOS = $oldGoos } else { Remove-Item Env:GOOS -ErrorAction SilentlyContinue }
  if ($null -ne $oldGoarch) { $env:GOARCH = $oldGoarch } else { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue }
  if ($null -ne $oldCgoEnabled) { $env:CGO_ENABLED = $oldCgoEnabled } else { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue }
  if ($null -ne $oldGoexperiment) { $env:GOEXPERIMENT = $oldGoexperiment } else { Remove-Item Env:GOEXPERIMENT -ErrorAction SilentlyContinue }
}

Write-Host "==> Verifying ELF header"
Assert-ElfBinary -Path $OutputPath

$artifact = Get-Item $OutputPath
Write-Host ("Done. Built Linux ELF binary: {0} ({1} bytes)" -f $artifact.FullName, $artifact.Length)
