$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$CorePort = if ($env:FORGE_CORE_PORT) { $env:FORGE_CORE_PORT } else { "18492" }
$CoreUrl = if ($env:VITE_FORGE_API_URL) { $env:VITE_FORGE_API_URL } else { "http://127.0.0.1:$CorePort" }

Set-Location $RootDir

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
