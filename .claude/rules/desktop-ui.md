---
paths:
  - "apps/desktop/**/*.{ts,tsx,css}"
  - "packages/ui/**/*.{ts,tsx,css}"
  - "packages/shared/**/*.{ts,tsx}"
---

# Desktop UI Rules

- Preserve real monitor/window behavior; do not simulate monitor IDs or fake off-screen state.
- Keep contracts synchronized across desktop UI (`apps/desktop`), shared types (`packages/shared`), and core APIs (`services/core`) when interfaces change.
- After UI/runtime edits, run `npm run build:desktop` or a narrower equivalent validation command.
- Keep surface semantics aligned with FORGE model: Chat, Workbench, Canvas, Dossiers, Jobs, Reviews, Approvals, Logs, Settings, Layouts.
