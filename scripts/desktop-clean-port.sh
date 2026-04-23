#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-5173}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

listener_pid="$(lsof -tiTCP:${PORT} -sTCP:LISTEN 2>/dev/null | head -n1 || true)"
if [[ -z "${listener_pid}" ]]; then
  exit 0
fi

listener_cmd="$(ps -o command= -p "${listener_pid}" 2>/dev/null || true)"
repo_vite_path="${REPO_ROOT}/node_modules/.bin/vite"

if [[ "${listener_cmd}" == *"${repo_vite_path}"* ]]; then
  echo "[forge desktop] Found stale local Vite process on :${PORT} (pid ${listener_pid}), stopping it."
  kill "${listener_pid}" || true
  sleep 1
  if lsof -tiTCP:${PORT} -sTCP:LISTEN >/dev/null 2>&1; then
    kill -9 "${listener_pid}" || true
  fi
  exit 0
fi

cat <<EOF
[forge desktop] Port ${PORT} is already in use by a non-FORGE process:
  pid: ${listener_pid}
  cmd: ${listener_cmd}

Stop that process or change your local desktop dev port configuration.
EOF
exit 1
