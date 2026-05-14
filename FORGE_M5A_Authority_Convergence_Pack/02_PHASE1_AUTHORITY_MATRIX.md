# Phase 1 Prompt — Runtime Authority Matrix

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Create a deterministic, test-covered authority matrix that lets FORGE answer:

> Who owns this route/action, what can it mutate, what approval is required, and where is the audit trail?

## Required matrix fields

Each row should contain id, surface, method, route, action, authority owner, capability id, gateway capability status, mutating flag, destructive flag, requires approval, approval mechanism, audit category/action, live authority, FORGE-K authority, host mutation, modelruntime mutation, semantic memory write, status, and notes.

## Required coverage

Include Gateway, modelruntime, OpenAI-compatible routes, Control Lane validation actions, memory surfaces, backup restore, HostBridge, FORGE-H, and System status.

## Test requirements

Tests must prove modelruntime route coverage, delete-file approval requirements, chat/generate explicit ownership, Gateway ownership of tool execution, FORGE-K authority false for live route rows unless partial validation, and host mutation false for diagnostics.

## WHAT NOT TO DO

Do not add mutation behavior. Do not create new execution routes unless necessary. Do not expose secrets/raw logs/raw prompts/raw memory.
