---
paths:
  - "services/core/**/*.go"
  - "services/core/go.mod"
  - "services/core/go.sum"
---

# Core Go Rules

- Preserve FORGE kernel/approval/audit invariants from `AGENTS.md`; do not bypass deterministic commit gates.
- Keep event/history behavior append-only where the architecture requires it.
- Prefer typed, minimal changes with explicit error handling and traceability fields.
- For core behavior changes, run `cd services/core && go test ./...` before finalizing.
