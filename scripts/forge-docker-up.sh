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

has_service() {
  local wanted="$1"
  local service
  for service in "${services[@]}"; do
    if [[ "$service" == "$wanted" ]]; then
      return 0
    fi
  done
  return 1
}

open_url_best_effort() {
  local url="$1"
  if [[ "${FORGE_DOCKER_OPEN:-1}" == "0" || "${FORGE_DOCKER_OPEN:-1}" == "false" ]]; then
    echo "Auto-open disabled. Open $url manually."
    return 0
  fi

  if command -v xdg-open >/dev/null 2>&1; then
    nohup xdg-open "$url" >/dev/null 2>&1 &
    echo "Opening Docker desktop web surface: $url"
    return 0
  fi
  if command -v gio >/dev/null 2>&1; then
    nohup gio open "$url" >/dev/null 2>&1 &
    echo "Opening Docker desktop web surface: $url"
    return 0
  fi
  if command -v open >/dev/null 2>&1; then
    open "$url" >/dev/null 2>&1 &
    echo "Opening Docker desktop web surface: $url"
    return 0
  fi

  echo "Docker desktop web surface is available at $url"
}

echo "Starting FORGE Docker stack without deleting volumes..."
echo "Env file: $([[ -f "$ENV_FILE" ]] && echo "$ENV_FILE" || echo "(none)")"
echo "Services: ${services[*]}"
echo "Intel iGPU telemetry: $([[ "$igpu_enabled" == "true" ]] && echo "enabled via docker-compose.igpu.yml" || echo "not enabled")"

docker compose "${compose_args[@]}" "${args[@]}" up -d --build "${services[@]}"

echo
docker compose "${compose_args[@]}" "${args[@]}" ps
echo
echo "FORGE Docker stack started."
if has_service desktop-web; then
  desktop_port="${FORGE_DESKTOP_PORT:-1420}"
  open_url_best_effort "http://127.0.0.1:${desktop_port}/#/dashboard"
fi
echo "Use './scripts/forge-docker-down.sh' or 'npm run docker:stop' to stop containers without deleting databases."
