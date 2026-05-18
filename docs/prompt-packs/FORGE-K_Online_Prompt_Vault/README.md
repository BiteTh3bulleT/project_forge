# FORGE-K Online Prompt Vault

Status: **repo-ready prompt pack / Obsidian-ready Markdown vault**  
Generated: **2026-05-17**

This vault turns the FORGE-K online cutover plan into a complete agent execution pack for Codex, Cursor, Claude Code, and reviewer agents.

## What this is

This is the full folder structure we use for serious AI/Codex/Cursor execution packs:

- `skills/` — reusable agent skills and operating rules
- `skill_breakdowns/` — detailed checklists for each skill
- `process/` — execution loops, validation plans, refactor control, rollback control
- `architecture/` — system doctrine, authority boundaries, NixOS substrate, live cutover design
- `prompts/` — copy/paste phase prompts and master YAML prompt pack
- `ai_context/` — current project truth, repo map, module index, status, glossary
- `docs/adr/` — architecture decision templates and seed ADRs
- `docs/testing/` — definition of done and validation gates
- `.cursor/rules/` — Cursor rules for agent behavior
- `_run/` — unpack, merge, validate, and usage instructions

## Core doctrine

**NixOS owns the host. Gateway owns tools. Modelruntime owns drivers. FORGE-K owns semantic truth flow. The operator owns dangerous authority.**

FORGE-K must become live one authority seam at a time. Do not directly import simulator services into live daemon authority. Do not create a second live authority path. Do not let model output become truth.

## Start here

1. Read `START_HERE.md`.
2. Read `AGENTS.md`.
3. Read `ai_context/CURRENT_TRUTH.md`.
4. Read `process/three_pass_execution.md`.
5. Use `prompts/phase_00_repo_orientation.md` before changing code.
6. Run exactly one phase at a time.
7. Update `docs/status/PHASE_STATUS_TEMPLATE.md` after each phase.

## Expected use

Copy this vault into the repo as a planning/execution pack, or keep it as an Obsidian vault and copy prompts into Codex/Cursor as needed.

Do not dump every prompt into an agent at once. That is how you create a very confident Roomba with root access.
