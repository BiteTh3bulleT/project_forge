$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

function Test-EnvSet {
  param([string]$Name)
  return -not [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($Name, "Process"))
}

function Set-DefaultEnv {
  param(
    [string]$Name,
    [string]$Value
  )
  if (-not (Test-EnvSet $Name)) {
    [Environment]::SetEnvironmentVariable($Name, $Value, "Process")
  }
}

function Enable-LocalOllamaDefaults {
  if ($env:FORGE_DISABLE_OLLAMA_AUTODETECT -eq "true") {
    return
  }

  $explicitRuntime = @(
    "FORGE_ENABLE_MODEL_RUNTIME",
    "FORGE_MODEL_OPENAI_COMPAT_ENDPOINT",
    "FORGE_MODEL_VLLM_ENDPOINT",
    "FORGE_LLAMA_CPP_ENDPOINT",
    "FORGE_LLAMA_CPP_BINARY_PATH"
  ) | Where-Object { Test-EnvSet $_ }
  if ($explicitRuntime.Count -gt 0) {
    return
  }

  $OllamaUrl = if (Test-EnvSet "OLLAMA_BASE_URL") { $env:OLLAMA_BASE_URL } else { "http://127.0.0.1:11434" }
  $OllamaUrl = $OllamaUrl.TrimEnd("/")

  try {
    $ModelsPayload = Invoke-RestMethod -Uri "$OllamaUrl/v1/models" -TimeoutSec 2
  } catch {
    return
  }

  Set-DefaultEnv "FORGE_ENABLE_MODEL_RUNTIME" "true"
  Set-DefaultEnv "FORGE_MODEL_OPENAI_COMPAT_ENDPOINT" $OllamaUrl
  Set-DefaultEnv "FORGE_MODEL_DEFAULT_BACKEND" "openai_compat"
  Set-DefaultEnv "FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD" "true"
  Set-DefaultEnv "FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD" "false"
  Set-DefaultEnv "FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE" "false"
  Set-DefaultEnv "OLLAMA_BASE_URL" $OllamaUrl

  if (-not (Test-EnvSet "FORGE_MODEL_DEFAULT_ID") -and -not (Test-EnvSet "OLLAMA_MODEL")) {
    $Ids = @()
    if ($ModelsPayload.data) {
      $Ids = @($ModelsPayload.data | ForEach-Object { $_.id } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    }
    $Preferred = @(
      "phi3:3.8b",
      "llama3.1:8b",
      "llama3.2:3b",
      "mistral:latest",
      "qwen2.5-coder:7b",
      "qwen2.5:14b",
      "qwen3.6:latest",
      "qwen3-coder:480b-cloud"
    )
    $Selected = $Preferred | Where-Object { $Ids -contains $_ } | Select-Object -First 1
    if (-not $Selected) {
      $Selected = $Ids | Where-Object { $_ -notlike "*embed*" } | Select-Object -First 1
    }
    if ($Selected) {
      $env:FORGE_MODEL_DEFAULT_ID = $Selected
      $env:OLLAMA_MODEL = $Selected
    }
  } else {
    if (-not (Test-EnvSet "FORGE_MODEL_DEFAULT_ID")) {
      $env:FORGE_MODEL_DEFAULT_ID = $env:OLLAMA_MODEL
    }
    if (-not (Test-EnvSet "OLLAMA_MODEL")) {
      $env:OLLAMA_MODEL = $env:FORGE_MODEL_DEFAULT_ID
    }
  }

  $Message = "FORGE modelruntime auto-enabled via local Ollama at $OllamaUrl"
  if (Test-EnvSet "FORGE_MODEL_DEFAULT_ID") {
    $Message += " (default model: $env:FORGE_MODEL_DEFAULT_ID)"
  }
  Write-Host $Message
}

& node (Join-Path $RootDir "scripts/check-vsa-files.mjs") --require-tracked
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}

Enable-LocalOllamaDefaults

Set-DefaultEnv "FORGE_ENABLE_MODEL_RUNTIME" "true"
Set-Location (Join-Path $RootDir "services/core")
& go run .
exit $LASTEXITCODE
