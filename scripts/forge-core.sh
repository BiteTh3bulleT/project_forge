#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Authoritative bring-up path requires VSA files to be present and git-tracked.
bash "$ROOT_DIR/scripts/check-vsa-files.sh" --require-tracked

cd "$ROOT_DIR/services/core"
export FORGE_ENABLE_MODEL_RUNTIME="${FORGE_ENABLE_MODEL_RUNTIME:-true}"
exec go run .
