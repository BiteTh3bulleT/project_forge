// Package controllane is the FORGE kernel: the single semantic-write
// authority for canonical cognitive state (memory notes, semantic links,
// state items/versions, open loops, contradictions, supersessions, derived
// models, context packet snapshots, artifact refs, journal events).
//
// Authority invariants enforced by tests in this package:
//
//   - Direct INSERT/UPDATE/DELETE against canonical cognitive tables is
//     forbidden outside three allow-listed paths: this package's
//     sqlite_store.go, backup/service.go (restore-only), and
//     store/migrate.go (schema migrations). Enforced by
//     TestCanonicalCognitiveWritesStayBounded.
//
//   - controllane.NewProcessor has exactly one production call site.
//     Adding a second is a structural change that requires updating
//     authority_guard_test.go's allow-list and docs/status/duplicate_systems.md.
//     Enforced by TestKernelProcessorHasSingleConstructionSite.
//
// Adapters, autonomy rule cells, and any future IRIS bridge propose
// SyscallRequests; the kernel validates and commits. Do not bypass.
package controllane
