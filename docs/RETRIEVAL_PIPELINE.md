# FORGE Retrieval Pipeline

## Overview

Retrieval is a persisted pipeline, not a single search call.

Stages:
1. Scope resolution
2. Candidate generation
3. Hybrid scoring
4. Structural/usefulness reranking
5. VSA authority exclusion
6. Coverage-aware packet selection
7. Production FORGE-K evidence commit

## 1) Scope Resolution

Inputs:
- query
- mode (`keyword`, `semantic`, `hybrid`)
- dossier id (optional)
- explicit source ids (optional)

When dossier is provided and source ids are omitted, FORGE derives source scope from `dossier_sources`.

K20G resolves every selected source id to its canonical source-root path before search. At least one source is required. The sorted, unique source-root set becomes `ForgeScope.SelectedPaths`; any caller-supplied selected paths must match that set exactly. The `RECORD_RETRIEVAL_EVIDENCE` validator also requires every result's canonical absolute path to remain inside one of those sealed roots.

## 2) Candidate Generation

- Keyword candidates from FTS (`search.SearchScoped`)
- Semantic candidates from embeddings (`embeddings.SemanticSearch`)
- Aggregated by chunk id

## 3) Hybrid Scoring

Base scoring:
- keyword-only: keyword score
- semantic-only: semantic score
- hybrid: weighted keyword + semantic

Weights default from settings and can be overridden per run.

## 4) Reranking

FORGE applies reranking signals:
- historical usefulness by path
- dossier file bias (`high_value_files` boost, `noisy_files` penalty)
- small recency/consensus boost when both retrieval modes agree

## 5) VSA Authority Exclusion

K20G forces VSA influence off for governed retrieval evidence. Legacy observation ids and manifest-less VSA signals are neither applied nor persisted in the retrieval evidence bundle. Selection reasons record only `vsaInfluence=disabled_unscoped`. Re-enabling VSA influence requires a separately reviewed, immutable manifest with exact workspace/lane/selected-path scope and atomic admission through FORGE-K.

## 6) Coverage-Aware Selection

Packet selection avoids simple top-N duplicates.

Selection strategy:
- first pass prefers unique file-path coverage
- second pass fills remaining slots by score order

This reduces redundant packet context.

## 7) Production FORGE-K Evidence Commit

`retrieval.Service.Run` computes candidates but does not write retrieval tables directly. It submits one `RECORD_RETRIEVAL_EVIDENCE` semantic syscall through the boot-selected production `forgekernel.Kernel`. The Kernel binds authenticated actor/source, exact scope and provenance, registry/capability proof, deterministic prepared plan, request fingerprint, journal entry, audit-outbox intent, and idempotency evidence.

One SQLite transaction writes:

- `retrieval_runs`
- ordered `retrieval_results`
- one `retrieval_result_selection` reason per result
- `packet_retrieval_runs` only when a packet id is already known at commit time
- provenance, journal hash-chain entry/head, immutable audit intent, and idempotency proof

The transaction creates no `memory_observations`, `retrieval_result_observations`, or `retrieval_result_vsa_signals`. Models remain proposal-only and have no retrieval commit authority.

Run and result evidence rows are immutable. Their `original_*_id` fields preserve the source dossier/packet/job/chunk/file identities even after detachable live foreign keys become null during normal parent cleanup. Packet joins remain a live projection and may cascade away without erasing the immutable run evidence.

System/internal commits bind the constructed `forge.core` service principal. API-triggered commits retain the authenticated user origin; tokenless access is attested only for verified loopback/local-shell requests. Adapter and future-IRIS sources cannot commit this action.

Deterministic internal job retries first verify an existing immutable bundle against the exact job, actor, provenance, query, mode, and reconstructed source-root scope before search. User/proposal requests never use that shortcut and must traverse Kernel authorization.

## Usefulness Projection

Retrieval usefulness is a separate K20G append-only utility-evidence syscall. It does not update immutable result rows or VSA reliability counters. Reads overlay `retrieval_usefulness_projection`, which is explicitly noncanonical and rebuildable from immutable usefulness events.

## Operator Inspection

UI surfaces:
- `#/retrieval-runs`: run list, deterministic score breakdown, selection reason JSON, governed usefulness actions
- `#/memory`: legacy observation inspection and separately governed acceleration projections
