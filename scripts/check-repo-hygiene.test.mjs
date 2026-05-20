import test from "node:test";
import assert from "node:assert/strict";

import { findTrackedToolArtifacts, repoRoot } from "./check-repo-hygiene.mjs";

test("repo has no tracked tool execution artifacts", () => {
  assert.deepEqual(findTrackedToolArtifacts(repoRoot), []);
});
