# FORGE → IRIS transfer notes

FORGE is intentionally layered so **durable mechanisms** can be promoted into IRIS without dragging along UI chrome or product-specific copy.

## Transfer-friendly modules (generic)

- **Gateway + lanes + permission profiles**: execution contract pattern (scope, risk, approval, audit) is reusable.
- **Audit record schema**: correlation-first tracing maps cleanly to enterprise audit sinks.
- **Backup bundle format**: JSON bundles with explicit `schema` version and `entities` map — portable across deployments.
- **Release readiness checklist**: operational pattern, not FORGE-brand-specific.

## FORGE-specific (review before reuse)

- Adapter ids (`ollama`, `codex`, `claude_code`) and job templates tied to FORGE UX.
- “Smith” tone strings in the desktop shell.
- Workspace-relative defaults where `FORGE_WORKSPACE_DIR` is implicit operator context.

## Suggested export order for IRIS intake

1. `portable_snapshot` or `full_backup` bundle from **Backup / Export**.
2. `audit_history` bundle for accountability.
3. `policy_profiles` + `strategies` + `automation_rules` if policy automation should be mirrored.
4. `release` artifact records as packaging provenance (optional).

Keep IRIS-side ingestion **idempotent** and version-aware using bundle `schema` + `versionTag`.
