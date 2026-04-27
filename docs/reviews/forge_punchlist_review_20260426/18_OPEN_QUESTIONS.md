# Open Questions

| Question | Why It Matters | Current Best Answer | Needed Evidence |
|---|---|---|---|
| Should core bind loopback-only by default? | Local-first safety. | Yes. | Config/main tests and operator docs. |
| Should Dream reports be append-only? | Evidence integrity. | Not decided; current behavior upserts by ID. | Product/security decision + tests. |
| Which evidence tables need DB immutability triggers? | Historical truth integrity. | `journal_events` only today. | Migration design. |
| Should embedding records restore exactly or rebuild? | Vector/index truth boundary. | VSA is export-only; embeddings restore today. | Explicit policy decision. |
| Should modelruntime remote calls have budgets? | Cloud cost/egress control. | Yes for M4. | Governance design/tests. |
| Is `/api/process/health` a trace endpoint or global health? | Operator clarity. | Trace endpoint today. | Endpoint naming/UI decision. |
| Should `/forge/*` routes move under `/api`? | Client consistency. | Not urgent. | API versioning plan. |
| What is the first real Rule Cell substrate? | Avoid agent sprawl. | Deterministic registry with trace/no-mutation. | Design pass. |
| Should public syscall API be exposed? | External proposer integration. | Yes, bounded dry-run/submit/inspect. | Security/authority design. |
| How should Windows smoke be supported? | Current review/validation is Windows-hosted. | Add Node/PowerShell smoke. | Script pass. |

