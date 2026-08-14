// Package controllane is the temporary durable SQLite port implementation
// behind the production FORGE-K authority. It persists canonical cognitive state
// (memory notes, semantic links, state items/versions, open loops,
// contradictions, supersessions, derived models, context packet snapshots,
// artifact refs, journal events). K20B moved orchestration into FORGE-K-owned
// port stages; this package still supplies validation, apply, and persistence.
//
// Authority invariants enforced by tests in this package:
//
//   - Direct INSERT/UPDATE/DELETE against canonical cognitive tables is
//     forbidden outside three allow-listed paths: this package's
//     sqlite_store.go, backup/service.go (restore-only), and
//     store/migrate.go (schema migrations). Enforced by
//     TestCanonicalCognitiveWritesStayBounded.
//
//   - controllane.NewProcessor has exactly one production assembly site and is
//     selected behind internal/forgekernel at daemon boot. Adding a second is a
//     structural change that requires updating
//     authority_guard_test.go's allow-list and docs/status/duplicate_systems.md.
//     Enforced by TestKernelProcessorHasSingleConstructionSite.
//
// Adapters, autonomy rule cells, and any future IRIS bridge propose
// SyscallRequests; production FORGE-K owns ingress and stage orchestration while
// this package applies the durable transaction until later subsystem and store
// extraction phases retire the compatibility implementation. Do not bypass
// either boundary.
package controllane
