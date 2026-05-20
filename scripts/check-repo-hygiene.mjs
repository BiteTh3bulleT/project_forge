#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

export const repoRoot = dirname(dirname(fileURLToPath(import.meta.url)));

const trackedToolArtifactPrefixes = [
  "services/core/ForgeTestFile/",
  "services/core/directory/",
  "services/core/test_project/",
  "services/core/scratch/",
];

export function findTrackedToolArtifacts(rootDir = repoRoot) {
  const output = execFileSync("git", ["ls-files"], {
    cwd: rootDir,
    encoding: "utf8",
  });
  return output
    .split(/\r?\n/)
    .filter(Boolean)
    .filter((path) =>
      trackedToolArtifactPrefixes.some((prefix) => path.startsWith(prefix)),
    )
    .sort();
}

function printResult(artifacts) {
  if (artifacts.length === 0) {
    console.log("Repo hygiene OK: no tracked tool execution artifacts.");
    return;
  }
  console.error(`Repo hygiene FAILED: ${artifacts.length} tracked tool artifact(s).`);
  for (const artifact of artifacts) {
    console.error(`- ${artifact}`);
  }
}

if (process.argv[1] && relative(repoRoot, fileURLToPath(import.meta.url)) === relative(repoRoot, process.argv[1])) {
  const artifacts = findTrackedToolArtifacts(repoRoot);
  printResult(artifacts);
  process.exit(artifacts.length === 0 ? 0 : 1);
}
