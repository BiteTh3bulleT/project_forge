# ADR Prompt: FORGE-T Temporal Tuner

Create an ADR for FORGE-T as a system-level temporal tuner beside FORGE-HMK.

Explain why it sits beside FORGE-HMK, its job scheduling, cache TTL, retry/backoff, worker cadence, replay window, backpressure, performance metrics, and authority limits.

Decision: FORGE-T governs timing and performance, but cannot commit memory truth.
