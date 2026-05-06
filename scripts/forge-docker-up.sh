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

services=("$@")
if [[ ${#services[@]} -eq 0 ]]; then
  services=(postgres redis qdrant core desktop-web)
fi

echo "Starting FORGE Docker stack without deleting volumes..."
echo "Env file: $([[ -f "$ENV_FILE" ]] && echo "$ENV_FILE" || echo "(none)")"
echo "Services: ${services[*]}"

docker compose "${args[@]}" up -d --build "${services[@]}"

echo
docker compose "${args[@]}" ps
echo
echo "FORGE Docker stack started."
echo "Use './scripts/forge-docker-down.sh' or 'npm run docker:stop' to stop containers without deleting databases."
