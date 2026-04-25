$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$RunDir = Join-Path $RootDir ".forge/run"
$LogDir = Join-Path $RootDir ".forge/logs"
$CorePidFile = Join-Path $RunDir "core.pid"
$DesktopPidFile = Join-Path $RunDir "desktop.pid"
$CoreLog = Join-Path $LogDir "core.log"
$DesktopLog = Join-Path $LogDir "desktop.log"
$CoreUrl = "http://127.0.0.1:18492/health"
$DesktopPort = 1420

New-Item -ItemType Directory -Force -Path $RunDir | Out-Null
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

function Test-ProcessRunning {
  param([int]$ProcessId)
  if ($ProcessId -le 0) { return $false }
  return $null -ne (Get-Process -Id $ProcessId -ErrorAction SilentlyContinue)
}

function Start-IfNeeded {
  param(
    [string]$Name,
    [string]$PidFile,
    [string]$LogFile,
    [string]$Command
  )

  if (Test-Path $PidFile) {
    $ExistingText = (Get-Content -Raw $PidFile -ErrorAction SilentlyContinue).Trim()
    if ($ExistingText -match "^\d+$") {
      $Existing = [int]$ExistingText
      if (Test-ProcessRunning $Existing) {
        Write-Host "$Name already running (pid $Existing)"
        return
      }
    }
    Remove-Item -Force $PidFile
  }

  Write-Host "Starting $Name..."
  $CmdLine = "cd /d `"$RootDir`" && $Command >> `"$LogFile`" 2>&1"
  $Proc = Start-Process -FilePath "cmd.exe" -ArgumentList "/d", "/s", "/c", $CmdLine -PassThru
  Set-Content -Path $PidFile -Value $Proc.Id -NoNewline

  Start-Sleep -Milliseconds 200
  if (-not (Test-ProcessRunning $Proc.Id)) {
    throw "Failed to start $Name. Check $LogFile"
  }
  Write-Host "$Name started (pid $($Proc.Id))"
}

function Wait-ForCore {
  $Attempts = 40
  $DelayMs = 500

  Write-Host "Waiting for core health..."
  for ($i = 0; $i -lt $Attempts; $i++) {
    try {
      Invoke-WebRequest -Uri $CoreUrl -UseBasicParsing -TimeoutSec 2 | Out-Null
      Write-Host "Core is healthy."
      return
    } catch {
      Start-Sleep -Milliseconds $DelayMs
    }
  }
  throw "Core did not become healthy in time. Check $CoreLog"
}

Start-IfNeeded -Name "core" -PidFile $CorePidFile -LogFile $CoreLog -Command "npm run core"
Wait-ForCore
Start-IfNeeded -Name "desktop" -PidFile $DesktopPidFile -LogFile $DesktopLog -Command "npm run desktop"

Write-Host "FORGE start initiated."
Write-Host "Desktop startup runs in background and can take time on first compile."
Write-Host "Core log:    $CoreLog"
Write-Host "Desktop log: $DesktopLog"
