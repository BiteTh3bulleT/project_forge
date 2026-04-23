#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
require_tracked=0

usage() {
  cat <<'EOF'
usage: check-vsa-files.sh [--require-tracked]

Verifies required VSA source files exist for FORGE core build/boot.
With --require-tracked, also requires those files to be git-tracked in this checkout.

Exit codes:
  0   success
  2   invalid arguments
  42  required VSA file missing
  43  required VSA file present but untracked
  44  --require-tracked requested but git is unavailable
  45  --require-tracked requested outside a git work tree
EOF
}

for arg in "$@"; do
  case "$arg" in
    --require-tracked)
      require_tracked=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $arg" >&2
      usage >&2
      exit 2
      ;;
  esac
done

required=(
  "services/core/internal/memory/vsa_engine.go"
  "services/core/internal/memory/vsa_indexer.go"
  "services/core/internal/memory/vsa_signals.go"
)

missing=()
for rel in "${required[@]}"; do
  if [[ ! -f "$REPO_ROOT/$rel" ]]; then
    missing+=("$rel")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "FORGE bring-up preflight failed: VSA source files are missing." >&2
  echo "" >&2
  echo "Missing files:" >&2
  for rel in "${missing[@]}"; do
    echo "  - $rel" >&2
  done
  echo "" >&2
  echo "Action:" >&2
  echo "  1) Ensure these files are present in your branch/checkout." >&2
  echo "  2) Re-run the command after syncing the missing files." >&2
  echo "" >&2
  echo "Why this check exists:" >&2
  echo "  The core memory service depends on these VSA implementations at compile time." >&2
  echo "  Without them, core build/boot will fail with undefined symbol errors." >&2
  exit 42
fi

if [[ "$require_tracked" -eq 1 ]]; then
  if ! command -v git >/dev/null 2>&1; then
    echo "FORGE bring-up preflight failed: cannot verify tracked-state for VSA files." >&2
    echo "" >&2
    echo "Reason:" >&2
    echo "  --require-tracked was requested, but git is not installed." >&2
    echo "" >&2
    echo "Action:" >&2
    echo "  1) Install git." >&2
    echo "  2) Re-run preflight from a git checkout of this repository." >&2
    exit 44
  fi

  if ! git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "FORGE bring-up preflight failed: cannot verify tracked-state for VSA files." >&2
    echo "" >&2
    echo "Reason:" >&2
    echo "  --require-tracked was requested outside a git work tree." >&2
    echo "" >&2
    echo "Action:" >&2
    echo "  1) Run this command from a git checkout of project_forge." >&2
    echo "  2) Re-run once tracked-state can be verified." >&2
    exit 45
  fi

  untracked=()
  for rel in "${required[@]}"; do
    if ! git -C "$REPO_ROOT" ls-files --error-unmatch "$rel" >/dev/null 2>&1; then
      untracked+=("$rel")
    fi
  done
  if [[ ${#untracked[@]} -gt 0 ]]; then
    echo "FORGE bring-up preflight failed: VSA source files are present but not git-tracked in this checkout." >&2
    echo "" >&2
    echo "Files not git-tracked:" >&2
    for rel in "${untracked[@]}"; do
      echo "  - $rel" >&2
    done
    echo "" >&2
    echo "Action:" >&2
    echo "  1) Add these files to git tracked-state in this checkout (for example: git add <file>)." >&2
    echo "  2) Re-run preflight after tracked-state is visible to git ls-files." >&2
    echo "" >&2
    echo "Verification command:" >&2
    echo "  git ls-files services/core/internal/memory/vsa_*.go" >&2
    echo "" >&2
    echo "Why this check exists:" >&2
    echo "  Fresh clones only receive git-tracked files." >&2
    echo "  This check validates tracked-state in the checkout, not commit history depth." >&2
    echo "  Untracked VSA sources make local bring-up conditional and non-reproducible." >&2
    exit 43
  fi
fi
