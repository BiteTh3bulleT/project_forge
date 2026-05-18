# Review Prompt: FORGE-HMK Performance Audit

Review FORGE-HMK and FORGE-T for performance bottlenecks.

Check time-to-useful-context, cache hit/miss ratios, duplicate job rate, worker utilization, queue wait p95, stale/dirty cache blocks, synapse traversal depth, replay window cost, prewarm success/waste, and backpressure behavior.

Output top bottlenecks, quick wins, refactor candidates, missing metrics, benchmark gaps, and over-optimization risks.

Do not weaken validation for speed.
