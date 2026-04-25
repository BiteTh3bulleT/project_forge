#!/usr/bin/env node
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
let requireTracked = false;

function usage() {
  console.log(`usage: check-vsa-files.mjs [--require-tracked]

Verifies required VSA source files exist for FORGE core build/boot.
With --require-tracked, also requires those files to be git-tracked in this checkout.`);
}

for (const arg of process.argv.slice(2)) {
  if (arg === "--require-tracked") {
    requireTracked = true;
  } else if (arg === "-h" || arg === "--help") {
    usage();
    process.exit(0);
  } else {
    console.error(`unknown argument: ${arg}`);
    usage();
    process.exit(2);
  }
}

const required = [
  "services/core/internal/memory/vsa_engine.go",
  "services/core/internal/memory/vsa_indexer.go",
  "services/core/internal/memory/vsa_signals.go",
];

const missing = required.filter((rel) => !existsSync(join(repoRoot, rel)));
if (missing.length > 0) {
  console.error("FORGE bring-up preflight failed: VSA source files are missing.\n");
  console.error("Missing files:");
  for (const rel of missing) console.error(`  - ${rel}`);
  console.error("\nAction:");
  console.error("  1) Ensure these files are present in your branch/checkout.");
  console.error("  2) Re-run the command after syncing the missing files.");
  console.error("\nWhy this check exists:");
  console.error("  The core memory service depends on these VSA implementations at compile time.");
  process.exit(42);
}

if (requireTracked) {
  const inside = spawnSync("git", ["-C", repoRoot, "rev-parse", "--is-inside-work-tree"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (inside.error && inside.error.code === "ENOENT") {
    console.error("FORGE bring-up preflight failed: --require-tracked was requested, but git is not installed.");
    process.exit(44);
  }
  if (inside.status !== 0) {
    console.error("FORGE bring-up preflight failed: --require-tracked was requested outside a git work tree.");
    process.exit(45);
  }

  const untracked = [];
  for (const rel of required) {
    const res = spawnSync("git", ["-C", repoRoot, "ls-files", "--error-unmatch", rel], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    });
    if (res.status !== 0) untracked.push(rel);
  }
  if (untracked.length > 0) {
    console.error("FORGE bring-up preflight failed: VSA source files are present but not git-tracked in this checkout.\n");
    console.error("Files not git-tracked:");
    for (const rel of untracked) console.error(`  - ${rel}`);
    console.error("\nVerification command:");
    console.error("  git ls-files services/core/internal/memory/vsa_*.go");
    process.exit(43);
  }
}
