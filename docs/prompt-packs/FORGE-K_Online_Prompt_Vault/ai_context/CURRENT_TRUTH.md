# Current Truth

FORGE-K is target cognitive microkernel architecture with simulator implementation, read-only/shadow diagnostics, storage infrastructure foundations, and partial live validation seams.

Current live daemon authority remains outside FORGE-K simulator services:

- API routes: `services/core/internal/api`
- Semantic mutation: `services/core/internal/aios/controllane`
- Tool execution: `services/core/internal/gateway`
- Policy gates: `services/core/internal/permissions` and `services/core/internal/lanes`
- Audit/provenance: `services/core/internal/audit`
- Model generation/governance: `services/core/internal/modelruntime`
- Retrieval/search/embeddings/memory: existing live packages and governed commit paths

## Current target

Get FORGE-K fully online by migrating one authority seam at a time.

## Current danger

The most dangerous mistake is importing simulator services directly into live daemon paths and calling that "integration." That creates a second authority path and invalidates the architecture.
