#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${FORGE_DOCKER_ENV_FILE:-$ROOT_DIR/.env.docker}"

cd "$ROOT_DIR"

args=()
if [[ -f "$ENV_FILE" ]]; then
  args+=(--env-file "$ENV_FILE")
fi

if [[ "${FORGE_DOCKER_PROFILES:-}" != "" ]]; then
  IFS=',' read -ra profiles <<<"$FORGE_DOCKER_PROFILES"
  for profile in "${profiles[@]}"; do
    profile="${profile//[[:space:]]/}"
    if [[ -n "$profile" ]]; then
      args+=(--profile "$profile")
    fi
  done
fi

echo "Stopping FORGE Docker stack without deleting volumes..."
docker compose "${args[@]}" down --remove-orphans

echo
echo "FORGE Docker stack stopped. Named volumes were preserved."
echo "To erase databases intentionally, run: docker compose down -v"
