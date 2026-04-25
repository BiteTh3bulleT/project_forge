#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/.forge/run"
CORE_PID_FILE="$RUN_DIR/core.pid"
DESKTOP_PID_FILE="$RUN_DIR/desktop.pid"

is_running() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null
}

collect_descendants() {
  local parent="$1"
  local children
  children="$(pgrep -P "$parent" || true)"
  for child in $children; do
    collect_descendants "$child"
    echo "$child"
  done
}

kill_tree() {
  local pid="$1"
  if ! is_running "$pid"; then
    return 0
  fi

  local descendants
  descendants="$(collect_descendants "$pid")"

  for d in $descendants; do
    kill -TERM "$d" 2>/dev/null || true
  done
  kill -TERM "$pid" 2>/dev/null || true

  sleep 1

  for d in $descendants; do
    if is_running "$d"; then
      kill -KILL "$d" 2>/dev/null || true
    fi
  done
  if is_running "$pid"; then
    kill -KILL "$pid" 2>/dev/null || true
  fi
}

stop_from_pid_file() {
  local name="$1"
  local pid_file="$2"

  if [[ ! -f "$pid_file" ]]; then
    echo "$name not tracked (no pid file)."
    return 0
  fi

  local pid
  pid="$(cat "$pid_file" 2>/dev/null || true)"

  if [[ -z "$pid" ]]; then
    rm -f "$pid_file"
    echo "$name pid file was empty; cleaned."
    return 0
  fi

  if is_running "$pid"; then
    echo "Stopping $name (pid $pid)..."
    kill_tree "$pid"
    echo "$name stopped."
  else
    echo "$name already stopped (stale pid $pid)."
  fi

  rm -f "$pid_file"
}

# Fallback: kill listeners on known FORGE dev ports in case pid tracking drifted.
kill_port_listener() {
  local port="$1"
  local pids
  pids="$(ss -ltnp 2>/dev/null | awk -v p=":$port" '$4 ~ p {print $NF}' | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' | sort -u)"
  for pid in $pids; do
    if is_running "$pid"; then
      echo "Stopping listener on :$port (pid $pid)..."
      kill_tree "$pid"
    fi
  done
}

# Stop desktop first, then core.
stop_from_pid_file "desktop" "$DESKTOP_PID_FILE"
stop_from_pid_file "core" "$CORE_PID_FILE"
kill_port_listener 1420
kill_port_listener 18492
pkill -f "target/debug/forge_desktop" 2>/dev/null || true

echo "FORGE stopped."
