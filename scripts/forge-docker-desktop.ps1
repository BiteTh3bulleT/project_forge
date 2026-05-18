$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$CorePort = if ($env:FORGE_CORE_PORT) { $env:FORGE_CORE_PORT } else { "18492" }
$CoreUrl = if ($env:VITE_FORGE_API_URL) { $env:VITE_FORGE_API_URL } else { "http://127.0.0.1:$CorePort" }
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

Write-Host "Starting Docker-backed FORGE services first..."
$env:FORGE_CORE_PORT = $CorePort
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $RootDir "scripts/forge-docker-up.ps1") postgres redis qdrant core
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}

Write-Host ""
Write-Host "Ensuring browser-served desktop-web is not holding the Tauri dev port..."
& docker compose stop desktop-web *> $null

Write-Host ""
Write-Host "Launching native Tauri desktop shell against $CoreUrl"
Write-Host "The desktop shell runs on the host; Docker provides core and data services."

$env:VITE_FORGE_API_URL = $CoreUrl
& npm run desktop
exit $LASTEXITCODE
