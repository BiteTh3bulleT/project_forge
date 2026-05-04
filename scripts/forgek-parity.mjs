#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const root = resolve(new URL("..", import.meta.url).pathname);
const fixtureRoot = resolve(root, "fixtures/forgek");
const goldenPath = resolve(root, "fixtures/forgek/golden/hashes.json");

const unorderedArrayKeys = new Set([
  "source_refs",
  "blocks_refs",
  "supported_models",
  "supported_capabilities",
  "allowed_syscalls",
  "workspace_scope",
  "provenance_refs",
  "context_block_refs",
]);

const stableExcludedFields = new Set([
  "created_at",
  "updated_at",
  "sealed_at",
  "expired_at",
  "last_used_at",
  "invalidated_at",
  "journal_refs",
  "shape_hash",
  "source_hash",
  "content_hash",
  "token_input_hash",
  "bundle_hash",
  "stable_prefix_hash",
  "volatile_suffix_hash",
  "cache_id",
  "snapshot_id",
  "block_id",
  "bundle_id",
  "driver_id",
  "reuse_count",
]);

const expectedKinds = new Map([
  ["context_block", "context_block_hash"],
  ["context_bundle", "context_bundle_hash"],
  ["kv_cache_manifest", "kv_manifest_identity_hash"],
  ["runtime_driver_manifest", "runtime_manifest_hash"],
  ["snapshot_manifest", "snapshot_shape_hash"],
]);

function runStep(label, command, args, cwd) {
  console.log(`[forgek-parity] ${label}`);
  const result = spawnSync(command, args, {
    cwd,
    stdio: "inherit",
  });
  if (result.error) {
    console.error(`[forgek-parity] ${label} failed to start: ${result.error.message}`);
    process.exit(1);
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function readJson(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function normalizeWhitespace(value) {
  return value.split(/\s+/u).filter(Boolean).join(" ");
}

function shouldSortArray(parentKey) {
  return Boolean(parentKey && (parentKey.endsWith("_refs") || unorderedArrayKeys.has(parentKey)));
}

function stableSortKey(value) {
  return JSON.stringify(value);
}

function canonicalize(value, parentKey = undefined) {
  if (Array.isArray(value)) {
    const out = value.map((item) => canonicalize(item, parentKey));
    if (shouldSortArray(parentKey)) {
      out.sort((left, right) => stableSortKey(left).localeCompare(stableSortKey(right)));
    }
    return out;
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((key) => [key, canonicalize(value[key], key)]),
    );
  }
  if (typeof value === "string") {
    return normalizeWhitespace(value);
  }
  return value;
}

function removeExcluded(value) {
  if (Array.isArray(value)) {
    return value.map(removeExcluded);
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value)
        .filter(([key]) => !stableExcludedFields.has(key))
        .map(([key, child]) => [key, removeExcluded(child)]),
    );
  }
  return value;
}

function stableProjectionHash(value) {
  const projected = removeExcluded(canonicalize(value));
  const canonicalJson = JSON.stringify(projected, null, 2);
  return createHash("sha256").update(canonicalJson, "utf8").digest("hex");
}

function detectFixtureKind(value) {
  if (Object.hasOwn(value, "snapshot_id")) return "snapshot_manifest";
  if (Object.hasOwn(value, "bundle_id") && Object.hasOwn(value, "blocks")) return "context_bundle";
  if (Object.hasOwn(value, "block_id")) return "context_block";
  if (Object.hasOwn(value, "cache_id")) return "kv_cache_manifest";
  if (Object.hasOwn(value, "driver_id")) return "runtime_driver_manifest";
  return "unknown";
}

function assert(condition, message, failures) {
  if (!condition) failures.push(message);
}

runStep("Go FORGE-K simulator tests", "go", ["test", "./internal/forgek/..."], resolve(root, "services/core"));
runStep("Rust validator tests", "cargo", ["test"], resolve(root, "crates/forgek-validate"));
runStep(
  "Rust fixture validation",
  "cargo",
  ["run", "--", "validate-fixtures", "../../fixtures/forgek"],
  resolve(root, "crates/forgek-validate"),
);

console.log("[forgek-parity] Node golden hash manifest check");

const golden = readJson(goldenPath);
const entries = Array.isArray(golden.fixtures) ? golden.fixtures : [];
const failures = [];

assert(entries.length === 5, `expected 5 golden fixture entries, found ${entries.length}`, failures);

for (const entry of entries) {
  const fixturePath = entry.fixture_path;
  const fixtureKind = entry.fixture_kind;
  const hashKind = entry.hash_kind;
  const expectedHash = entry.expected_hash;

  assert(typeof fixturePath === "string" && fixturePath.length > 0, "fixture entry missing fixture_path", failures);
  assert(typeof fixtureKind === "string" && expectedKinds.has(fixtureKind), `${fixturePath}: unknown fixture_kind ${fixtureKind}`, failures);
  assert(hashKind === expectedKinds.get(fixtureKind), `${fixturePath}: hash_kind ${hashKind} does not match ${fixtureKind}`, failures);
  assert(/^[0-9a-f]{64}$/.test(expectedHash), `${fixturePath}: expected_hash is not sha256 hex`, failures);
  assert(Array.isArray(entry.included_fields_summary) && entry.included_fields_summary.length > 0, `${fixturePath}: missing included_fields_summary`, failures);
  assert(Array.isArray(entry.excluded_fields_summary) && entry.excluded_fields_summary.length > 0, `${fixturePath}: missing excluded_fields_summary`, failures);

  if (!fixturePath || !fixtureKind || !hashKind || !expectedHash) continue;

  const value = readJson(resolve(fixtureRoot, fixturePath));
  const detectedKind = detectFixtureKind(value);
  const actualHash = stableProjectionHash(value);

  assert(detectedKind === fixtureKind, `${fixturePath}: detected ${detectedKind}, expected ${fixtureKind}`, failures);
  assert(actualHash === expectedHash, `${fixturePath}: ${hashKind} expected ${expectedHash}, got ${actualHash}`, failures);

  if (golden[hashKind] !== undefined) {
    assert(golden[hashKind] === expectedHash, `${fixturePath}: legacy ${hashKind} does not match entry expected_hash`, failures);
  }
}

if (failures.length > 0) {
  console.error("FORGE-K parity fixture check failed:");
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log(`FORGE-K parity fixture check passed (${entries.length} golden entries).`);
