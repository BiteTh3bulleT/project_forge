# K20E FORGE-K Restore Retirement Report

Date: 2026-08-14  
Status: live apply retired; deterministic inspection active

## Outcome

K20E removes raw bundle row merge from production authority. The backup restore
API is inspection-only. Non-dry API and direct-service calls return the stable
`FORGE_K_RESTORE_APPLY_DISABLED` failure before reading the bundle, creating an
approval, creating an approval job, or mutating SQLite. The live I/O lane now
depends on `InspectBundle`, not restore apply. The old raw upsert engine is
unexported and has no production callsite; it exists only to preserve focused
legacy compatibility and rollback tests while the replacement is designed.

Valid restore-outcome feedback POST bodies return
`FORGE_K_RESTORE_OUTCOME_FEEDBACK_DISABLED` without changing the outcome row or
canonical memory. JSON decoding, required-field validation, and the one-MiB body
limit still occur first, so malformed and oversized requests retain 400/413
behavior. List/get outcome evidence remains readable and workspace-scoped.

## Inspection proof

Dry inspection performs bounded governed-path reading and returns:

- SHA-256 of the exact raw bundle bytes;
- normalized, deduplicated, sorted effective sections;
- strict schema/kind/manifest checks, including duplicate JSON and manifest keys;
- computed and declared row counts and canonical row checksums;
- local-versus-declared section authority and policy validation;
- a deterministic plan digest binding the bundle digest, schema, kind, sections,
  dispositions, integrity values, and blockers.

Journal events and idempotency proof are marked `never_live_merge`. Journal head,
audit outbox, and Courthouse exhibits/rulings/appeals are
`offline_recovery_only`. Other current/history/provenance sections require a
Kernel semantic migration, evidence quarantine, rebuild, or owning-subsystem
reconciliation. Inspection never claims transaction atomicity or apply success.

## Export coverage and limits

`full_backup` now exports Court exhibits, rulings, appeals, the FORGE-K journal
head, immutable audit-outbox records, journal events, and typed idempotency proof
rows. This is preservation and inspection evidence only. It does not make a
foreign proof valid in the live store and does not provide recovery parity.

A future recovery path must stop the daemon, validate exact bundle and manifest
identity, restore the whole database rather than merge rows, run SQLite integrity
checks, verify the complete journal hash chain and head, validate immutable
Courthouse/audit/idempotency proof relationships and workspace identity, and
atomically swap the store with a tested rollback. Until those gates exist, apply
remains disabled.

## Validation

Focused coverage includes direct and API fail-close order, no approval/job
creation, no feedback mutation, preserved request body limits, deterministic
inspection, count/checksum tamper rejection, full-backup authority section
presence, live I/O interface shape, and a static production-callsite guard.

Commands run from `services/core` or the repository root:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/forgekernel/... ./internal/aios/controllane ./internal/store ./internal/api ./internal/memory ./internal/retrieval ./internal/backup`
- `go test -count=1 ./internal/backup ./internal/memory ./internal/retrieval ./internal/aios/iolane ./internal/authoritymatrix`
- `git diff --check`

## Remaining blockers

- Daemon-stopped staged whole-store recovery is not implemented.
- Bounded semantic import requires object-specific Kernel operations; foreign
  current/history rows must not be raw-upserted.
- External audit delivery and `audit_id` backfill remain projections rather
  than part of the already-atomic canonical outbox intent.
