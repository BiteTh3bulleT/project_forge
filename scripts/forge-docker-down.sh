#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${FORGE_DOCKER_ENV_FILE:-$ROOT_DIR/.env.docker}"

cd "$ROOT_DIR"

compose_args=()
igpu_mode="${FORGE_DOCKER_IGPU:-auto}"
if [[ "$igpu_mode" != "0" && "$igpu_mode" != "false" && -d /dev/dri && -e /dev/dri/renderD128 ]]; then
  export FORGE_RENDER_GROUP_ID="${FORGE_RENDER_GROUP_ID:-$(stat -c '%g' /dev/dri/renderD128)}"
  if [[ -e /dev/dri/card1 ]]; then
    export FORGE_VIDEO_GROUP_ID="${FORGE_VIDEO_GROUP_ID:-$(stat -c '%g' /dev/dri/card1)}"
  fi
  compose_args+=(-f docker-compose.yml -f docker-compose.igpu.yml)
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

echo "Stopping FORGE Docker stack without deleting volumes..."
docker compose "${compose_args[@]}" "${args[@]}" down --remove-orphans

echo
echo "FORGE Docker stack stopped. Named volumes were preserved."
echo "To erase databases intentionally, run: docker compose down -v"
