# FORGE Memory Taxonomy

Status date: 2026-04-27.

This document formalizes FORGE memory around three temporal horizons, six processing functions, and nine memory types. It is a classification and authority guide, not a new persistence layer.

Core rule:

> Memory type does not grant truth authority.

FORGE is the OS. Kernel/control lane owns canonical truth. Durable canonical mutation still goes through semantic syscalls, validation, transaction boundaries, and audit. Restore snapshots, Dream reports, retrieval records, vector/VSA records, and restore outcome events are evidence or routing signals unless a future governed syscall promotes a specific claim into canonical memory.

## Temporal Horizons

| Horizon | Definition | Typical lifetime | Authority posture | Current examples |
|---|---|---:|---|---|
| Short-term | Active, recent, volatile, high-detail working context. | current turn to session | volatile or non-canonical evidence | chat thread context, active context packets, retrieval runs, recent observations |
| Mid-term | Consolidating, reviewable, project/session-scoped evidence and utility signals. | hours to weeks | mostly non-canonical evidence; some governed current projections | context snapshots, restore outcomes, open loops, usefulness events, Dream proposals |
| Long-term | Durable, governed, high-confidence memory/state. | indefinite until superseded/archived | canonical only when committed through semantic syscalls | active memory notes, state items, semantic links, procedural/structural derived records |

Horizon is about operating lifetime, not truth. A short-term event can be true evidence, and a long-term derived model can still be non-authoritative if it is only a model of evidence.

## Processing Functions

| Function | Definition | Authority rule |
|---|---|---|
| Capture | Record observations, events, snapshots, outcomes, artifacts, or proposed facts. | Capture may write evidence through governed paths; canonical capture requires semantic syscall commit. |
| Recall | Retrieve relevant memory for an operator, context packet, route, or report. | Recall never makes recalled content canonical truth by itself. |
| Route | Choose a lane, tier, review path, or next processing surface. | Routing may get stricter, but cannot bypass kernel/gateway/modelruntime policy. |
| Score | Assign deterministic relevance, salience, utility, confidence, or risk weights. | Scores are advisory evidence unless committed through an authority path. |
| Consolidate | Merge, promote, summarize, supersede, or propose durable retention. | Dream Mode may propose; kernel/control lane validates and commits. |
| Forget | Decay, demote, archive, supersede, or exclude from future recall. | Canonical forgetting is archive/supersede through syscall; evidence rows are preserved unless a governed retention policy exists. |

## Memory Types

| Type | Definition | Default horizon | Authority level | Storage locations today | TTL/decay expectation |
|---|---|---|---|---|---|
| Working | Active execution and context assembly state used for immediate work. | Short-term | volatile or non-canonical | chat messages, context packets, `retrieval_runs`, `retrieval_results`, `context_packet_snapshots` metadata | Decays quickly; persisted snapshots remain evidence but should not dominate fresh compile forever. |
| Episodic | Event history and observed episodes of work. | Short to mid-term | canonical for committed syscall journal; otherwise evidence | `journal_events`, `events`, `job_events`, `memory_observations`, `context_evidence` | Recent events are highly recallable; older events move to audit/history unless linked to active work. |
| Salience | Importance, attention, novelty, contradiction, blocked-loop, or review priority signals. | Short to mid-term | non-canonical evidence | Dream salience report fields, `contradiction_records`, `open_loops`, Rule Cell traces, attention signals | Decays with recency unless reinforced by correction, contradiction, failure, or active loop state. |
| Prospective | Future-oriented commitments, intentions, blockers, next actions, and planned work. | Mid-term | canonical when represented as governed loops/state; otherwise proposal | `open_loops`, task packets, autonomy intents/budgets where present, `state_items` keys for planned state | Expires by resolution, archive, supersession, or operator review. |
| Reflective | Self-review, diagnostics, repair findings, Dream proposals, and synthesis over prior evidence. | Mid-term | non-canonical until a syscall commits a resulting claim | Dream reports, `memory_repair_runs`, `memory_repair_items`, truth rebuild reports, review items | Should route to operator review; stale reflections should be superseded by newer reports. |
| Utility | Feedback about whether memory helped or hurt recall, restore, routing, or execution. | Mid-term | non-canonical evidence | `restore_outcome_events`, `memory_usefulness_events`, `retrieval_results.usefulness_*`, `retrieval_result_vsa_signals` | Bounded scoring influence; decays or is outweighed by newer utility evidence. |
| Semantic | Facts, preferences, decisions, goals, concepts, and current values. | Long-term | canonical only through control-lane syscalls | `memory_notes`, `state_items`, `state_versions`, `semantic_links`, `journal_events` | Retained until archived, superseded, contradicted, or revised through syscall. |
| Procedural | How-to knowledge, workflows, policies, and repeatable operating patterns. | Long-term | canonical when committed as notes/state/links; non-canonical when derived | `memory_notes` of procedural type, `derived_models`, packet guidance, docs references | Periodic review; deprecated when failures, policy drift, or supersession evidence accumulates. |
| Structural | Relationships, scope, dependency graph, topology, fingerprints, roles, and embeddings/VSA structure. | Mid to long-term | mixed: canonical links/state where syscall-backed; retrieval indexes are non-canonical | `semantic_links`, `artifact_refs`, `context_packet_snapshots`, `embedding_records`, `memory_vsa_*`, `retrieval_result_vsa_signals` | Recomputed when source fingerprints change; indexes are never truth authority. |

