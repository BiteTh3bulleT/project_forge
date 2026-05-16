#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_PORT="${FORGE_CORE_PORT:-18492}"
CORE_URL="${VITE_FORGE_API_URL:-http://127.0.0.1:${CORE_PORT}}"
TOKEN_FILE="$ROOT_DIR/.forge/docker-api-token"

cd "$ROOT_DIR"

if [[ -z "${FORGE_API_TOKEN:-}" ]]; then
  mkdir -p "$(dirname "$TOKEN_FILE")"
  if [[ ! -s "$TOKEN_FILE" ]]; then
    if command -v openssl >/dev/null 2>&1; then
      openssl rand -hex 32 >"$TOKEN_FILE"
    else
      python3 -c 'import secrets; print(secrets.token_hex(32))' >"$TOKEN_FILE"
    fi
    chmod 600 "$TOKEN_FILE" 2>/dev/null || true
  fi
  export FORGE_API_TOKEN
  FORGE_API_TOKEN="$(tr -d '\r\n\t ' < "$TOKEN_FILE")"
fi

echo "Starting Docker-backed FORGE services first..."
FORGE_CORE_PORT="$CORE_PORT" "$ROOT_DIR/scripts/forge-docker-up.sh" postgres redis qdrant core

echo
echo "Ensuring browser-served desktop-web is not holding the Tauri dev port..."
docker compose stop desktop-web >/dev/null 2>&1 || true

echo
echo "Launching native Tauri desktop shell against $CORE_URL"
echo "The desktop shell runs on the host; Docker provides core and data services."

VITE_FORGE_API_URL="$CORE_URL" npm run desktop
