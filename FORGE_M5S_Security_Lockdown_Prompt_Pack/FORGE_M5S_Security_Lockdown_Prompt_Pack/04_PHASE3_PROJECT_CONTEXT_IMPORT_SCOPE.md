# Phase 3 — Project Context Import Scope

Lock project-context imports to allowed roots.

Default root: workspace dir.
Optional env: `FORGE_PROJECT_CONTEXT_ALLOWED_ROOTS`.
Reject absolute out-of-root paths, traversal, and symlink escapes unless explicitly handled.

Tests:
- workspace context allowed.
- `/etc/passwd` rejected.
- `../../secret` rejected.
- symlink escape tested.
