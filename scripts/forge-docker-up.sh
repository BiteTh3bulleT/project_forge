#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${FORGE_DOCKER_ENV_FILE:-$ROOT_DIR/.env.docker}"

cd "$ROOT_DIR"

compose_args=()
igpu_enabled=false
igpu_mode="${FORGE_DOCKER_IGPU:-auto}"
if [[ "$igpu_mode" != "0" && "$igpu_mode" != "false" ]]; then
  if [[ -d /dev/dri && -e /dev/dri/renderD128 ]]; then
    export FORGE_RENDER_GROUP_ID="${FORGE_RENDER_GROUP_ID:-$(stat -c '%g' /dev/dri/renderD128)}"
    if [[ -e /dev/dri/card1 ]]; then
      export FORGE_VIDEO_GROUP_ID="${FORGE_VIDEO_GROUP_ID:-$(stat -c '%g' /dev/dri/card1)}"
    fi
    compose_args+=(-f docker-compose.yml -f docker-compose.igpu.yml)
    igpu_enabled=true
  elif [[ "$igpu_mode" == "1" || "$igpu_mode" == "true" ]]; then
    echo "FORGE_DOCKER_IGPU requested, but /dev/dri/renderD128 is not present." >&2
    exit 1
  fi
fi

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
echo "Intel iGPU telemetry: $([[ "$igpu_enabled" == "true" ]] && echo "enabled via docker-compose.igpu.yml" || echo "not enabled")"

docker compose "${compose_args[@]}" "${args[@]}" up -d --build "${services[@]}"

echo
docker compose "${compose_args[@]}" "${args[@]}" ps
echo
echo "FORGE Docker stack started."
echo "Use './scripts/forge-docker-down.sh' or 'npm run docker:stop' to stop containers without deleting databases."
