# PHASE 10 — Shadow Parity and Cutover Prep

## Objective

Run FORGE-HMK in shadow mode against current memory/context behavior and prepare narrow live-promotion candidates.

## Instructions

Add shadow harness. Record safe metadata only. Compare relevance, latency, cache hit ratio, contradiction warnings, provenance coverage, missing/extra memory. Write readiness report.

## Validation

Zero user-visible output changes. No canonical mutation. Safe metadata only. Rollback documented.

## What not to do

Do not cut over in this phase. Do not store raw sensitive payloads.

## Exit gate

Exit when safe shadow parity is demonstrated and one narrow reversible live candidate is documented.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
