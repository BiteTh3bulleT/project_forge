# Phase 6 Prompt — Validation, CI, and Performance Gates

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Make M5A hard to regress.

## Required tests

Add tests for authority matrix coverage, modelruntime/gateway metadata alignment, `model.delete_file` current status, `model.chat` and `model.generate` authority posture, Control Lane fingerprint determinism, HostBridge/FORGE-H cache behavior, and System status shape.

## Performance baseline

Create/update:

- `docs/status/m5a_latency_baseline.md`

Include measured or placeholder-to-be-measured sections for `/forge/system/status`, HostBridge snapshot, FORGE-H policy build, modelruntime chat, Gateway low-risk read tool, Control Lane validation syscall, backup restore dry-run if available.

Do not invent measurements. If not measured, mark `NOT_MEASURED`.

## Commands to attempt

```bash
npm test
npm run lint
npm run validate:js
npm run validate:local
npm run build
npm run smoke
```

## WHAT NOT TO DO

Do not fake performance numbers. Do not mark skipped checks as passed. Do not require unavailable external services for normal local validation.
