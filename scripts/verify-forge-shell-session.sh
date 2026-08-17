#!/usr/bin/env bash
# Verify the Forge desktop shell session launch path and reject unintended browser daemons.
# Intended to be run inside a logged-in graphical Forge operator session.

set -euo pipefail

# shellcheck disable=SC2009
has_process() {
  local pattern="$1"
  local label="$2"
  local pids
  pids="$(pgrep -f "(^|/)${pattern}([[:space:]]|$)" || true)"
  if [[ -z "$pids" ]]; then
    return 1
  fi
  echo "ok $label: $pids"
}

require_process() {
  local pattern="$1"
  local label="$2"
  if ! has_process "$pattern" "$label"; then
    echo "FAIL missing $label (pattern: $pattern)" >&2
    return 1
  fi
}

require_any_process() {
  local patterns="$1"
  local label="$2"
  local matched=""
  local pids=""
  local pattern

  for pattern in $patterns; do
    pids="$(pgrep -f "(^|/)${pattern}([[:space:]]|$)" || true)"
    if [[ -n "$pids" ]]; then
      echo "ok ${label}: $pids (pattern: ${pattern})"
      return 0
    fi
    matched="${matched}${matched:+,}${pattern}"
  done

  echo "FAIL missing ${label} (patterns: ${matched})" >&2
  return 1
}

forbid_process() {
  local pattern="$1"
  local label="$2"
  local pids
  pids="$(pgrep -f "(^|/)${pattern}([[:space:]]|$)" || true)"
  if [[ -n "$pids" ]]; then
    echo "FAIL unexpected $label running before deliberate launch: $pids" >&2
    return 1
  fi
  echo "ok no uncontrolled $label"
}

echo "==> verifying Forge shell session chain"
require_process "labwc" "labwc compositor"
require_any_process "forge-desktop-shell forge_desktop" "forge-desktop shell binary"
require_any_process "forge-shell-session forge-desktop-shell forge_desktop" "forge-shell session entrypoint"

for b in firefox firefox-esr chromium chrome google-chrome microsoft-edge webex; do
  forbid_process "$b" "$b"
done

# This is expected for Tauri surface processes.
if pgrep -f "(^|/)WebKitWebProcess([[:space:]]|$)" >/dev/null; then
  echo "ok expected WebKit child process is present"
else
  echo "warn no WebKit child found; still acceptable if shell launch is pre-UI but unusual for Tauri"
fi

echo "==> session chain appears correct"
