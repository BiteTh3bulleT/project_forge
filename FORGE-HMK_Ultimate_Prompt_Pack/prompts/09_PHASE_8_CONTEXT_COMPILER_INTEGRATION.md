# PHASE 8 — Context Compiler Integration

## Objective

Integrate FORGE-HMK outputs with context compiler through shadow-only compiled context bundles.

## Instructions

Inspect existing compiler. Add adapter for non-canonical artifacts. Build bundles with included/excluded cells, provenance, relevance scores, contradiction warnings, stable/volatile blocks, HKV manifest. Add shadow comparison.

## Validation

Existing compiler default unchanged. FORGE-HMK output shadow-only. Token budget respected.

## What not to do

Do not replace compiler outright. Do not hide contradiction warnings.

## Exit gate

Exit when shadow context packets build with provenance and no live output changes.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
