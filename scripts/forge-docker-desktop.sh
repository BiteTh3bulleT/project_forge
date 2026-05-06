#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_PORT="${FORGE_CORE_PORT:-18492}"
CORE_URL="${VITE_FORGE_API_URL:-http://127.0.0.1:${CORE_PORT}}"

cd "$ROOT_DIR"

echo "Starting Docker-backed FORGE services first..."
FORGE_CORE_PORT="$CORE_PORT" "$ROOT_DIR/scripts/forge-docker-up.sh" postgres redis qdrant core

echo
echo "Ensuring browser-served desktop-web is not holding the Tauri dev port..."
docker compose stop desktop-web >/dev/null 2>&1 || true

echo
echo "Launching native Tauri desktop shell against $CORE_URL"
echo "The desktop shell runs on the host; Docker provides core and data services."

VITE_FORGE_API_URL="$CORE_URL" npm run desktop
