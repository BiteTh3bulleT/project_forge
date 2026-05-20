$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$EnvFile = if ($env:FORGE_DOCKER_ENV_FILE) { $env:FORGE_DOCKER_ENV_FILE } else { Join-Path $RootDir ".env.docker" }
$TokenFile = Join-Path $RootDir ".forge/docker-api-token"

Set-Location $RootDir

if ([string]::IsNullOrWhiteSpace($env:FORGE_API_TOKEN)) {
  $TokenDir = Split-Path -Parent $TokenFile
  New-Item -ItemType Directory -Force -Path $TokenDir | Out-Null
  if (-not (Test-Path $TokenFile) -or (Get-Item $TokenFile).Length -eq 0) {
    $Bytes = [byte[]]::new(32)
    [System.Security.Cryptography.RandomNumberGenerator]::Fill($Bytes)
    $Token = -join ($Bytes | ForEach-Object { $_.ToString("x2") })
    Set-Content -LiteralPath $TokenFile -Value $Token -NoNewline
  }
  $env:FORGE_API_TOKEN = (Get-Content -Raw -LiteralPath $TokenFile).Trim()
}

$ComposeArgs = @()
$IgpuMode = if ($env:FORGE_DOCKER_IGPU) { $env:FORGE_DOCKER_IGPU } else { "auto" }
if ($IgpuMode -eq "1" -or $IgpuMode -eq "true") {
  throw "FORGE_DOCKER_IGPU requested, but Windows host iGPU compose passthrough is not supported by this script."
}

function Test-EnvSet {
  param([string]$Name)
  return -not [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($Name, "Process"))
}

function Test-EnvFileSet {
  param([string]$Name)
  if (-not (Test-Path -LiteralPath $EnvFile)) {
    return $false
  }
  return Select-String -LiteralPath $EnvFile -Pattern "^\s*(export\s+)?$([regex]::Escape($Name))=" -Quiet
}

function Set-DefaultEnv {
  param(
    [string]$Name,
    [string]$Value
  )
  if (-not (Test-EnvSet $Name) -and -not (Test-EnvFileSet $Name)) {
    [Environment]::SetEnvironmentVariable($Name, $Value, "Process")
  }
}

function Get-OllamaProbeUrl {
  $ContainerUrl = if (Test-EnvSet "OLLAMA_BASE_URL") { $env:OLLAMA_BASE_URL } else { "http://host.docker.internal:11434" }
  $ProbeUrl = if (Test-EnvSet "FORGE_DOCKER_OLLAMA_PROBE_URL") { $env:FORGE_DOCKER_OLLAMA_PROBE_URL } else { $ContainerUrl }
  if ($ProbeUrl -match "host\.docker\.internal") {
    $ProbeUrl = "http://127.0.0.1:11434"
  }
  return $ProbeUrl.TrimEnd("/")
}

