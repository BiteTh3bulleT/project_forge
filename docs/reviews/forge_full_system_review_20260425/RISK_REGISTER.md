# Risk Register

| Severity | Label | Risk | Evidence | Recommendation |
|---|---|---|---|---|
| High | RISK | Model management mutates runtime registry/filesystem metadata outside gateway-equivalent approval. | `/forge/models/import`, `/archive`, `/remove`, modelruntime store management | Add approval gates or model gateway capabilities. |
| High | RISK | Gateway approval IDs can be reused without request fingerprint binding. | `gateway/service.go`, `approvalGrantPresent` | Bind approval to job/tool/lane/capability/paths/risk/actor. |
| High | RISK | Backup misses retrieval/observation evidence. | backup sections vs retrieval/memory tables | Add backup/restore sections and tests. |
| Medium/High | RISK | Restore accepts structurally valid bundle without hash/entity verification. | `backup/service.go` restore path | Verify hash/counts/signature before applying. |
| Medium | PARTIAL | Context scoring over-filters SQLite candidates by exact query. | `ListContextSnapshots` query filter | Candidate list by scope/kind/recency, then rank query similarity. |
| Medium | PARTIAL | Dream Mode reports are ephemeral. | `aios/dream/service.go`, no dream table | Persist non-canonical dream reports/proposals. |
| Medium | RISK | Authority-adjacent config APIs lack uniform approval/audit. | lanes, permission profiles, sources/settings | Approval/audit policy for privilege-expanding changes. |
| Medium | RISK | Provider URL/SSRF posture is thin. | TEI/OpenAI-compatible endpoints | Add endpoint allow policy and secret redaction tests. |
| Medium | MISSING | Frontend tests are absent. | no `*.test.tsx`/`*.spec.tsx` | Add Vitest/RTL and Playwright smoke. |
| Medium | BROKEN | Smoke/desktop helper npm scripts remain Bash-only on Windows. | `npm run smoke`, `desktop:check`, `desktop:clean-port` | Convert to Node wrappers or PowerShell-compatible scripts. |
| Low/Medium | PARTIAL | Nix foundation cannot be verified here. | `nix` unavailable | Validate on Nix-capable host and publish evidence. |
| Low | PARTIAL | Large desktop JS bundle warning. | Vite warning ~646 kB | Code split major pages. |

