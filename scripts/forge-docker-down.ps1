$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$EnvFile = if ($env:FORGE_DOCKER_ENV_FILE) { $env:FORGE_DOCKER_ENV_FILE } else { Join-Path $RootDir ".env.docker" }

Set-Location $RootDir

$ComposeArgs = @()
$IgpuMode = if ($env:FORGE_DOCKER_IGPU) { $env:FORGE_DOCKER_IGPU } else { "auto" }
if ($IgpuMode -eq "1" -or $IgpuMode -eq "true") {
  throw "FORGE_DOCKER_IGPU requested, but Windows host iGPU compose passthrough is not supported by this script."
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

Write-Host "Stopping FORGE Docker stack without deleting volumes..."
& docker compose @ComposeArgs @ArgsList down --remove-orphans
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}

Write-Host ""
Write-Host "FORGE Docker stack stopped. Named volumes were preserved."
Write-Host "To erase databases intentionally, run: docker compose down -v"
