# Phase 1 — API Auth and Bind Policy

Add backend auth to all non-health routes.

Requirements:
- `/health` public.
- Everything else token-protected.
- `Authorization: Bearer <token>`.
- token from env/file/generated first-run file.
- wildcard bind requires auth.
- Docker no longer defaults to unauth wildcard.

Tests:
- health public.
- non-health unauth rejected.
- valid token accepted.
- wildcard bind without token rejected.
