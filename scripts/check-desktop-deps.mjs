#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import process from "node:process";

if (process.platform !== "linux") {
  process.exit(0);
}

function hasCommand(command) {
  return spawnSync("sh", ["-lc", `command -v ${command} >/dev/null 2>&1`]).status === 0;
}

if (!hasCommand("pkg-config")) {
  console.error("[forge desktop] Missing required tool: pkg-config");
  console.error("Install pkg-config, then rerun `npm run desktop`.");
  process.exit(1);
}

const missing = [];
for (const dep of ["webkit2gtk-4.1", "javascriptcoregtk-4.1", "gtk+-3.0"]) {
  const result = spawnSync("pkg-config", ["--exists", dep]);
  if (result.status !== 0) missing.push(dep);
}

if (missing.length === 0) process.exit(0);

console.error("[forge desktop] Missing required Linux desktop libraries:");
for (const dep of missing) {
  console.error(`  - ${dep}`);
}
console.error("");
console.error("Install packages that provide:");
console.error("  pkgconfig(webkit2gtk-4.1)");
console.error("  pkgconfig(javascriptcoregtk-4.1)");
console.error("  pkgconfig(gtk+-3.0)");
process.exit(1);
