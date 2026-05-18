# PHASE 2 — FORGE-T Temporal Tuner

## Objective

Implement scheduler-adjacent timing/performance governor for memory jobs and worker leases.

## Instructions

Implement job admission, dedupe/coalescing, leases, heartbeat, timeout, retry/backoff, cancellation, priority aging, TTL hooks, and backpressure. Start with in-memory queue.

## Validation

Duplicate jobs coalesce. Timed-out leases requeue. Cancelled jobs cannot complete. No canonical writes.

## What not to do

Do not create infinite worker loops. Do not run jobs without budgets. Do not prewarm everything yet.

## Exit gate

Exit when FORGE-T can safely admit, dedupe, lease, retry, cancel, and observe jobs.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
