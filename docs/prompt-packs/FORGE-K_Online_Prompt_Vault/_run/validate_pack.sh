#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
required=(
  "README.md"
  "START_HERE.md"
  "AGENTS.md"
  "SKILLS.md"
  "PROMPTS.md"
  "AI_CONTEXT.md"
  "skills"
  "skill_breakdowns"
  "process"
  "architecture"
  "prompts/MASTER_PROMPT.md"
  "ai_context/CURRENT_TRUTH.md"
  ".cursor/rules/forge-k-online.mdc"
)
for item in "${required[@]}"; do
  if [[ ! -e "$ROOT/$item" ]]; then
    echo "missing: $item" >&2
    exit 1
  fi
done
echo "FORGE-K prompt vault structure looks sane. Shockingly civilized."
