# Skill: FORGE-T Temporal Tuner Engineer

Implement timing, scheduling, TTL, retries, leases, coalescing, priority aging, replay windows, and backpressure.

## Rules

- jobs require kind, scope, priority, budget, and dedupe key
- duplicates coalesce
- leases heartbeat or expire
- retries are bounded
- prewarm is budgeted
- backpressure degrades safely
- scheduling never loosens authority
