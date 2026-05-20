import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import { findTrackedToolArtifacts, repoRoot } from "./check-repo-hygiene.mjs";

test("repo has no tracked tool execution artifacts", () => {
  assert.deepEqual(findTrackedToolArtifacts(repoRoot), []);
});

test("repo root has no tracked review-looking prompt templates", () => {
  const output = execFileSync("git", ["ls-files", "Full-Code-Review.md"], {
    cwd: repoRoot,
    encoding: "utf8",
  });
  assert.equal(output.trim(), "");
});

test("docker compose does not ship a hardcoded Postgres password default", () => {
  const compose = readFileSync(join(repoRoot, "docker-compose.yml"), "utf8");
  assert.equal(compose.includes("forge_dev_password"), false);
});
