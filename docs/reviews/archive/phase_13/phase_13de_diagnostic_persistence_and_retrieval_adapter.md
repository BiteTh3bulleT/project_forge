# Phase 13D-E Diagnostic Persistence and Retrieval Adapter

Scope: `LIVE_INFRA / DIAGNOSTIC_STORAGE / DEFAULT_SQLITE / DISABLED_BY_DEFAULT`.

## Summary

Phase 13D-E adds the first safe diagnostic persistence primitives and retrieval metadata relational adapter scaffold. SQLite remains the live default. Postgres diagnostic persistence is opt-in, disabled by default, and non-authoritative.

## Persisted Data

When explicitly enabled, the Postgres diagnostic repository may persist:

- report id and report kind,
- workspace, request, and correlation refs,
- observed, stored, and retention expiry timestamps,
- route/chat/retrieval/advisory summary counts and classes,
- safe retrieval run/result/source refs and score summaries,
- warnings,
- no-effect verification,
- schema version,
- safe bounded metadata.

These rows are diagnostics only. They are not memory, evidence admission, canonical truth, or a read path.

## Explicitly Not Persisted

Diagnostic persistence rejects or omits:

- raw prompts and completions,
- message bodies,
- request and response bodies,
- source text, source chunks, document text, snippets, and retrieval content,
- raw queries,
- embeddings and vectors,
- tool payloads or outputs,
- memory content,
- auth headers, cookies, tokens, passwords, API keys, and secrets.

## Flags

- `FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED=false`
- `FORGE_SHADOW_DIAGNOSTIC_RETENTION_DAYS=30`
- `FORGE_SHADOW_DIAGNOSTIC_MAX_PAYLOAD_BYTES=65536`

Enabling diagnostic persistence requires explicit Postgres DSN configuration and does not change `FORGE_STORE_BACKEND`.

## Fallback Behavior

The existing in-memory diagnostic sink remains the primary diagnostic surface. The persistence sink wrapper stores in memory first and treats repository persistence as best effort. Repository failure does not fail diagnostic observation or live response handling.

Unsafe metadata and oversized payloads are rejected before persistence.

## Postgres Role

Postgres is used only as an optional diagnostic summary repository in this phase. It is not the live default store and does not receive canonical memory, retrieval content, jobs, approvals, settings, gateway records, or FORGE-K authority state.

## Retrieval Metadata Boundary

The retrieval metadata relational adapter maps existing safe retrieval metadata observations into relational-safe DTOs:

- retrieval run/result/source refs,
- counts and ranking position,
- score summary,
- retrieval strategy,
- index type,
- freshness status,
- duration,
- deterministic canonical JSON.

It does not execute retrieval, call search providers, call embedding providers, compile context, admit evidence, write memory, call Qdrant, or implement live RAG.

## Redis And Qdrant

Redis remains unwired to queues, caches, locks, or streams in this phase.

Qdrant remains unwired to retrieval and vector indexing in this phase.

## Validation

Default validation does not require Docker or Postgres. Optional Postgres integration tests run only when `FORGE_POSTGRES_TEST_DSN` is set.

Phase 13D-E validation includes config defaults, safe persistence row construction, unsafe metadata rejection, payload-size rejection, persistence failure isolation, Postgres migration/file parity, retrieval DTO safety, deterministic serialization, and no live retrieval execution.