## Authority Matrix

| Memory type | Horizon | Canonical / non-canonical / volatile | Source | Promotion path | Demotion path | Used by restore scoring | Used by Dream Mode | Operator review required? |
|---|---|---|---|---|---|---|---|---|
| Working | Short-term | volatile or non-canonical | chat/context/retrieval runtime | select evidence into `COMPILE_CONTEXT`, then syscall if a durable semantic claim is needed | natural expiry, fresh compile, snapshot staleness | IMPLEMENTED via context snapshot candidates | PARTIAL via snapshots/recent candidates | Usually no, except high-risk context choices |
| Episodic | Short to mid-term | mixed | journal/events/jobs/observations | syscall-created journal and notes; evidence can support later note/link/state commits | archive history from active recall; never erase append-only journal | PARTIAL via included events and snapshot lineage | IMPLEMENTED via `journal_events` and observations where loaded | Review when becoming semantic truth |
| Salience | Short to mid-term | non-canonical evidence | Dream scoring, contradictions, blockers, Rule Cells | operator-reviewed proposal to note/link/state/loop | score decay, resolved loop, contradiction repair | PARTIAL via contradiction/staleness penalties | IMPLEMENTED in Dream salience | Yes for long-term promotion |
| Prospective | Mid-term | canonical when loop/state syscall-backed; otherwise proposal | loops/tasks/autonomy | `OPEN_LOOP`, `UPDATE_STATE`, note/link syscall | `CLOSE_LOOP`, archive, supersede | PARTIAL via open-loop overlap | IMPLEMENTED via active blocked-loop candidates | Yes for high-risk or autonomous intent |
| Reflective | Mid-term | non-canonical evidence | Dream/repair/truth reports | explicit semantic syscall from reviewed proposal | supersede stale report or no-op | SCAFFOLD via repair/review signals | IMPLEMENTED as dry-run proposals | Yes |
| Utility | Mid-term | non-canonical evidence | restore outcomes, usefulness labels, retrieval/VSA feedback | only as evidence for future governed memory changes | decay, cap, counter-signal, repair | IMPLEMENTED for restore outcomes; PARTIAL for retrieval usefulness | IMPLEMENTED for restore outcomes; PARTIAL for retrieval usefulness | Review when changing canonical memory |
| Semantic | Long-term | canonical when syscall-backed | notes/state/links/journal | semantic syscalls and kernel validation | archive, supersede, contradiction registration | IMPLEMENTED as candidate graph evidence | IMPLEMENTED as replay evidence | Depends on risk; yes for contradiction/promotion |
| Procedural | Long-term | mixed | docs, notes, derived models, packet guidance | syscall note/model/link after review | deprecate model, archive note, supersede doc | PARTIAL through notes/models/artifacts | PARTIAL as notes/models and repair proposals | Usually yes |
| Structural | Mid to long-term | mixed; indexes non-canonical | links, artifacts, snapshots, embeddings, VSA | semantic link/artifact syscall for canonical structure | reindex, mark stale, supersede link | IMPLEMENTED for links/artifacts/snapshots; vector/VSA is not truth | PARTIAL through links/artifacts/snapshots | Review for canonical graph changes |

## Processing Function Matrix

