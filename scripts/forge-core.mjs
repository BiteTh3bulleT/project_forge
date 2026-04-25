import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.resolve(__dirname, "..");

const check = spawnSync(
  process.execPath,
  [path.join(rootDir, "scripts", "check-vsa-files.mjs"), "--require-tracked"],
  { cwd: rootDir, stdio: "inherit" },
);

if (check.error) {
  console.error(check.error.message);
  process.exit(1);
}
if ((check.status ?? 1) !== 0) {
  process.exit(check.status ?? 1);
}

const env = {
  ...process.env,
  FORGE_ENABLE_MODEL_RUNTIME: process.env.FORGE_ENABLE_MODEL_RUNTIME ?? "true",
};

const result = spawnSync("go", ["run", "."], {
  cwd: path.join(rootDir, "services", "core"),
  env,
  stdio: "inherit",
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);
