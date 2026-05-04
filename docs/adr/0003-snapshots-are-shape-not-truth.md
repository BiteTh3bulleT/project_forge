# ADR 0003 - Snapshots Are Shape, Not Truth

Status: Accepted

Date: 2026-05-03

## Context

FORGE context restore and future FORGE-K snapshot work need compact restorable structures. These structures can preserve context shape, route shape, decision shape, or runtime shape, but they must not become canonical truth by existing.

## Decision

Snapshots preserve restorable semantic shape but do not become canonical truth.

Snapshots must cite source objects, hashes, operation records, and summaries. They may seed restoration, replay, context compilation, review, or cache eligibility. Canonical state still comes from Kernel commits and journaled semantic transitions.

## Consequences

- Snapshot restoration must verify source references and hashes.
- Snapshot content must not override current canonical state.
- Snapshot compaction must preserve provenance and operation records.
- Superseded snapshots remain inspectable when retained by policy.

## Alternatives considered

- Snapshot as memory: rejected because it would collapse shape into truth.
- Raw context replay as truth: rejected because replayed prompt text lacks deterministic admission and commit semantics.
- Full duplication of canonical content in snapshots: rejected because it increases drift risk and weakens source-of-truth boundaries.