| Function | Neural lane | Arterial lane | Lymphatic lane | Kernel/control lane | Operator surface |
|---|---|---|---|---|---|
| Capture | Classify input/event as correction, failure, task, memory candidate. PARTIAL. | Capture context packet/snapshot inputs. IMPLEMENTED. | Capture replay candidates and repair facts. PARTIAL. | Commit canonical notes/state/links/loops/journal and non-canonical outcome evidence through governed paths. IMPLEMENTED. | Manual notes, feedback, approvals, diagnostics. PARTIAL. |
| Recall | Tag obvious memory candidates for later recall. SCAFFOLD. | Deterministic restore candidate listing and scoring. IMPLEMENTED. | Replay selection for Dream Mode. IMPLEMENTED. | Scope-aware repository reads. IMPLEMENTED. | Inspector/search/history views. PARTIAL. |
| Route | Fast no-model/tagging hints. SCAFFOLD. | Fresh compile vs restore, context budget hints. PARTIAL. | Tier routing proposals. IMPLEMENTED. | Approval/capability/scope denials win. IMPLEMENTED. | Attention/review badges. PARTIAL. |
| Score | Classification confidence. SCAFFOLD. | Restore score and bounded outcome/rule adjustments. IMPLEMENTED. | Salience and bounded rule adjustments. IMPLEMENTED. | Validation risk and policy state. IMPLEMENTED. | Usefulness/operator feedback scores. PARTIAL. |
| Consolidate | Candidate extraction only. SCAFFOLD. | Snapshot hygiene/fresh compile evidence. PARTIAL. | Dream proposal generation. IMPLEMENTED. | Syscall commit, supersession, contradiction registration. IMPLEMENTED. | Review/approve/apply path. PLANNED for Dream apply. |
| Forget | Suppress low-signal candidates. PLANNED. | Penalize stale snapshots and require fresh compile. IMPLEMENTED. | Demote/discard/no-op proposals. IMPLEMENTED. | Archive/supersede through syscall, preserve history. IMPLEMENTED. | Horizon-based archive/review views. PLANNED. |

## Current Schema To Memory Type Mapping

| Current object/table | Primary memory type(s) | Horizon | Status | Notes |
|---|---|---|---|---|
| `memory_notes` | Semantic, Procedural | Long-term | IMPLEMENTED | Canonical when committed by syscall; status changes preserve history. |
| `journal_events` | Episodic, Semantic | Long-term history | IMPLEMENTED | Append-only committed semantic trail. |
| `state_items` | Semantic, Prospective | Long-term current | IMPLEMENTED | Current projection; history lives in `state_versions`. |
| `state_versions` | Episodic, Semantic | Long-term history | IMPLEMENTED | Historical values are preserved. |
| `open_loops` | Prospective, Salience | Mid-term | IMPLEMENTED | Durable lifecycle for work/blockers/next actions. |
| `derived_models` | Procedural, Reflective | Mid to long-term | PARTIAL | Non-canonical truth layer unless governed promotion path validates claims. |
| `context_packet_snapshots` | Working, Structural, Episodic | Short to mid-term | IMPLEMENTED | Non-canonical restore/context evidence. |
| Dream reports | Reflective, Salience, Utility | Mid-term | IMPLEMENTED | Generated dry-run reports; no verified `dream_reports` table in current schema. |
| `restore_outcome_events` | Utility, Episodic, Reflective | Mid-term | IMPLEMENTED | Non-canonical feedback evidence used by restore scoring and Dream Mode. |
| `semantic_links` | Structural, Semantic | Long-term | IMPLEMENTED | Canonical relationships when syscall-backed. |
| `artifact_refs` | Structural, Episodic | Mid to long-term | IMPLEMENTED | Evidence references, not binary truth. |
| `provenance_records` | Structural, Episodic | Long-term | IMPLEMENTED | Trace/source lineage for memory authority. |
| `contradiction_records` | Salience, Reflective, Structural | Mid to long-term | IMPLEMENTED | Preserves disagreement; does not delete either side. |
| `supersession_records` | Reflective, Structural, Semantic | Long-term | IMPLEMENTED | Durable lineage for replacement without erasure. |
| `memory_observations` | Episodic, Working, Utility | Short to mid-term | PARTIAL | Legacy/retrieval observation layer; mutation routes are retired, but reads and repair/retrieval uses remain. |
| `memory_usefulness_events` | Utility | Mid-term | IMPLEMENTED | Retrieval memory usefulness signal; non-canonical. |
| `packet_alignment_notes` | Reflective, Utility | Mid-term | IMPLEMENTED | Packet-to-memory alignment evidence. |
| `memory_repair_runs`, `memory_repair_items` | Reflective, Utility | Mid-term | PARTIAL | Repair workflow exists, projection repair remains incomplete. |
| `embedding_records` | Structural | Mid-term derived index | IMPLEMENTED | Retrieval index only; vectors are not truth authority. |
| `memory_vsa_pointers`, `memory_vsa_role_bindings`, `memory_vsa_associations` | Structural, Utility | Mid-term derived index | PARTIAL | Derived from observations/fingerprints; export-only on restore by policy. |
| `retrieval_runs`, `retrieval_results`, `retrieval_result_selection`, `retrieval_result_vsa_signals` | Working, Utility, Structural | Short to mid-term | IMPLEMENTED | Recall evidence and scoring traces, not semantic truth. |
| `context_evidence` | Episodic, Utility | Short to mid-term | IMPLEMENTED | Evidence rows for downstream packet/job context. |
| `task_packets`, `packet_retrieval_runs` | Working, Prospective | Short to mid-term | IMPLEMENTED | Execution/context scaffolding; not semantic truth by itself. |
| `events`, `job_events`, `job_status_history` | Episodic | Short to long-term operational | IMPLEMENTED | Operational event projections; `journal_events` remains semantic truth trail. |

