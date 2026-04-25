# Requirements Traceability Matrix

| ID | Requirement | Source | Status | Evidence | Verification | Test Coverage | Gap / Recommendation |
|---|---|---|---|---|---|---|---|
| REQ-001 | Canonical semantic mutation must use control-lane syscalls. | Doctrine | IMPLEMENTED | `aios/controllane` | Go tests | Good | Add public facade |
| REQ-002 | Semantic journal must be append-only. | Cognitive FS | IMPLEMENTED | `journal_events` triggers | Go tests | Good | Keep migration tests |
| REQ-003 | Legacy memory mutation APIs must stay retired. | Status docs | IMPLEMENTED | API retired routes | API tests | Good | Keep guard tests |
| REQ-004 | Public syscall API should exist. | Review | MISSING | none | Not verified | Missing | Add `/api/aios/syscalls` |
| REQ-005 | Tool execution must be gateway-only. | Authoritative paths | IMPLEMENTED | `gateway.Execute` | API/gateway tests | Good | Keep side-door tests |
| REQ-006 | Legacy adapter invoke must remain gateway-mediated. | Status docs | IMPLEMENTED | `legacy_adapter_gateway_tool.go` | API tests | Good | Monitor routes |
| REQ-007 | Approval grants must bind to request fingerprint. | Review | IMPLEMENTED | `gateway/service.go` | Gateway tests | Good | Extend to config/model actions |
| REQ-008 | Dangerous tools default approval-only or disabled. | Dangerous capabilities | MOSTLY | registry/policy | Gateway tests | Good | Add dependency failure tests |
| REQ-009 | Gateway status changes must be audited. | Doctrine | PARTIAL | capability API | API tests | Partial | Add approval policy |
| REQ-010 | Permission profile privilege expansion requires governance. | Review | RISK | `permissions`, API handlers | Not complete | Partial | Approval-gate expansion |
| REQ-011 | Lane authority changes require governance. | Review | RISK | lanes/API | Not complete | Partial | Approval-gate changes |
| REQ-012 | Settings/source mutations need consistent audit. | Review | PARTIAL | API handlers | API tests | Partial | Audit policy matrix |
| REQ-013 | Modelruntime owns inference. | Model runtime docs | IMPLEMENTED | `modelruntime.Service` | Go tests | Good | Keep no-bypass tests |
| REQ-014 | Model import/register is governed. | Review | RISK | management APIs | Tests lack approval | Partial | Add model capabilities |
| REQ-015 | Model load/unload/archive/remove audit is traceable. | Review | PARTIAL | model API/bridge | Go tests | Partial | Add approval trace |
| REQ-016 | Destructive model file delete requires approval. | Roadmap | DEFERRED | no delete flow | Not applicable | Missing | M4 design |
| REQ-017 | Streaming must be governed before enablement. | Roadmap | DEFERRED | non-streaming | Not applicable | Missing | M4 tests |
| REQ-018 | `/v1/*` API is explicit opt-in. | Config docs | PARTIAL | config/API | API tests | Partial | Add exposure/auth tests |
| REQ-019 | Provider URLs are SSRF-hardened. | Security review | RISK | config/backend | Missing | Missing | Allowlist policy |
| REQ-020 | Provider secrets are redacted. | Security review | MISSING TEST | config/audit | Missing | Missing | Redaction tests |
| REQ-021 | CPU/RAM kernel must not require GPU. | CPU/GPU docs | IMPLEMENTED | config/health docs | Tests partial | Partial | Add regression |
| REQ-022 | GPU telemetry must not be truth authority. | CPU/GPU docs | IMPLEMENTED DOCTRINE | gpu/modelruntime | Docs | Partial | Add tests |
| REQ-023 | Current/historical truth must be separate. | Cognitive FS | IMPLEMENTED | `state_items`, `state_versions` | Go tests | Good | UI surfacing |
| REQ-024 | Canonical hard delete is not normal path. | Doctrine | IMPLEMENTED/PARTIAL | archive/supersede | Tests partial | Partial | Negative tests |
| REQ-025 | Backup covers cognitive FS tables. | Backup docs | MOSTLY | `backup/service.go` | Backup tests | Good | close export-only sections |
| REQ-026 | Backup includes retrieval evidence. | Review | MISSING/PARTIAL | retrieval tables | Missing | Missing | Add sections |
| REQ-027 | Backup includes observation/usefulness evidence. | Review | MISSING/PARTIAL | memory tables | Missing | Missing | Add sections |
| REQ-028 | Restore verifies bundle integrity. | Review | RISK | backup service | Missing | Missing | Hash/count checks |
| REQ-029 | Restore atomicity limits are explicit. | Durability docs | PARTIAL | restore response | Backup tests | Partial | UI warning |
| REQ-030 | VSA files stay tracked/preflighted. | AGENTS | IMPLEMENTED | scripts/package | Build/test | Good | Keep preflight |
| REQ-031 | Context snapshots are non-canonical. | Semantic docs | IMPLEMENTED | `context_packet_snapshots` | Tests | Good | UI label |
| REQ-032 | Restore scoring ranks beyond exact query. | Review | PARTIAL/RISK | scoring code | Tests incomplete | Partial | Fix candidate listing |
| REQ-033 | Restore fallback reason visible. | Review | PARTIAL | metadata/API | Partial | Partial | UI inspector |
| REQ-034 | Header-first restore proves bounded loading. | Review | PARTIAL | package metadata | Partial | Partial | Large snapshot tests |
| REQ-035 | Snapshot card retention policy exists. | Review | MISSING | none | Missing | Missing | Retention design |
| REQ-036 | Dream Mode defaults dry-run. | Dream docs | IMPLEMENTED | dream service | Go tests | Good | Preserve default |
| REQ-037 | Dream reports persist as evidence. | Review | MISSING | none | Missing | Missing | Add tables/API |
| REQ-038 | Dream commits use syscalls. | Doctrine | PLANNED | no commit mode | Future tests | Missing | Design v1 |
| REQ-039 | Dream UI review workflow exists. | Roadmap | MISSING | desktop partial | Missing | Missing | Add page/workflow |
| REQ-040 | Autonomy requires charter/intent/budget/policy. | Autonomy docs | PARTIAL | autonomy package | Tests | Partial | Expand traces |
| REQ-041 | Autonomy tool calls consume budget. | Review | RISK | gateway authorizer | Missing | Missing | Budget tests |
| REQ-042 | Rule agents propose only. | Doctrine | PARTIAL SAFE | rule agents | Safety tests | Partial | Rule Cell v0 |
| REQ-043 | Truth engine explains current/history/conflict. | Cognitive FS | PARTIAL | `truth/engine.go` | Go tests | Partial | UI/API surfacing |
| REQ-044 | `events` is operational projection. | Status docs | PARTIAL | events/journal | Docs | Partial | Terminology pass |
| REQ-045 | Trace unifies chat/gateway/syscall/audit/artifacts/model/snapshots. | Review | PARTIAL | trace report/API | Tests partial | Partial | Trace workbench |
| REQ-046 | Desktop has unit/component/e2e tests. | Test review | MISSING | none | Missing | Missing | Vitest/Playwright |
| REQ-047 | Scripts work cross-platform. | User/Windows | PARTIAL/BROKEN | package scripts | Build works | Partial | Convert Bash scripts |
| REQ-048 | Nix foundation is verified before hard claims. | Status docs | BLOCKED | nix files | Not verified | Missing | Validate on Nix host |
| REQ-049 | Cloud providers do not silently fallback. | Doctrine | PARTIAL | config defaults | Tests partial | Partial | More provider tests |
| REQ-050 | Operator can inspect approval/audit/gateway state. | UI docs | PARTIAL | desktop pages | Build/typecheck | Partial | Trace-first UX |

