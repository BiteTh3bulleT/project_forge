#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.forge/run"
LOG_DIR="$ROOT_DIR/.forge/logs"
CORE_PID_FILE="$RUN_DIR/core.pid"
DESKTOP_PID_FILE="$RUN_DIR/desktop.pid"
CORE_LOG="$LOG_DIR/core.log"
DESKTOP_LOG="$LOG_DIR/desktop.log"
CORE_URL="http://127.0.0.1:18492/health"
DESKTOP_PORT="5173"

mkdir -p "$RUN_DIR" "$LOG_DIR"

is_running() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null
}

start_if_needed() {
  local name="$1"
  local pid_file="$2"
  local log_file="$3"
  local cmd="$4"

  if [[ -f "$pid_file" ]]; then
    local existing
    existing="$(cat "$pid_file" 2>/dev/null || true)"
    if [[ -n "$existing" ]] && is_running "$existing"; then
      echo "$name already running (pid $existing)"
      return 0
    fi
    rm -f "$pid_file"
  fi

  echo "Starting $name..."
  (
    cd "$ROOT_DIR"
    nohup bash -lc "$cmd" >>"$log_file" 2>&1 &
    echo $! >"$pid_file"
  )

  local pid
  pid="$(cat "$pid_file")"
  if ! is_running "$pid"; then
    echo "Failed to start $name. Check $log_file"
    exit 1
  fi
  echo "$name started (pid $pid)"
}

wait_for_core() {
  local attempts=40
  local delay=0.5

  echo "Waiting for core health..."
  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$CORE_URL" >/dev/null 2>&1; then
      echo "Core is healthy."
      return 0
    fi
    sleep "$delay"
  done

  echo "Core did not become healthy in time. Check $CORE_LOG"
  exit 1
}

wait_for_desktop() {
  :
}

start_if_needed "core" "$CORE_PID_FILE" "$CORE_LOG" "npm run core"
wait_for_core
start_if_needed "desktop" "$DESKTOP_PID_FILE" "$DESKTOP_LOG" "npm run desktop"

echo "FORGE start initiated."
echo "Desktop startup runs in background and can take time on first compile."
echo "Core log:    $CORE_LOG"
echo "Desktop log: $DESKTOP_LOG"
