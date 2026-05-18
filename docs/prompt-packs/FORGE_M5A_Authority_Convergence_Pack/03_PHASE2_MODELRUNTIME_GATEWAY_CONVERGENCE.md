# Phase 2 Prompt — Modelruntime / Gateway Policy Convergence

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Eliminate drift between modelruntime route reality and Gateway capability metadata.

## Known drift to investigate

- `model.delete_file`: implemented in modelruntime governance but may be marked deferred/future in Gateway.
- `model.chat` and `model.generate`: Gateway metadata may be approval-sensitive while API execution lives under modelruntime.

## Recommended M5A posture

Modelruntime owns `/forge/models*` and `/v1/*` model execution governance. Gateway capability registry is policy-visible taxonomy unless a shared capability evaluator already exists cleanly.

If using this posture:
- mark `model.chat` and `model.generate` metadata honestly,
- authority matrix must say modelruntime owns these routes,
- docs must explicitly state modelruntime governance applies,
- tests must prove route/capability truth does not drift.

## Required changes

Update `tool_capability_registry.go`, focused gateway/modelruntime tests, authority matrix tests, and docs.

## Acceptance criteria

No stale “future approval flow” text remains for implemented `model.delete_file`. `model.delete_file` is approval-required/destructive and modelruntime-owned. `model.chat` and `model.generate` are not ambiguous.

## WHAT NOT TO DO

Do not remove `/forge/models*` compatibility. Do not disable streaming. Do not add a new side-channel. Do not make model output truth.
