# Phase 4 — CORS, Remote Tokens, Secret Hardening

CORS default: exact Tauri/configured origins only.
Localhost wildcard only under explicit dev flag.

Secrets:
- redact settings reads,
- do not log generated tokens,
- document OS keychain future in `docs/architecture/secret_storage.md` if not implemented.

Tests:
- random localhost rejected by default.
- configured/Tauri origins accepted.
- secrets redacted.
