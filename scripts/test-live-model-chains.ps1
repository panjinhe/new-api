[CmdletBinding()]
param(
  [string]$BaseUrl = "",
  [string]$ApiKey = "",
  [string[]]$OpenAIModels = @(),
  [string]$ClaudeModel = "",
  [string]$ClaudeExpectedModel = "",
  [int]$Count = 1
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot

function Get-EffectiveValue {
  param(
    [string]$Preferred,
    [Parameter(Mandatory = $true)]
    [string]$EnvName
  )

  if (-not [string]::IsNullOrWhiteSpace($Preferred)) {
    return $Preferred.Trim()
  }

  $current = [Environment]::GetEnvironmentVariable($EnvName)
  if ([string]::IsNullOrWhiteSpace($current)) {
    return ""
  }
  return $current.Trim()
}

function Mask-Secret {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Value
  )

  if ($Value.Length -le 8) {
    return "********"
  }

  return "{0}...{1}" -f $Value.Substring(0, 4), $Value.Substring($Value.Length - 4)
}

function Set-Or-ClearEnv {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Name,
    [string]$Value
  )

  if ([string]::IsNullOrWhiteSpace($Value)) {
    Remove-Item "Env:$Name" -ErrorAction SilentlyContinue
    return
  }

  Set-Item "Env:$Name" $Value
}

$effectiveBaseUrl = Get-EffectiveValue -Preferred $BaseUrl -EnvName "NEWAPI_LIVE_BASE_URL"
$effectiveApiKey = Get-EffectiveValue -Preferred $ApiKey -EnvName "NEWAPI_LIVE_API_KEY"
$effectiveClaudeModel = Get-EffectiveValue -Preferred $ClaudeModel -EnvName "NEWAPI_LIVE_CLAUDE_MODEL"
$effectiveClaudeExpectedModel = Get-EffectiveValue -Preferred $ClaudeExpectedModel -EnvName "NEWAPI_LIVE_CLAUDE_EXPECTED_MODEL"

$effectiveOpenAIModels = @()
if ($OpenAIModels.Count -gt 0) {
  $effectiveOpenAIModels = $OpenAIModels | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
} else {
  $fromEnv = Get-EffectiveValue -Preferred "" -EnvName "NEWAPI_LIVE_OPENAI_MODELS"
  if (-not [string]::IsNullOrWhiteSpace($fromEnv)) {
    $effectiveOpenAIModels = $fromEnv.Split(",") | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
  }
}

if ([string]::IsNullOrWhiteSpace($effectiveBaseUrl) -or [string]::IsNullOrWhiteSpace($effectiveApiKey)) {
  throw "BaseUrl and ApiKey are required. Pass -BaseUrl/-ApiKey or set NEWAPI_LIVE_BASE_URL and NEWAPI_LIVE_API_KEY."
}

Write-Host "Root: $RootDir"
Write-Host "Base URL: $effectiveBaseUrl"
Write-Host "API Key: $(Mask-Secret -Value $effectiveApiKey)"
if ($effectiveOpenAIModels.Count -gt 0) {
  Write-Host "OpenAI models: $($effectiveOpenAIModels -join ', ')"
}
if (-not [string]::IsNullOrWhiteSpace($effectiveClaudeModel)) {
  Write-Host "Claude model: $effectiveClaudeModel"
}
if (-not [string]::IsNullOrWhiteSpace($effectiveClaudeExpectedModel)) {
  Write-Host "Claude expected model: $effectiveClaudeExpectedModel"
}

$oldBaseUrl = [Environment]::GetEnvironmentVariable("NEWAPI_LIVE_BASE_URL")
$oldApiKey = [Environment]::GetEnvironmentVariable("NEWAPI_LIVE_API_KEY")
$oldOpenAIModels = [Environment]::GetEnvironmentVariable("NEWAPI_LIVE_OPENAI_MODELS")
$oldClaudeModel = [Environment]::GetEnvironmentVariable("NEWAPI_LIVE_CLAUDE_MODEL")
$oldClaudeExpectedModel = [Environment]::GetEnvironmentVariable("NEWAPI_LIVE_CLAUDE_EXPECTED_MODEL")

try {
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_BASE_URL" -Value $effectiveBaseUrl
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_API_KEY" -Value $effectiveApiKey
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_OPENAI_MODELS" -Value ($effectiveOpenAIModels -join ",")
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_CLAUDE_MODEL" -Value $effectiveClaudeModel
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_CLAUDE_EXPECTED_MODEL" -Value $effectiveClaudeExpectedModel

  Push-Location $RootDir
  try {
    & go test ./integration -run TestLiveModelChains ("-count=$Count") -v
  }
  finally {
    Pop-Location
  }
}
finally {
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_BASE_URL" -Value $oldBaseUrl
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_API_KEY" -Value $oldApiKey
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_OPENAI_MODELS" -Value $oldOpenAIModels
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_CLAUDE_MODEL" -Value $oldClaudeModel
  Set-Or-ClearEnv -Name "NEWAPI_LIVE_CLAUDE_EXPECTED_MODEL" -Value $oldClaudeExpectedModel
}
