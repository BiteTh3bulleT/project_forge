#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

const requiredDocs = [
  "docs/architecture/forge_axiom_cognition.md",
  "docs/architecture/big_brain_search.md",
  "docs/architecture/factual_search_lanes.md",
  "docs/architecture/layered_rag_context_engine.md",
  "docs/architecture/context_compiler_search_index.md",
  "docs/adr/0016-forge-axiom-cognition-engine.md",
];

const globalRequiredPhrases = [
  "AXIOM is an internal FORGE cognition, search, and context layer",
  "does not execute tools",
  "does not write canonical memory",
  "does not bypass Gateway, approvals, audit, Control Lane, or modelruntime",
  "search results are evidence candidates",
  "FORGE-K remains simulator and shadow validation",
];

const perFileRequiredPhrases = new Map([
  [
    "docs/architecture/forge_axiom_cognition.md",
    [
      "single authority map",
      "SearchEvidencePacket",
      "ContextPacket",
      "TrustTier",
      "RoutingMode",
    ],
  ],
  [
    "docs/architecture/big_brain_search.md",
    [
      "trust-tiered",
      "freshness",
      "rejected candidates",
      "official documentation",
    ],
  ],
  [
    "docs/architecture/factual_search_lanes.md",
    [
      "local live workspace",
      "official source",
      "web search",
      "vector recall",
      "never authorization",
    ],
  ],
  [
    "docs/architecture/layered_rag_context_engine.md",
    [
      "expansion",
      "contraction",
      "stale memory",
      "low-trust",
      "citation",
    ],
  ],
  [
    "docs/architecture/context_compiler_search_index.md",
    [
      "COMPILE_CONTEXT",
      "non-canonical index",
      "source refs",
      "rejected candidate",
      "shape, not truth",
    ],
  ],
  [
    "docs/adr/0016-forge-axiom-cognition-engine.md",
    [
      "Status: Proposed",
      "Decision",
      "Consequences",
      "Out Of Scope",
      "no duplicate authority plane",
    ],
  ],
]);

function missingPhrases(text, phrases) {
  const lowerText = text.toLowerCase();
  return phrases.filter((phrase) => !lowerText.includes(phrase.toLowerCase()));
}

const failures = [];

for (const rel of requiredDocs) {
  const abs = join(repoRoot, rel);
  if (!existsSync(abs)) {
    failures.push(`${rel}: missing required document`);
    continue;
  }

  const text = readFileSync(abs, "utf8");
  for (const phrase of missingPhrases(text, globalRequiredPhrases)) {
    failures.push(`${rel}: missing phrase "${phrase}"`);
  }
  for (const phrase of missingPhrases(text, perFileRequiredPhrases.get(rel) ?? [])) {
    failures.push(`${rel}: missing phrase "${phrase}"`);
  }
}

if (failures.length > 0) {
  console.error("AXIOM cognition docs validation FAILED:");
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log("AXIOM cognition docs validation OK.");
