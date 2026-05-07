import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const isWindows = process.platform === "win32";
const scriptPath = path.join(__dirname, isWindows ? "forge-smoke.ps1" : "forge-smoke.sh");
const command = isWindows ? "powershell" : "bash";
const args = isWindows
  ? ["-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, ...process.argv.slice(2)]
  : [scriptPath, ...process.argv.slice(2)];

const result = spawnSync(command, args, { stdio: "inherit" });
if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
