$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$RunDir = Join-Path $RootDir ".forge/run"
$CorePidFile = Join-Path $RunDir "core.pid"
$DesktopPidFile = Join-Path $RunDir "desktop.pid"

function Test-ProcessRunning {
  param([int]$ProcessId)
  if ($ProcessId -le 0) { return $false }
  return $null -ne (Get-Process -Id $ProcessId -ErrorAction SilentlyContinue)
}

function Stop-Tree {
  param([int]$ProcessId)
  if (-not (Test-ProcessRunning $ProcessId)) { return }
  & taskkill /PID $ProcessId /T /F | Out-Null
}

function Stop-FromPidFile {
  param(
    [string]$Name,
    [string]$PidFile
  )

  if (-not (Test-Path $PidFile)) {
    Write-Host "$Name not tracked (no pid file)."
    return
  }

  $PidText = (Get-Content -Raw $PidFile -ErrorAction SilentlyContinue).Trim()
  if (-not ($PidText -match "^\d+$")) {
    Remove-Item -Force $PidFile
    Write-Host "$Name pid file was invalid; cleaned."
    return
  }

  $ProcessId = [int]$PidText
  if (Test-ProcessRunning $ProcessId) {
    Write-Host "Stopping $Name (pid $ProcessId)..."
    Stop-Tree -ProcessId $ProcessId
    Write-Host "$Name stopped."
  } else {
    Write-Host "$Name already stopped (stale pid $ProcessId)."
  }

  Remove-Item -Force $PidFile
}

function Stop-PortListeners {
  param([int]$Port)
  $Lines = netstat -ano | Select-String -Pattern "[:\.]$Port\s"
  if (-not $Lines) { return }

  $Pids = @()
  foreach ($Line in $Lines) {
    $Parts = ($Line.ToString() -split "\s+") | Where-Object { $_ -ne "" }
    if ($Parts.Count -ge 5 -and $Parts[-1] -match "^\d+$") {
      $Pids += [int]$Parts[-1]
    }
  }

  $Pids = $Pids | Sort-Object -Unique
  foreach ($ProcessId in $Pids) {
    if (Test-ProcessRunning $ProcessId) {
      Write-Host "Stopping listener on :$Port (pid $ProcessId)..."
      Stop-Tree -ProcessId $ProcessId
    }
  }
}

Stop-FromPidFile -Name "desktop" -PidFile $DesktopPidFile
Stop-FromPidFile -Name "core" -PidFile $CorePidFile
Stop-PortListeners -Port 1420
Stop-PortListeners -Port 18492

Get-Process -Name "forge_desktop" -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host "Stopping forge_desktop process (pid $($_.Id))..."
  Stop-Tree -ProcessId $_.Id
}

Write-Host "FORGE stopped."
