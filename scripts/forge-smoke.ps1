$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$Port = if ($env:FORGE_CORE_PORT) { [int]$env:FORGE_CORE_PORT } else { 18492 }
$DataDir = Join-Path ([System.IO.Path]::GetTempPath()) "forge-smoke.$([System.Guid]::NewGuid().ToString("N"))"
$WorkspaceDir = Join-Path ([System.IO.Path]::GetTempPath()) "forge-smoke-ws.$([System.Guid]::NewGuid().ToString("N"))"
$Log = Join-Path $DataDir "core.log"
$CoreProcess = $null
$StartedCore = $false
$Succeeded = $false

function Stop-ProcessTree {
  param([int]$ProcessId)
  if ($ProcessId -le 0) { return }
  $Process = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
  if (-not $Process) { return }
  & taskkill.exe /PID $ProcessId /T /F | Out-Null
}

function Stop-PortListeners {
  param([int]$LocalPort)
  $Connections = Get-NetTCPConnection -LocalPort $LocalPort -State Listen -ErrorAction SilentlyContinue
  $Pids = @($Connections | Select-Object -ExpandProperty OwningProcess -Unique)
  foreach ($Pid in $Pids) {
    if ($CoreProcess -and $Pid -eq $CoreProcess.Id) { continue }
    if (Get-Process -Id $Pid -ErrorAction SilentlyContinue) {
      Stop-ProcessTree -ProcessId $Pid
    }
  }
}

function Test-PortFree {
  param([int]$LocalPort)
  return -not (Get-NetTCPConnection -LocalPort $LocalPort -State Listen -ErrorAction SilentlyContinue)
}

function Invoke-Probe {
  param(
    [string]$Path,
    [int]$WantHttp = 200
  )
  $Uri = "http://127.0.0.1:$Port$Path"
  try {
    $Response = Invoke-WebRequest -Uri $Uri -UseBasicParsing -TimeoutSec 5
    $StatusCode = [int]$Response.StatusCode
    $Body = $Response.Content
  } catch {
    $Response = $_.Exception.Response
    $StatusCode = if ($Response) { [int]$Response.StatusCode } else { 0 }
    $Body = $_.Exception.Message
  }

  if ($StatusCode -ne $WantHttp) {
    [Console]::Error.WriteLine("FAIL  $Path -> http $StatusCode (expected $WantHttp)")
    if ($Body) {
      [Console]::Error.WriteLine($Body)
    }
    throw "probe failed"
  }
  Write-Host "ok    $Path -> $StatusCode"
}

try {
  New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
  New-Item -ItemType Directory -Force -Path $WorkspaceDir | Out-Null

  Write-Host "==> port $Port must be free"
  if (-not (Test-PortFree -LocalPort $Port)) {
    throw "FAIL  port $Port already in use"
  }

  Write-Host "==> starting forge-core (data=$DataDir)"
  & node (Join-Path $RepoRoot "scripts/check-vsa-files.mjs") --require-tracked
  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }

  $OldDataDir = $env:FORGE_DATA_DIR
  $OldWorkspaceDir = $env:FORGE_WORKSPACE_DIR
  $OldCorePort = $env:FORGE_CORE_PORT
  try {
    $env:FORGE_DATA_DIR = $DataDir
    $env:FORGE_WORKSPACE_DIR = $WorkspaceDir
    $env:FORGE_CORE_PORT = [string]$Port

    $CoreDir = Join-Path $RepoRoot "services/core"
    $CmdLine = "cd /d `"$CoreDir`" && go run . >> `"$Log`" 2>&1"
    $CoreProcess = Start-Process -FilePath "cmd.exe" -ArgumentList "/d", "/s", "/c", $CmdLine -PassThru -WindowStyle Hidden
    $StartedCore = $true
  } finally {
    $env:FORGE_DATA_DIR = $OldDataDir
    $env:FORGE_WORKSPACE_DIR = $OldWorkspaceDir
    $env:FORGE_CORE_PORT = $OldCorePort
  }

  Write-Host "==> waiting for /health (up to 30s)"
  for ($i = 1; $i -le 60; $i++) {
    try {
      Invoke-WebRequest -Uri "http://127.0.0.1:$Port/health" -UseBasicParsing -TimeoutSec 2 | Out-Null
      Write-Host "ok    /health after $i attempts"
      break
    } catch {
      if (-not (Get-Process -Id $CoreProcess.Id -ErrorAction SilentlyContinue)) {
        throw "FAIL  core process exited before becoming healthy"
      }
      Start-Sleep -Milliseconds 500
      if ($i -eq 60) {
        throw "FAIL  /health did not respond in 30s"
      }
    }
  }

  Write-Host "==> probing endpoints"
  Invoke-Probe "/health" 200
  Invoke-Probe "/api/meta" 200
  Invoke-Probe "/api/autonomy/status" 200
  Invoke-Probe "/api/telegram/status" 200
  Invoke-Probe "/api/discord/status" 200
  Invoke-Probe "/api/adapters" 200
  Invoke-Probe "/api/jobs" 200

  Write-Host "==> shutting down"
  if ($CoreProcess) {
    Stop-ProcessTree -ProcessId $CoreProcess.Id
    $CoreProcess = $null
  }

  $Succeeded = $true
  Write-Host "==> smoke OK"
} catch {
  [Console]::Error.WriteLine($_.Exception.Message)
  exit 1
} finally {
  if ($CoreProcess) {
    Stop-ProcessTree -ProcessId $CoreProcess.Id
  }
  if ($StartedCore) {
    Stop-PortListeners -LocalPort $Port
  }
  if (-not $Succeeded) {
    [Console]::Error.WriteLine("---- smoke failed; core.log tail ----")
    if (Test-Path $Log) {
      Get-Content -Tail 40 $Log -ErrorAction SilentlyContinue | ForEach-Object {
        [Console]::Error.WriteLine($_)
      }
    } else {
      [Console]::Error.WriteLine("(core.log not created; preflight likely failed before core start)")
    }
  }
  Remove-Item -LiteralPath $DataDir -Recurse -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath $WorkspaceDir -Recurse -Force -ErrorAction SilentlyContinue
}
