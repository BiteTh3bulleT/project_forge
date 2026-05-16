#!/usr/bin/env bash
# Bring-up smoke test for current FORGE core.
# Boots core against an isolated data dir, probes health/meta/autonomy/
# adapters, tears it down. Exits non-zero on any failure.
#
# Not a replacement for `go test` — a complement that exercises the
# real HTTP surface to catch wiring regressions.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PORT="${FORGE_CORE_PORT:-18492}"
DATA_DIR="$(mktemp -d /tmp/forge-smoke.XXXXXX)"
WORKSPACE_DIR="$(mktemp -d /tmp/forge-smoke-ws.XXXXXX)"
LOG="$DATA_DIR/core.log"
TOKEN_FILE="$DATA_DIR/auth/api_token"
PID=""

cleanup() {
  local ec=$?
  if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
  # Kill any child go-run subprocess holding the port.
  if command -v lsof >/dev/null 2>&1; then
    local kids
    kids="$(lsof -ti tcp:"$PORT" 2>/dev/null || true)"
    [[ -n "$kids" ]] && kill $kids 2>/dev/null || true
  fi
  if [[ $ec -ne 0 ]]; then
    echo "---- smoke failed; core.log tail ----" >&2
    if [[ -f "$LOG" ]]; then
      tail -40 "$LOG" >&2 || true
    else
      echo "(core.log not created; preflight likely failed before core start)" >&2
    fi
  fi
  rm -rf "$DATA_DIR" "$WORKSPACE_DIR"
  exit $ec
}
trap cleanup EXIT

probe() {
  local path="$1"
  local want_http="${2:-200}"
  local got
  if [[ "$path" == "/health" ]]; then
    got="$(curl -s -o /tmp/forge-smoke-body -w '%{http_code}' "http://127.0.0.1:$PORT$path" || true)"
  else
    got="$(curl -s -H "Authorization: Bearer $FORGE_SMOKE_API_TOKEN" -o /tmp/forge-smoke-body -w '%{http_code}' "http://127.0.0.1:$PORT$path" || true)"
  fi
  if [[ "$got" != "$want_http" ]]; then
    echo "FAIL  $path -> http $got (expected $want_http)" >&2
    cat /tmp/forge-smoke-body >&2 || true
    echo >&2
    return 1
  fi
  echo "ok    $path -> $got"
}

echo "==> port $PORT must be free"
if ss -ltn "sport = :$PORT" 2>/dev/null | grep -q LISTEN; then
  echo "FAIL  port $PORT already in use" >&2
  exit 1
fi

echo "==> starting forge-core (data=$DATA_DIR)"
# Fail closed when VSA git tracked-state cannot be verified.
bash "$REPO_ROOT/scripts/check-vsa-files.sh" --require-tracked
cd "$REPO_ROOT/services/core"
FORGE_DATA_DIR="$DATA_DIR" \
FORGE_WORKSPACE_DIR="$WORKSPACE_DIR" \
FORGE_CORE_PORT="$PORT" \
  go run . >"$LOG" 2>&1 &
PID=$!

echo "==> waiting for /health (up to 30s)"
for i in $(seq 1 60); do
  if curl -sf "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
    echo "ok    /health after ${i} attempts"
    break
  fi
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "FAIL  core process exited before becoming healthy" >&2
    exit 1
  fi
  sleep 0.5
  if [[ "$i" == "60" ]]; then
    echo "FAIL  /health did not respond in 30s" >&2
    exit 1
  fi
done

echo "==> probing endpoints"
FORGE_SMOKE_API_TOKEN="$(tr -d '\r\n\t ' < "$TOKEN_FILE")"
probe /health 200
probe /api/meta 200
probe /api/autonomy/status 200
probe /api/telegram/status 200
probe /api/discord/status 200
probe /api/adapters 200
probe /api/jobs 200

echo "==> shutting down"
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true
PID=""

echo "==> smoke OK"