function Enable-DockerOllamaDefaults {
  if ($env:FORGE_DISABLE_OLLAMA_AUTODETECT -eq "true") {
    return
  }

  Set-DefaultEnv "OLLAMA_BASE_URL" "http://host.docker.internal:11434"
  Set-DefaultEnv "FORGE_ENABLE_MODEL_RUNTIME" "true"
  Set-DefaultEnv "FORGE_MODEL_DEFAULT_BACKEND" "ollama_compat"

  if ((Test-EnvSet "FORGE_MODEL_DEFAULT_ID") -and -not (Test-EnvSet "OLLAMA_MODEL")) {
    $env:OLLAMA_MODEL = $env:FORGE_MODEL_DEFAULT_ID
    return
  }
  if ((Test-EnvSet "OLLAMA_MODEL") -and -not (Test-EnvSet "FORGE_MODEL_DEFAULT_ID")) {
    $env:FORGE_MODEL_DEFAULT_ID = $env:OLLAMA_MODEL
    return
  }
  if ((Test-EnvSet "FORGE_MODEL_DEFAULT_ID") -or (Test-EnvSet "OLLAMA_MODEL")) {
    return
  }

  $ProbeUrl = Get-OllamaProbeUrl
  try {
    $ModelsPayload = Invoke-RestMethod -Uri "$ProbeUrl/v1/models" -TimeoutSec 2
  } catch {
    return
  }

  $Ids = @()
  if ($ModelsPayload.data) {
    $Ids = @($ModelsPayload.data | ForEach-Object { $_.id } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
  }
  $Preferred = @(
    "phi4-mini:latest",
    "llama3.2:latest",
    "llama3.2:3b",
    "phi3:3.8b",
    "llama3.1:8b",
    "qwen2.5-coder:7b",
    "mistral:latest",
    "qwen2.5:14b",
    "qwen3.6:latest"
  )
  $Selected = $Preferred | Where-Object { $Ids -contains $_ } | Select-Object -First 1
  if (-not $Selected) {
    $Selected = $Ids | Where-Object { $_ -notlike "*embed*" -and $_ -notlike "*cloud*" } | Select-Object -First 1
  }
  if ($Selected) {
    $env:FORGE_MODEL_DEFAULT_ID = $Selected
    $env:OLLAMA_MODEL = $Selected
    Write-Host "FORGE Docker model default selected from host Ollama: $Selected"
  }
}

Set-DefaultEnv "FORGE_CORE_BIND_HOST" "0.0.0.0"
Set-DefaultEnv "FORGE_ALLOW_WILDCARD_BIND" "true"

function Test-DockerEngineReady {
  try {
    & docker version --format "{{.Server.Version}}" *> $null
    return $LASTEXITCODE -eq 0
  } catch {
    return $false
  }
}

function Get-DockerDesktopExecutable {
  $Candidates = @(
    (Join-Path $env:ProgramFiles "Docker\Docker\Docker Desktop.exe"),
    (Join-Path $env:ProgramFiles "Docker\Docker\frontend\Docker Desktop.exe"),
    (Join-Path $env:LOCALAPPDATA "Docker\Docker Desktop.exe")
  )

  foreach ($Candidate in $Candidates) {
    if ($Candidate -and (Test-Path $Candidate)) {
      return $Candidate
    }
  }
  return $null
}

function Wait-DockerEngineReady {
  if (Test-DockerEngineReady) {
    return
  }

  $TimeoutSeconds = 90
  if ($env:FORGE_DOCKER_READY_TIMEOUT_SECONDS) {
    $TimeoutSeconds = [int]$env:FORGE_DOCKER_READY_TIMEOUT_SECONDS
  }
  $AutoStart = -not ($env:FORGE_DOCKER_AUTO_START -eq "0" -or $env:FORGE_DOCKER_AUTO_START -eq "false")

  Write-Host "Docker engine is not ready yet."
  if ($AutoStart) {
    $Service = Get-Service com.docker.service -ErrorAction SilentlyContinue
    if ($Service -and $Service.Status -ne "Running") {
      try {
        Start-Service com.docker.service -ErrorAction Stop
        Write-Host "Requested Docker Windows service start."
      } catch {
        Write-Host "Docker Windows service could not be started from this shell: $($_.Exception.Message)"
      }
    }

    $DockerDesktop = Get-DockerDesktopExecutable
    if ($DockerDesktop) {
      try {
        Start-Process -FilePath $DockerDesktop -WindowStyle Hidden | Out-Null
        Write-Host "Requested Docker Desktop start."
      } catch {
        Write-Host "Docker Desktop could not be started from this shell: $($_.Exception.Message)"
      }
    }
  }

  Write-Host "Waiting up to $TimeoutSeconds seconds for Docker engine readiness..."
  $Deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $Deadline) {
    Start-Sleep -Seconds 2
    if (Test-DockerEngineReady) {
      Write-Host "Docker engine is ready."
      return
    }
  }

  throw @"
Docker engine is not reachable.

Docker Desktop may be open, but the Linux engine is not running. Current Docker context must be usable before FORGE containers can start.

Try one of these:
- Open Docker Desktop and wait until it reports "Engine running".
- Run: wsl -l -v
- Confirm docker-desktop is Running, then retry: npm run docker:desktop

Set FORGE_DOCKER_READY_TIMEOUT_SECONDS to wait longer, or FORGE_DOCKER_AUTO_START=0 to disable the auto-start attempt.
"@
}

$ArgsList = @()
if (Test-Path $EnvFile) {
  $ArgsList += @("--env-file", $EnvFile)
}

if (-not [string]::IsNullOrWhiteSpace($env:FORGE_DOCKER_PROFILES)) {
  foreach ($Profile in ($env:FORGE_DOCKER_PROFILES -split ",")) {
    $Trimmed = $Profile.Trim()
    if ($Trimmed) {
      $ArgsList += @("--profile", $Trimmed)
    }
  }
}

$Services = @($args)
if ($Services.Count -eq 0) {
  $Services = @("postgres", "redis", "qdrant", "core")
}

function Test-ServiceRequested {
  param([string]$Name)
  return $Services -contains $Name
}

function Open-UrlBestEffort {
  param([string]$Url)
  if ($env:FORGE_DOCKER_OPEN -eq "0" -or $env:FORGE_DOCKER_OPEN -eq "false") {
    Write-Host "Auto-open disabled. Open $Url manually."
    return
  }
  try {
    Start-Process $Url | Out-Null
    Write-Host "Opening Docker desktop web surface: $Url"
  } catch {
    Write-Host "Docker desktop web surface is available at $Url"
  }
}

Write-Host "Starting FORGE Docker stack without deleting volumes..."
Write-Host "Env file: $(if (Test-Path $EnvFile) { $EnvFile } else { "(none)" })"
Write-Host "Services: $($Services -join " ")"
Write-Host "Intel iGPU telemetry: not enabled"

Enable-DockerOllamaDefaults
Wait-DockerEngineReady

& docker compose @ComposeArgs @ArgsList up -d --build @Services
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}

Write-Host ""
& docker compose @ComposeArgs @ArgsList ps
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}
Write-Host ""
Write-Host "FORGE Docker stack started."
if (Test-ServiceRequested "desktop-web") {
  $DesktopPort = if ($env:FORGE_DESKTOP_PORT) { $env:FORGE_DESKTOP_PORT } else { "1420" }
  Open-UrlBestEffort "http://127.0.0.1:$DesktopPort/#/dashboard"
}
Write-Host "Use '.\scripts\forge-docker-down.ps1' or 'npm run docker:stop' to stop containers without deleting databases."
