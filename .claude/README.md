# Claude Code Project Configuration

This project ships a shared Claude Code configuration tuned for speed and safe autonomy.

## Included

- `settings.json`: project-level permissions, effort level, and hook registration
- `hooks/session-start.mjs`: injects concise repo/session context at startup
- `hooks/pretool-bash-guard.mjs`: denies high-risk destructive shell commands
- `rules/*.md`: path-scoped rules that load only when matching files are touched
- `skills/forge-verify/SKILL.md`: user-invocable verification workflow (`/forge-verify`)

## Local Overrides

Use `.claude/settings.local.json` for machine-specific preferences. It is gitignored.

## Quick Checks In Claude Code

1. Run `/hooks` and confirm both project hooks are visible.
2. Run `/permissions` and verify project allow/deny rules loaded.
3. Type `/forge-verify` to run the verification workflow on demand.
