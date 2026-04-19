# FORGE AI-OS Scaffolds (Phase 1-2)

This package tree introduces repo-native scaffolding for the AI-OS lane model without replacing existing services.

## Lane scaffolds

- `controllane/`: deterministic semantic syscall kernel (registry, validator, capability checks, approval gate, transaction boundary, audit sink)
- `iolane/`: gateway/adapters/artifacts/import-export interfaces
- `computelane/`: inference/context/model/retrieval interfaces + IRIS seam
- `compute/librarian/`: internal librarian cell contracts
- `domain/`: shared AI-OS primitives and syscall contracts used by lane interfaces

## Design rule

Cells and semantic services can propose candidate semantic actions.

FORGE Control Lane validates and commits.

No cell or future IRIS seam implementation in this scaffold writes canonical state directly.

## Scope note

Phase 1 established interfaces and domain primitives.

Phase 2 adds a deterministic Control Lane syscall processor using in-memory transactional storage for testability.

Existing runtime behavior in current modules (`internal/jobs`, `internal/gateway`, `internal/permissions`, `internal/audit`, `internal/retrieval`, `internal/memory`, etc.) remains authoritative until later phases complete persistence and integration cutovers.
