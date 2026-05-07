#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";
import process from "node:process";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dryRun = process.argv.includes("--dry-run");

function run(command, args) {
  const result = spawnSync(command, args, { encoding: "utf8", shell: false });
  if (result.error) {
    return { code: 1, output: result.error.message };
  }
  return {
    code: result.status ?? 0,
    output: `${result.stdout || ""}${result.stderr || ""}`,
  };
}

function cleanWindows() {
  const escapedRoot = repoRoot.replaceAll("'", "''");
  const dryRunLiteral = dryRun ? "$true" : "$false";
  const script = `
$ErrorActionPreference = "Stop"
$RootDir = '${escapedRoot}'
$DryRun = ${dryRunLiteral}
$DesktopExe = (Join-Path $RootDir 'apps\\desktop\\src-tauri\\target\\debug\\forge_desktop.exe').ToLowerInvariant()
$All = Get-CimInstance Win32_Process
$ByParent = @{}
foreach ($Process in $All) {
  if ($null -eq $Process.ParentProcessId) { continue }
  if (-not $ByParent.ContainsKey($Process.ParentProcessId)) {
    $ByParent[$Process.ParentProcessId] = New-Object System.Collections.ArrayList
  }
  [void]$ByParent[$Process.ParentProcessId].Add($Process)
}

function Add-Tree {
  param([int]$ProcessId, [System.Collections.Generic.HashSet[int]]$Targets)
  if ($Targets.Contains($ProcessId)) { return }
  [void]$Targets.Add($ProcessId)
  if ($ByParent.ContainsKey($ProcessId)) {
    foreach ($Child in $ByParent[$ProcessId]) {
      Add-Tree -ProcessId ([int]$Child.ProcessId) -Targets $Targets
    }
  }
}

$Targets = [System.Collections.Generic.HashSet[int]]::new()
foreach ($Process in $All) {
  $CommandLine = [string]$Process.CommandLine
  $ExecutablePath = [string]$Process.ExecutablePath
  $Name = [string]$Process.Name
  $MatchesDesktopExe = $Name -eq 'forge_desktop.exe' -and $ExecutablePath.ToLowerInvariant() -eq $DesktopExe
  $MatchesTauriCli = $CommandLine.Contains($RootDir) -and $CommandLine.Contains('@tauri-apps') -and $CommandLine.Contains('tauri.js') -and $CommandLine.Contains(' dev')
  if ($MatchesDesktopExe -or $MatchesTauriCli) {
    Add-Tree -ProcessId ([int]$Process.ProcessId) -Targets $Targets
    if ($Process.ParentProcessId) {
      $Parent = $All | Where-Object { $_.ProcessId -eq $Process.ParentProcessId } | Select-Object -First 1
      if ($Parent -and ([string]$Parent.Name) -in @('cargo.exe', 'cmd.exe')) {
        Add-Tree -ProcessId ([int]$Parent.ProcessId) -Targets $Targets
      }
    }
  }
}

if ($Targets.Count -eq 0) {
  exit 0
}

$CurrentPid = $PID
$Targets = @($Targets | Where-Object { $_ -ne $CurrentPid } | Sort-Object -Descending)
foreach ($ProcessId in $Targets) {
  $Process = $All | Where-Object { $_.ProcessId -eq $ProcessId } | Select-Object -First 1
  if (-not $Process) { continue }
  Write-Host "[forge desktop] Stopping stale Tauri dev process $ProcessId ($($Process.Name))."
  if (-not $DryRun) {
    Stop-Process -Id $ProcessId -Force -ErrorAction SilentlyContinue
  }
}
`;
  const result = run("powershell.exe", ["-NoProfile", "-Command", script]);
  process.stdout.write(result.output);
  return result.code;
}

function cleanUnix() {
  return 0;
}

const code = process.platform === "win32" ? cleanWindows() : cleanUnix();
process.exit(code);
