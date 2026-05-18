# PHASE 9 — Telemetry and Performance Governor

## Objective

Add metrics, benchmarks, and governor behavior around time-to-useful-context and cache/worker efficiency.

## Instructions

Implement metrics, local benchmarks, governor actions for suspend prewarm, coalesce duplicates, throttle low-priority work, fresh compile fallback, and cache bypass.

## Validation

Benchmarks run locally. Metrics emitted. Backpressure testable.

## What not to do

Do not claim performance wins without measurement. Do not optimize by weakening validation.

## Exit gate

Exit when performance can be measured and governed safely.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
