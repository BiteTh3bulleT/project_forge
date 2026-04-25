# Open Questions

| Question | Why It Matters | Current Best Answer | Needed Evidence |
|---|---|---|---|
| Should model management go through gateway `model.*` capabilities or parallel approval policy? | Avoid duplicate authority models | Prefer gateway capabilities unless mismatch is proven | Design + tests |
| Is observation memory compatibility-only or fully governed memory? | Backup/restore and truth semantics | Treat mutation as retired; reads as compatibility | Product decision |
| What is the canonical operator syscall workflow? | Needed for safe semantic writes | Public dry-run-first API plus UI | API/UX design |
| Should `/v1/*` be loopback-only by default? | Security boundary | Keep explicit opt-in and local-first default | Exposure policy |
| What retention applies to snapshots and Dream reports? | Storage scalability | Add retention/compaction policy | Ops design |
| How should restore outcome feedback feed scoring? | Restore quality | Add non-canonical feedback evidence | Schema/API design |
| What is the minimum Rule Cell substrate? | Avoid agent sprawl | Deterministic rule packs with traces | RFC |
| How should frontier models be escalated? | Cost/safety | Governed modelruntime/provider policy | Provider policy |
| What UI is required before autonomy maintain mode is trustworthy? | Operator control | Trace-first workbench and budget views | UX tests |
| What must be true before external demo posture? | Safety and credibility | Model governance, backup parity, UI trace, frontend tests | Release gate |
| Should backup restore require approval like a dangerous tool? | It mutates durable state outside gateway | Likely yes or explicit operator-confirmed administrative gate | Policy design |
| Should shared TS contracts be generated from Go/API schemas? | Prevent drift such as approval expiry status | Prefer generated or contract-tested schemas | Contract tests |
