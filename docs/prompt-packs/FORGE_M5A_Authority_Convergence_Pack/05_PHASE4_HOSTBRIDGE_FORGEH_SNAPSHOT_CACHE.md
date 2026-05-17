# Phase 4 Prompt — HostBridge / FORGE-H Snapshot Cache

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Reduce status/diagnostic latency by moving expensive HostBridge/FORGE-H sampling out of routine request paths.

## Required design

Add a cache/sampler pattern:

```text
background sampler -> latest bounded snapshot -> read-only status consumers
```

## Requirements

Snapshot cache is read-only, does not mutate host state, does not write semantic memory, does not make FORGE-H authority, exposes age/staleness, preserves source errors, degrades honestly, and has bounded force-refresh only if needed.

## Implementation options

Option A: TTL cache wrapper.
Option B: background sampler.

Recommended for M5A: Option A first unless lifecycle is already clean.

## Tests

Cache hit, cache miss, stale TTL refresh, error preservation, no panic on nil snapshot, no host mutation flags, advisory-only posture remains true.

## WHAT NOT TO DO

Do not add host mutation, raw command controls, `systemctl` from UI, raw logs, hidden stale data, or authoritative FORGE-H decisions.
