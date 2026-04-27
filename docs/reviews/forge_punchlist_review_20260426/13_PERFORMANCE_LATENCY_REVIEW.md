# Performance / Latency Review

## Risks

RISK: Restore scoring may degrade with many snapshots if candidate listing broadens without matching indexes.

RISK: `proc.run` captures output into memory before final truncation.

RISK: Modelruntime scheduler is simple in-memory FIFO with limited durable accounting.

RISK: Provider health/telemetry calls can become visible latency if pulled into hot UI polling loops.

PARTIAL: Dream Mode is CPU-only and deterministic, but scheduling policy and cadence are not yet a mature background workload model.

PARTIAL: UI bundle exceeds Vite's 500 kB warning threshold after minification.

## Recommendations

- Add restore candidate query benchmarks and indexes.
- Cap process output while streaming from child process, not after buffering.
- Add modelruntime queue latency metrics and timeout tests.
- Keep Dream jobs CPU-side and background-only unless explicitly operator-triggered.
- Cache provider capability/health summaries with short TTLs for UI.
- Consider route-level code splitting for desktop pages.

## Benchmarks To Add

- Context restore candidate ranking with 1k/10k snapshots.
- Backup restore integrity verification on large bundles.
- Gateway process output stress.
- Modelruntime queue admission/cooldown latency.
- Desktop initial bundle/load measurement.

