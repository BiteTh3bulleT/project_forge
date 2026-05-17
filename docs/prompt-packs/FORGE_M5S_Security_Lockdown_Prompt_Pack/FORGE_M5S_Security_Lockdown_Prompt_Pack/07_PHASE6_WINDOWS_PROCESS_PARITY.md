# Phase 6 — Windows Process Tool Parity

Remove unconditional Unix assumptions from process tools.
No unconditional `bash -lc` or `kill` on Windows.
Use build tags or platform-specific implementation.
Unsupported shell semantics return structured unsupported errors.
