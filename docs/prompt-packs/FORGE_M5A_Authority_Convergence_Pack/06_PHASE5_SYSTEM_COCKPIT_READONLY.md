# Phase 5 Prompt — Read-Only System Cockpit Authority Display

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Upgrade the existing System surface into a read-only authority cockpit.

This is not a new UI direction. It is the current System page evolving into a command-center view.

## Required display data

Core status, Gateway/tool authority, modelruntime authority, authority matrix summary, `model.*` alignment status, Control Lane fingerprint/seam status, FORGE-K live authority disabled/partial validation status, HostBridge snapshot age/staleness, FORGE-H advisory posture, storage truth authority, Postgres/Qdrant/Redis readiness, safe-mode flags, pending approvals count, recent warnings, and latest validation/build evidence if available.

## Backend preferred approach

Extend `GET /forge/system/status` unless a separate read-only route is cleaner.

## UX rules

Missing data displays as `not wired`, `unavailable`, `disabled`, or `unknown`. Healthy only means explicitly healthy. Stale means stale. Deferred means deferred. Advisory means advisory.

## Tests

System page renders missing data safely, authority matrix summary displays, stale labels display, no mutation buttons exist, dangerous strings are not action labels.

## WHAT NOT TO DO

Do not redesign the whole desktop. Do not introduce a new UI framework. Do not add mutation controls. Do not show secrets/raw logs/raw memory/raw prompts.
