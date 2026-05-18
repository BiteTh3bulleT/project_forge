# Emergency Rollback Prompt

Use when a FORGE-K online phase breaks live behavior.

## Task

Identify the smallest rollback that restores previous live behavior without deleting evidence.

## Required actions

1. Identify changed files.
2. Identify feature flags/config toggles.
3. Identify data written by the failed phase.
4. Disable new path if possible.
5. Re-run tests proving old behavior.
6. Produce incident report.

## WHAT NOT TO DO

- Do not delete audit/provenance evidence.
- Do not perform broad rewrites.
- Do not continue feature work during rollback.
