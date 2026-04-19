# FORGE - AI-OS Architecture Baseline (Phase 1)

## Decision framing

FORGE is the AI-OS.

IRIS is a future resident semantic service inside FORGE. IRIS is not the OS and cannot own canonical truth.

FORGE owns:

- kernel logic
- memory/state authority
- workspace scope
- runtime and process lifecycle
- permissions and capabilities
- tool execution boundary
- audit and trace
- context compilation

Rule of operation:

> IRIS proposes. FORGE validates. FORGE commits.

## Existing FORGE doctrine mapped to AI-OS terms

| FORGE doctrine | AI-OS interpretation | Current implementation evidence |
|---|---|---|
| events are truth | system journal | `events`, `job_events`, `audit_records` tables; `internal/events`, `internal/jobs`, `internal/audit` |
| jobs are projections | process lifecycle projection | `jobs` + `job_status_history`; `internal/jobs` |
| packets are contracts | execution contract + working set | `task_packets`; `internal/packets` |
| approvals are gates | kernel gate decisions before risky execution | `approval_requests`, `approval_decisions`; `internal/approvals`, `internal/permissions` |
| artifacts are evidence | durable execution evidence | `artifacts`, `backup_bundles`, `release_artifacts`; `internal/artifacts`, `internal/backup`, `internal/release` |
| adapters are bounded workers | user-space workers behind contracts | `internal/adapters` registry + typed invoke contract |
| gateway is controlled boundary | controlled I/O + execution syscall boundary | `internal/gateway` + `gateway_invocations` |
| action lanes | permissioned execution lanes | `internal/lanes` + `action_lanes` |
| audit | kernel/system trace | `internal/audit` + correlation-id trace APIs |

## Tri-lane architecture (target on current repo)

### Control Lane (kernel lane)

Responsibilities:

- semantic syscall validation
- permissions/capabilities checks
- approvals/gates
- state transitions and lifecycle enforcement
- audit append

Current modules in this lane:

- `internal/jobs`
- `internal/approvals`
- `internal/permissions`
- `internal/lanes`
- `internal/gateway` (validation/decision path)
- `internal/audit`

### I/O Lane

Responsibilities:

- tool gateway invocation
- adapter ingress/egress
- artifact writes and export/import
- event ingestion of external sources

Current modules in this lane:

- `internal/gateway`
- `internal/adapters`
- `internal/artifacts`
- `internal/imports`
- `internal/ingest`
- `internal/backup`

### Compute Lane

Responsibilities:

- internal librarian cells (retrieval/ranking/repair workers)
- embedding/index workers
- pattern/model workers
- context compiler
- future semantic inference seam for IRIS

Current modules in this lane:

- `internal/retrieval`
- `internal/embeddings`
- `internal/memory`
- `internal/packetopt`
- `internal/failurepatterns`
- `internal/policy`
- `internal/projectcontext`

Gap note: the lane boundary exists conceptually today but is not yet enforced by package-level kernel/compute namespaces. That is a forward phase item.

## Cognitive filesystem model (FORGE flavor)

The cognitive filesystem in FORGE is inspectable storage and references, not an opaque memory blob.

| Cognitive filesystem element | Current repo mapping | Status |
|---|---|---|
| raw event journal | `events`, `job_events`, `audit_records` | implemented |
| notes | `canvas_notes`, packet/project notes fields, dossier briefs | partial (distributed, no unified note taxonomy yet) |
| typed links | `memory_observation_links`, `job_lineage` | implemented (initial) |
| active state | `jobs.status`, `approval_requests.status`, active permission profile | implemented |
| open loops | pending approvals/reviews/reconciliation states | partial (no dedicated `open_loops` domain yet) |
| artifact refs | `artifacts`, `context_evidence`, packet/retrieval link tables | implemented |
| derived models | `routing_policy_recommendations`, `packet_guidance_records`, `failure_pattern_snapshots`, `routing_insights` | implemented (advisory) |
| context packets | `task_packets` + packet retrieval/alignment links | implemented |

## Exact truth vs derived policy

Exact truth (authoritative evidence) lives in:

- events and job/audit journals
- memory observations and links
- active lifecycle state
- approvals and decisions
- artifacts and references

Derived policy/model layers live in:

- routing recommendations
- packet guidance scores
- failure pattern snapshots
- strategy/policy presets

Derived layers are adaptive and can change. They are not canonical truth.

## Kernel, heuristics, and user space boundary

- LLM/generative systems are user-space proposers.
- FORGE kernel rules are deterministic validators and committers.
- Heuristics (ranking/scoring/prioritization) sit between them and remain advisory.

No LLM or future IRIS service may directly mutate canonical state.

All durable writes must pass through validated FORGE APIs/syscalls (`jobs`, `gateway`, `approvals`, `packets`, and related services).

## Current vs target (phase-indexed)

| AI-OS primitive | Existing FORGE concept | Current status | Phase needed |
|---|---|---|---|
| system journal | `events`, `job_events`, `audit_records` | implemented and append-style | harden replay/projection invariants in Phase 2 |
| process projections | `jobs` + `job_status_history` | implemented | unify with kernel process semantics in Phase 2 |
| execution contracts | `task_packets` | implemented | context packet schema expansion in Phase 6 |
| approval gates | `approval_requests` + `approval_decisions` | implemented | wire all gateway approval flows end-to-end in Phase 2 |
| evidence filesystem | `artifacts`, `backup_bundles`, `release_artifacts` | implemented | converge artifact refs into cognitive fs model in Phase 3 |
| bounded workers | `internal/adapters` | implemented | maintain bounded contract; no direct state writes (ongoing) |
| controlled I/O boundary | `internal/gateway` + `gateway_invocations` | implemented | formal semantic syscall surface in Phase 2 |
| execution lanes | `action_lanes` + permission profiles | implemented | lane taxonomy aligned to 3-lane OS model in Phase 1/2 |
| permission kernel | `internal/permissions` | implemented | capability model and syscall policy coupling in Phase 2 |
| audit trace | `internal/audit` trace by correlation id | implemented | expand observability/API surfaces in Phase 9 |
| cognitive filesystem | memory/notes/links/state spread across tables | partial, distributed | normalize persistence model in Phase 3 and Phase 5 |
| internal librarian cells | retrieval/memory/repair services | partial, functionally present | formal cell runtime and scheduling in Phase 4 |
| context compiler | project context + packet assembly | partial | full context compiler contract in Phase 6 |
| workspace/runtime isolation | workspace paths + permission scopes | partial | explicit workspace runtime/event bus in Phase 7 |
| adaptive policy algebra | policy + guidance + insights snapshots | partial advisory | algebraic/adaptive layer in Phase 8 |
| IRIS integration seam | none as first-class service yet | planned only | seam + eval harness in Phase 10 |

