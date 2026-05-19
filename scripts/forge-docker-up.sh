#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${FORGE_DOCKER_ENV_FILE:-$ROOT_DIR/.env.docker}"
TOKEN_FILE="$ROOT_DIR/.forge/docker-api-token"

cd "$ROOT_DIR"

if [[ -z "${FORGE_API_TOKEN:-}" ]]; then
  mkdir -p "$(dirname "$TOKEN_FILE")"
  if [[ ! -s "$TOKEN_FILE" ]]; then
    od -An -N32 -tx1 /dev/urandom | tr -d ' \n' >"$TOKEN_FILE"
    chmod 600 "$TOKEN_FILE" || true
  fi
  export FORGE_API_TOKEN="$(tr -d '\r\n' <"$TOKEN_FILE")"
fi

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

env_set() {
  local name="$1"
  [[ -n "${!name:-}" ]]
}

env_file_sets() {
  local name="$1"
  [[ -f "$ENV_FILE" ]] && grep -Eq "^[[:space:]]*(export[[:space:]]+)?${name}=" "$ENV_FILE"
}

default_env() {
  local name="$1"
  local value="$2"
  if ! env_set "$name"; then
    export "$name=$value"
  fi
}

docker_ollama_probe_url() {
  local container_url="${OLLAMA_BASE_URL:-http://host.docker.internal:11434}"
  local probe_url="${FORGE_DOCKER_OLLAMA_PROBE_URL:-$container_url}"
  if [[ "$probe_url" == *"host.docker.internal"* ]]; then
    probe_url="http://127.0.0.1:11434"
  fi
  printf '%s' "${probe_url%/}"
}

enable_docker_ollama_defaults() {
  if [[ "${FORGE_DISABLE_OLLAMA_AUTODETECT:-false}" == "true" ]]; then
    return
  fi

  default_env OLLAMA_BASE_URL "http://host.docker.internal:11434"
  default_env FORGE_ENABLE_MODEL_RUNTIME "true"
  default_env FORGE_MODEL_DEFAULT_BACKEND "ollama_compat"

  if env_set FORGE_MODEL_DEFAULT_ID && ! env_set OLLAMA_MODEL; then
    export OLLAMA_MODEL="$FORGE_MODEL_DEFAULT_ID"
    return
  fi
  if env_set OLLAMA_MODEL && ! env_set FORGE_MODEL_DEFAULT_ID; then
    export FORGE_MODEL_DEFAULT_ID="$OLLAMA_MODEL"
    return
  fi
  if env_set FORGE_MODEL_DEFAULT_ID || env_set OLLAMA_MODEL; then
    return
  fi

  local probe_url
  probe_url="$(docker_ollama_probe_url)"
  local models_json
  if ! models_json="$(curl -fsS --max-time 2 "$probe_url/v1/models" 2>/dev/null)"; then
    return
  fi

  local selected_model
  selected_model="$(
    printf '%s' "$models_json" | node -e '
const fs = require("fs");
const raw = fs.readFileSync(0, "utf8");
const payload = JSON.parse(raw);
const ids = (payload.data || []).map((model) => model && model.id).filter(Boolean);
const preferred = [
  "phi4-mini:latest",
  "llama3.2:latest",
  "llama3.2:3b",
  "phi3:3.8b",
  "llama3.1:8b",
  "qwen2.5-coder:7b",
  "mistral:latest",
  "qwen2.5:14b",
  "qwen3.6:latest"
];
const chosen = preferred.find((id) => ids.includes(id)) || ids.find((id) => !id.includes("embed") && !id.includes("cloud")) || "";
process.stdout.write(chosen);
' 2>/dev/null || true
  )"
  if [[ -n "$selected_model" ]]; then
    export FORGE_MODEL_DEFAULT_ID="$selected_model"
    export OLLAMA_MODEL="$selected_model"
    printf 'FORGE Docker model default selected from host Ollama: %s\n' "$selected_model"
  fi
}

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
  services=(postgres redis qdrant core)
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

host_port_busy() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -H -ltn "sport = :${port}" 2>/dev/null | grep -q .
    return
  fi
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    return
  fi
  return 1
}

first_free_host_port() {
  local start="$1"
  local port="$start"
  while host_port_busy "$port"; do
    port=$((port + 1))
  done
  printf '%s' "$port"
}

docker_service_published_port() {
  local service="$1"
  local target_port="$2"
  local published
  published="$(docker compose "${compose_args[@]}" "${args[@]}" port "$service" "$target_port" 2>/dev/null || true)"
  if [[ -z "$published" ]]; then
    return 1
  fi
  printf '%s' "${published##*:}"
}

apply_desktop_port_fallbacks() {
  if ! has_service postgres; then
    return
  fi
  if env_set FORGE_POSTGRES_PORT || env_file_sets FORGE_POSTGRES_PORT; then
    return
  fi
  local current_postgres_port
  current_postgres_port="$(docker_service_published_port postgres 5432 || true)"
  if [[ -n "$current_postgres_port" ]]; then
    export FORGE_POSTGRES_PORT="$current_postgres_port"
    if [[ "$current_postgres_port" != "5432" ]]; then
      printf 'Keeping running FORGE Postgres published on 127.0.0.1:%s.\n' "$current_postgres_port"
    fi
    return
  fi
  if ! host_port_busy 5432; then
    return
  fi
  export FORGE_POSTGRES_PORT
  FORGE_POSTGRES_PORT="$(first_free_host_port 15432)"
  printf 'Host port 5432 is already in use; publishing FORGE Postgres on 127.0.0.1:%s instead.\n' "$FORGE_POSTGRES_PORT"
  printf 'Internal Docker services still use postgres:5432.\n'
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

enable_docker_ollama_defaults
apply_desktop_port_fallbacks
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
