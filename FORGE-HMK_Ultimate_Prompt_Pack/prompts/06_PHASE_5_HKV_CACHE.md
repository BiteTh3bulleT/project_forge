# PHASE 5 — HKV Hierarchical Cache

## Objective

Implement HKV manifest identity, dependency tracking, TTL policy, dirty-state handling, and cognitive cache metadata.

## Instructions

Implement cache layers, HKVManifest store, dependency invalidation, dirty states, workspace isolation, and tests for identity mismatch/expiry/dirty hit blocking.

## Validation

Cache hit requires exact identity/dependency match. Dirty/expired entries blocked. HKV disable safe.

## What not to do

Do not implement engine-specific GPU KV yet. Do not cache final answers as truth.

## Exit gate

Exit when HKV safely validates, invalidates, and blocks stale cache entries.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