## Dream Mode Role

Dream Mode is the lymphatic consolidation/replay surface. In v0 it is CPU-only, dry-run by default, and proposal-only.

Dream Mode may:

- select replay candidates from journal, snapshot, outcome, note, state, loop, contradiction, and artifact evidence
- score salience deterministically
- propose tier routing, repair, promotion, demotion, or memory gap actions
- include Rule Cell advisory traces

Dream Mode must not:

- commit canonical memory
- silently promote/demote long-term truth
- treat restore outcomes or vector/VSA indexes as truth
- bypass semantic syscall validation

## Restore Scoring Role

Restore scoring is the arterial recall surface for `COMPILE_CONTEXT`.

Restore scoring may:

- rank scoped context snapshot candidates deterministically
- apply bounded Rule Cell adjustments
- apply bounded restore outcome utility adjustments
- require fresh compile when candidates are stale, low-confidence, absent, or forced fresh

Restore scoring must not:

- include wrong-workspace candidates
- use Rule Cells as the only scope boundary
- treat Utility memory as truth
- bypass fresh compile when authoritative policy or scope requires caution

## Rule Cell And Hyperlane Role

Rule Cells and Hyperlane route and score obvious decisions before heavier paths. They are deterministic reflex routing, not agent reasoning.

Memory-facing roles:

- Arterial rules can adjust restore scoring within caps and emit `FreshCompileRequired`.
- Lymphatic rules can adjust Dream salience and block risky promotion proposals.
- Kernel rules can warn/reject stricter authority violations.
- Operator rules can emit attention signals for review surfaces.

Rule Cells cannot commit truth, execute tools, call modelruntime, call the network, scan storage, or make an authoritative denial more permissive.

## Operator UI Role

Operator surfaces should expose memory by both horizon and authority:

- Short-term: active context, recent retrieval, selected evidence, current restore package.
- Mid-term: snapshots, outcome events, Dream proposals, repair items, utility feedback.
- Long-term: canonical notes, state, links, loops, supersessions, contradictions, and provenance.

Current operator visibility is PARTIAL. FORGE has views for chat, workbench, retrieval, memory, restore details, and Dream reports, but it does not yet provide a unified horizon/authority filter that makes volatile/evidence/canonical boundaries obvious at a glance.

## Gap Review

| Area | Status | Gap |
|---|---|---|
| Working memory | PARTIAL | Active context exists, but horizon-specific UI and TTL policy are not explicit. |
| Episodic memory | IMPLEMENTED | Semantic and operational event streams coexist; docs must keep `journal_events` vs `events` authority clear. |
| Salience memory | PARTIAL | Dream salience exists; operator attention surface is not fully horizon-aware. |
| Prospective memory | PARTIAL | Open loops are durable; autonomy/task prospective memory is not fully unified under one inspectable surface. |
| Reflective memory | PARTIAL | Dream and repair reports exist; governed apply/consolidation remains planned. |
| Utility memory | IMPLEMENTED for restore outcomes; PARTIAL for retrieval usefulness | Restore outcome feedback is wired into restore and Dream; retrieval usefulness is not fully unified with restore utility. |
| Semantic memory | IMPLEMENTED | Canonical path is clear; future UI should show why a claim is canonical. |
| Procedural memory | PARTIAL | Procedural notes/models exist but lack a first-class taxonomy field and review workflow. |
| Structural memory | PARTIAL | Links/artifacts/snapshots are strong; vector/VSA indexes need clearer recompute/staleness visibility. |

Additional naming gaps:

- `memory_observations` is still named as memory even though it is no longer the canonical mutation path.
- Dream reports are report payloads, not a verified durable `dream_reports` table.
- `derived_models` can sound authoritative; docs should keep "derived model never replaces evidence" visible.
- Utility/outcome memory should be described as scoring evidence, not learned truth.

## Recommended Next Pass

Implement a small operator-facing memory inspector pass:

- add horizon and authority filters to existing memory/restore/Dream inspection surfaces
- surface whether an item is volatile, non-canonical evidence, or canonical
- show restore outcome utility next to snapshot scoring
- show Dream proposal routing by memory type
- keep all promotion/demotion actions behind semantic syscalls and review

Do not add new canonical tables for this. The current schema is sufficient for a v0 horizon/authority view.
