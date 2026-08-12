import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const repoRoot = resolve(import.meta.dirname, "..");
const vaults = [".obsidian", "FORGE/.obsidian"];
const sharedJsonFiles = [
  "app.json",
  "appearance.json",
  "community-plugins.json",
  "core-plugins.json",
  "graph.json",
  "plugins/forge-engine-mcp/manifest.json"
];
const sharedBundleFiles = [
  "plugins/forge-engine-mcp/main.js",
  "plugins/forge-engine-mcp/styles.css"
];

function read(relativePath) {
  return readFileSync(resolve(repoRoot, relativePath), "utf8");
}

function readJson(relativePath) {
  return JSON.parse(read(relativePath));
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

for (const vault of vaults) {
  const app = readJson(`${vault}/app.json`);
  const core = readJson(`${vault}/core-plugins.json`);
  const community = readJson(`${vault}/community-plugins.json`);
  const manifest = readJson(
    `${vault}/plugins/forge-engine-mcp/manifest.json`
  );

  readJson(`${vault}/appearance.json`);
  readJson(`${vault}/graph.json`);
  assert(app.promptDelete === true, `${vault}: delete confirmation must be on`);
  for (const ignoredPath of [
    ".git/",
    ".worktrees/",
    ".claude/worktrees/",
    ".forge/",
    "node_modules/",
    "apps/desktop/dist/",
    "apps/desktop/src-tauri/target/",
    "crates/forgek-validate/target/"
  ]) {
    assert(
      app.userIgnoreFilters?.includes(ignoredPath),
      `${vault}: missing generated-path exclusion ${ignoredPath}`
    );
  }
  assert(core.sync === false, `${vault}: shared baseline must keep Sync off`);
  assert(
    community.includes("forge-engine-mcp"),
    `${vault}: forge-engine-mcp is not enabled`
  );
  assert(
    manifest.id === "forge-engine-mcp" && manifest.isDesktopOnly === true,
    `${vault}: invalid forge-engine-mcp manifest`
  );
}

for (const relativePath of sharedJsonFiles) {
  assert(
    JSON.stringify(readJson(`.obsidian/${relativePath}`)) ===
      JSON.stringify(readJson(`FORGE/.obsidian/${relativePath}`)),
    `vault configuration drift: ${relativePath}`
  );
}

for (const relativePath of sharedBundleFiles) {
  assert(
    read(`.obsidian/${relativePath}`) ===
      read(`FORGE/.obsidian/${relativePath}`),
    `vault configuration drift: ${relativePath}`
  );
}

for (const relativePath of [
  ".obsidian/workspace.json",
  "FORGE/.obsidian/workspace.json",
  ".obsidian/plugins/forge-engine-mcp/context.json",
  "FORGE/.obsidian/plugins/forge-engine-mcp/context.json"
]) {
  execFileSync("git", ["check-ignore", "--quiet", relativePath], {
    cwd: repoRoot
  });
}

for (const relativePath of [
  ".obsidian/plugins/forge-engine-mcp/main.js",
  "FORGE/.obsidian/plugins/forge-engine-mcp/main.js"
]) {
  execFileSync(process.execPath, ["--check", relativePath], {
    cwd: repoRoot,
    stdio: "pipe"
  });
}

console.log("Obsidian configuration OK (2 vaults, bounded context bridge).");
