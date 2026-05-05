# FORGE-K Phase 11F Live Path Mapping

Status: Phase 11F `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`.

This mapping identifies current live daemon authority paths and their FORGE-K target components. It is diagnostic only. Live mutation allowed is `NO` for every row in this phase.

| Live system | Current live authority | FORGE-K target component | Integration risk | Required adapter | Required tests | Migration status | Live mutation allowed? |
|---|---|---|---|---|---|---|---|
| api routes | `services/core/internal/api`; live HTTP router and service composition for API, gateway, model runtime, memory, retrieval, embeddings, dream, backup, release, and settings surfaces. | Kernel ingress shell and subsystem routing boundary. | High | ReadOnlyLiveStateAdapter | Route inventory; no route behavior change tests. | Not started | NO |
| aios/controllane | `services/core/internal/aios/controllane`; semantic syscall processor with validation, transaction runner, journal append, and audit sink. | Kernel and semantic syscalls. | High | ReadOnlyLiveStateAdapter | Syscall parity; journal boundary; no bypass tests. | Not started | NO |
| gateway | `services/core/internal/gateway`; live tool execution authority with lane resolution, permissions, approval state, invocation records, artifacts, and audit. | Neuron Fabric / execution boundary. | High | LiveGatewayTraceAdapter | Approval gate parity; no tool execution from shadow tests. | Not started | NO |
| permissions | `services/core/internal/permissions`; active profile policy for read/write/execute/network/path/tool/risk checks. | Capability policy. | Medium | ReadOnlyLiveStateAdapter | Capability parity; denied-risk mirror tests. | Not started | NO |
| lanes | `services/core/internal/lanes`; bounded operation lanes with action type, allowed paths, write intent, risk, artifacts, and approval requirements. | Lane and syscall scope contracts. | Medium | ReadOnlyLiveStateAdapter | Lane mirror; risk scope tests. | Not started | NO |
| audit | `services/core/internal/audit`; append-only live audit records with correlation, job, gateway, approval, risk, and outcome fields. | Journal/provenance evidence. | Medium | LiveAuditMirrorAdapter | Audit mirror provenance; trace lineage tests. | Not started | NO |
| modelruntime | `services/core/internal/modelruntime`; live model registry, backend lifecycle, generation, chat routing, scheduler, GPU policy, and audit. | Runtime Boundary. | High | LiveModelRuntimeTraceAdapter | No modelruntime call tests; runtime trace mirror tests. | Not started | NO |
| memory | `services/core/internal/memory`; legacy observations, VSA records, utility signals, and evidence/projection surfaces. Direct mutation HTTP routes are retired. | Memory Palace and Courthouse evidence. | High | LiveMemoryMirrorAdapter | No memory write; memory provenance; canonical-promotion tests. | Not started | NO |
| retrieval | `services/core/internal/retrieval`; hybrid keyword/semantic/VSA retrieval runs, ranked results, selection reasons, usefulness, and context evidence. | Memory Palace. | High | LiveRetrievalMirrorAdapter / ReadOnlyRAGAdapter | No retrieval execution; retrieval provenance; evidence-only tests. | Not started | NO |
| search | `services/core/internal/search`; SQLite FTS read path over indexed chunks/files. | Memory Palace route evidence. | Medium | LiveSearchTraceAdapter | Search trace mirror; no query execution from FORGE-K tests. | Not started | NO |
| embeddings | `services/core/internal/embeddings`; embedding provider abstraction and records used by semantic retrieval. | Memory Palace structural evidence and KV metadata. | High | LiveEmbeddingTraceAdapter / ReadOnlyRAGAdapter | No embedding provider call; trace provenance tests. | Not started | NO |
| dream/autonomy | `services/core/internal/aios/dream` and autonomy surfaces; deterministic dry-run reports and autonomy planning. | Lymphatic Lane and Consensus Mesh. | Medium | ReadOnlyLiveStateAdapter | Non-canonical report; no cleanup mutation tests. | Not started | NO |
| backup/release | `services/core/internal/backup` and `services/core/internal/release`; portable bundles, restore surfaces, release readiness, and records. | Snapshots and Lymphatic Lane. | Medium | LiveMemoryMirrorAdapter | Restore non-execution; release evidence mirror tests. | Not started | NO |
| settings/config | `services/core/internal/config`; environment-derived runtime policy and settings. | Kernel policy substrate. | Medium | ReadOnlyLiveStateAdapter | Config mirror; policy provenance tests. | Not started | NO |

## Authority Classes

The live system currently has distinct authority classes:

- Gateway owns live tool execution authority.
- AI-OS/control lane owns live semantic cognitive write behavior where integrated.
- Permissions and lanes own live gate/scope policy.
- Audit owns live trace/provenance records.
- Modelruntime owns live model driver governance, not truth.
- Retrieval, search, embeddings, and memory are evidence/context infrastructure unless explicitly routed through a governed semantic syscall path.

## Phase 11F Boundary

FORGE-K may only define read-only mappings and future adapter contracts in this phase. It must not import these live packages from `services/core/internal/forgek/integrationready`, execute live behavior, query retrieval or embeddings, write memory, affect user-visible output, or create a second authority path.
