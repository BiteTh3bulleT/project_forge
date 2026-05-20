import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import {
  loadOSIntegrationSources,
  validateOSIntegrationSources,
} from "./validate-os-integration.mjs";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

test("current OS integration sources satisfy the readiness gate", () => {
  const result = validateOSIntegrationSources(loadOSIntegrationSources(repoRoot));

  assert.deepEqual(result.failures, []);
  assert.ok(result.checks.length >= 20, "expected a substantial invariant set");
});

test("readiness gate fails closed on unsafe native desktop autologin drift", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const nativeProfile = "nix/nixos/profiles/forge-native-desktop-runtime.nix";
  sources[nativeProfile] = sources[nativeProfile]
    .replace("autoLogin.enable = lib.mkForce false;", "autoLogin.enable = true;")
    .replace(
      "services.getty.autologinUser = lib.mkForce null;",
      'services.getty.autologinUser = "operator";',
    );

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("native.display-manager-autologin-disabled"),
    `expected display-manager autologin failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("native.tty-autologin-disabled"),
    `expected TTY autologin failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches operator shell package drift", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const flake = "flake.nix";
  sources[flake] = sources[flake].replaceAll(
    "forge-operator-desktop-shell",
    "forge-desktop-shell",
  );

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("flake.operator-desktop-shell-package"),
    `expected operator desktop shell package failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches missing npm integration wiring", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const packageJson = "package.json";
  sources[packageJson] = readFileSync(join(repoRoot, packageJson), "utf8");
  const pkg = JSON.parse(sources[packageJson]);
  delete pkg.scripts["test:os-integration"];
  delete pkg.scripts["validate:os-integration"];
  pkg.scripts["validate:local"] = pkg.scripts["validate:local"].replace(
    "npm run test:os-integration && ",
    "",
  ).replace(
    " && npm run validate:os-integration",
    "",
  );
  sources[packageJson] = JSON.stringify(pkg, null, 2);

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("package.test-os-integration-script"),
    `expected test script failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("package.validate-os-integration-script"),
    `expected validate script failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("package.validate-local-os-integration-tests"),
    `expected validate:local test wiring failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("package.validate-local-os-integration"),
    `expected validate:local wiring failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches missing CI integration wiring", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const workflow = ".github/workflows/ci.yml";
  sources[workflow] = readFileSync(join(repoRoot, workflow), "utf8").replace(
    /.*npm run (test|validate):os-integration.*\n?/g,
    "",
  );

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("ci.os-integration-tests"),
    `expected CI test wiring failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("ci.os-integration-validation"),
    `expected CI validation wiring failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches broken canonical VM local model loop wiring", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const vmConfig = "nix/nixos/configurations/forge-operator-vm.nix";
  sources[vmConfig] = sources[vmConfig]
    .replace("enableModelRuntime = true;", "enableModelRuntime = false;")
    .replace("services.ollama = {\n    enable = lib.mkDefault true;", "services.ollama = {\n    enable = lib.mkDefault false;")
    .replace('FORGE_MODEL_DEFAULT_BACKEND = lib.mkDefault "ollama_compat";', 'FORGE_MODEL_DEFAULT_BACKEND = lib.mkDefault "none";')
    .replace("FORGE_MODEL_DEFAULT_BACKEND=ollama_compat", "FORGE_MODEL_DEFAULT_BACKEND=none");

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("vm.modelruntime-enabled"),
    `expected modelruntime enablement failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("vm.ollama-enabled"),
    `expected Ollama enablement failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("vm.default-model-backend"),
    `expected model backend failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("vm.operator-env-default-backend"),
    `expected operator env backend failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches VM storage path drift away from /forge", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const vmConfig = "nix/nixos/configurations/forge-operator-vm.nix";
  sources[vmConfig] = sources[vmConfig]
    .replace('storageRoot = lib.mkDefault "/forge";', 'storageRoot = lib.mkDefault "/tmp/forge";')
    .replace('home = lib.mkDefault "/forge/models/ollama";', 'home = lib.mkDefault "/tmp/ollama";');

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("vm.storage-root"),
    `expected storage root failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("vm.ollama-model-home"),
    `expected Ollama home failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches canonical VM safe-mode drift", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const vmConfig = "nix/nixos/configurations/forge-operator-vm.nix";
  sources[vmConfig] = sources[vmConfig]
    .replace("safeMode = lib.mkDefault true;", "safeMode = lib.mkDefault false;")
    .replace("safeModeForceCPUOnly = lib.mkDefault true;", "safeModeForceCPUOnly = lib.mkDefault false;");

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("vm.forge-os-safe-mode"),
    `expected forge.os safe-mode failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("vm.core-safe-mode-force-cpu"),
    `expected core CPU safe-mode failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate catches missing agent guidance wiring", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  sources["AGENTS.md"] = readFileSync(join(repoRoot, "AGENTS.md"), "utf8");
  sources["AGENTS.md"] = sources["AGENTS.md"].replace(
    /.*npm run validate:os-integration.*\n?/g,
    "",
  );

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("agents.os-integration-command"),
    `expected AGENTS command wiring failure, got ${failureIds.join(", ")}`,
  );
});

test("CI runs OS integration readiness before desktop validation", () => {
  const workflow = readFileSync(join(repoRoot, ".github/workflows/ci.yml"), "utf8");

  assert.match(workflow, /npm run test:os-integration/);
  assert.match(workflow, /npm run validate:os-integration/);
  assert.ok(
    workflow.indexOf("npm run test:os-integration") < workflow.indexOf("npm run validate:os-integration"),
    "OS integration tests should run before OS integration validation",
  );
  assert.ok(
    workflow.indexOf("npm run validate:os-integration") < workflow.indexOf("npm run validate:js"),
    "OS integration validation should run before JS/TS validation",
  );
});

test("operator VM runbook keeps model controls governed instead of banned outright", () => {
  const runbook = readFileSync(
    join(repoRoot, "docs/runbooks/forge_operator_desktop_vm.md"),
    "utf8",
  );

  assert.doesNotMatch(
    runbook,
    /Do not add model load\/unload, service restart, or rebuild controls to the UI\./,
  );
  assert.match(runbook, /governed modelruntime approval-gated model load\/unload controls/);
  assert.match(runbook, /Do not add service restart or rebuild controls to the UI\./);
});

test("readiness gate catches stale operator VM model-control boundaries", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  const runbook = "docs/runbooks/forge_operator_desktop_vm.md";
  sources[runbook] =
    "- Do not add model load/unload, service restart, or rebuild controls to the UI.\n";

  const result = validateOSIntegrationSources(sources);
  const failureIds = result.failures.map((failure) => failure.id);

  assert.ok(
    failureIds.includes("operator-runbook.governed-model-controls"),
    `expected governed model control wording failure, got ${failureIds.join(", ")}`,
  );
  assert.ok(
    failureIds.includes("operator-runbook.no-stale-model-control-ban"),
    `expected stale boundary failure, got ${failureIds.join(", ")}`,
  );
});

test("readiness gate reports malformed package JSON as structured failures", () => {
  const sources = loadOSIntegrationSources(repoRoot);
  sources["package.json"] = "{";

  let result;
  assert.doesNotThrow(() => {
    result = validateOSIntegrationSources(sources);
  });
  const packageFailures = result.failures.filter(
    (failure) => failure.path === "package.json",
  );

  assert.equal(result.ok, false);
  assert.ok(packageFailures.length >= 1, "expected package.json failure");
  assert.match(packageFailures[0].message, /check threw:/);
});
