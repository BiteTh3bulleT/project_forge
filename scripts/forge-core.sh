#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

configure_local_modelruntime_defaults() {
  if [[ "${FORGE_DISABLE_OLLAMA_AUTODETECT:-false}" == "true" ]]; then
    return
  fi

  # Respect explicit runtime configuration. This helper is only for the local
  # dev bring-up path where Ollama is already running but FORGE has no backend.
  if [[ -n "${FORGE_ENABLE_MODEL_RUNTIME:-}" ||
    -n "${FORGE_MODEL_OPENAI_COMPAT_ENDPOINT:-}" ||
    -n "${FORGE_MODEL_VLLM_ENDPOINT:-}" ||
    -n "${FORGE_LLAMA_CPP_ENDPOINT:-}" ||
    -n "${FORGE_LLAMA_CPP_BINARY_PATH:-}" ]]; then
    return
  fi

  local ollama_url="${OLLAMA_BASE_URL:-http://127.0.0.1:11434}"
  ollama_url="${ollama_url%/}"

  local models_json
  if ! models_json="$(curl -fsS --max-time 1.5 "$ollama_url/v1/models" 2>/dev/null)"; then
    return
  fi

  export FORGE_ENABLE_MODEL_RUNTIME="${FORGE_ENABLE_MODEL_RUNTIME:-true}"
  export FORGE_MODEL_OPENAI_COMPAT_ENDPOINT="${FORGE_MODEL_OPENAI_COMPAT_ENDPOINT:-$ollama_url}"
  export FORGE_MODEL_DEFAULT_BACKEND="${FORGE_MODEL_DEFAULT_BACKEND:-openai_compat}"
  export FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD="${FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD:-true}"
  export FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD="${FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD:-false}"
  export FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE="${FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE:-false}"
  export OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-$ollama_url}"

  if [[ -z "${FORGE_MODEL_DEFAULT_ID:-}" && -z "${OLLAMA_MODEL:-}" ]]; then
    local selected_model
    selected_model="$(
      printf '%s' "$models_json" | node -e '
const fs = require("fs");
const raw = fs.readFileSync(0, "utf8");
const payload = JSON.parse(raw);
const ids = (payload.data || []).map((model) => model && model.id).filter(Boolean);
const preferred = [
  "phi3:3.8b",
  "llama3.1:8b",
  "llama3.2:3b",
  "mistral:latest",
  "qwen2.5-coder:7b",
  "qwen2.5:14b",
  "qwen3.6:latest",
  "qwen3-coder:480b-cloud"
];
const chosen = preferred.find((id) => ids.includes(id)) || ids.find((id) => !id.includes("embed")) || "";
process.stdout.write(chosen);
' 2>/dev/null || true
    )"
    if [[ -n "$selected_model" ]]; then
      export FORGE_MODEL_DEFAULT_ID="$selected_model"
      export OLLAMA_MODEL="$selected_model"
    fi
  else
    export FORGE_MODEL_DEFAULT_ID="${FORGE_MODEL_DEFAULT_ID:-${OLLAMA_MODEL:-}}"
    export OLLAMA_MODEL="${OLLAMA_MODEL:-${FORGE_MODEL_DEFAULT_ID:-}}"
  fi

  printf 'FORGE modelruntime auto-enabled via local Ollama at %s' "$ollama_url"
  if [[ -n "${FORGE_MODEL_DEFAULT_ID:-}" ]]; then
    printf ' (default model: %s)' "$FORGE_MODEL_DEFAULT_ID"
  fi
  printf '\n'
}

# Authoritative bring-up path requires VSA files to be present and git-tracked.
bash "$ROOT_DIR/scripts/check-vsa-files.sh" --require-tracked

configure_local_modelruntime_defaults

cd "$ROOT_DIR/services/core"
export FORGE_ENABLE_MODEL_RUNTIME="${FORGE_ENABLE_MODEL_RUNTIME:-true}"
exec go run .
