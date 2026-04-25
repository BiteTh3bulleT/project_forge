#!/usr/bin/env node
import { execFileSync, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";
import process from "node:process";

const port = String(process.argv[2] || "1420");
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

function run(command, args) {
  const result = spawnSync(command, args, { encoding: "utf8", shell: false });
  if (result.error) return "";
  return `${result.stdout || ""}${result.stderr || ""}`;
}

function windowsListeners() {
  const output = run("powershell.exe", [
    "-NoProfile",
    "-Command",
    `Get-NetTCPConnection -LocalPort ${port} -State Listen -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique`,
  ]);
  return output.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
}

function windowsCommandLine(pid) {
  return run("powershell.exe", [
    "-NoProfile",
    "-Command",
    `(Get-CimInstance Win32_Process -Filter "ProcessId=${pid}").CommandLine`,
  ]).trim();
}

function unixListeners() {
  const output = run("lsof", [`-tiTCP:${port}`, "-sTCP:LISTEN"]);
  return output.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
}

function unixCommandLine(pid) {
  return run("ps", ["-o", "command=", "-p", pid]).trim();
}

function isRepoVite(commandLine) {
  const normalized = commandLine.replaceAll("\\", "/").toLowerCase();
  const repo = repoRoot.replaceAll("\\", "/").toLowerCase();
  return normalized.includes(repo) && normalized.includes("vite");
}

const isWindows = process.platform === "win32";
const listeners = isWindows ? windowsListeners() : unixListeners();
if (listeners.length === 0) process.exit(0);

for (const pid of listeners) {
  const commandLine = isWindows ? windowsCommandLine(pid) : unixCommandLine(pid);
  if (isRepoVite(commandLine)) {
    console.log(`[forge desktop] Found stale local Vite process on :${port} (pid ${pid}), stopping it.`);
    if (isWindows) {
      execFileSync("taskkill.exe", ["/PID", pid, "/T", "/F"], { stdio: "ignore" });
    } else {
      process.kill(Number(pid), "SIGTERM");
    }
    continue;
  }

  console.error(`[forge desktop] Port ${port} is already in use by a non-FORGE process:`);
  console.error(`  pid: ${pid}`);
  console.error(`  cmd: ${commandLine || "(unknown)"}`);
  console.error("");
  console.error("Stop that process or change the desktop dev port in apps/desktop/vite.config.ts and src-tauri/tauri.conf.json.");
  process.exit(1);
}
