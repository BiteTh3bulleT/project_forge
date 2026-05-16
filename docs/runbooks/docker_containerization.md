# Docker Containerization

FORGE can run in Docker with the Go core service, managed data services, a persisted SQLite data volume, and an optional web-served desktop build.

## Current Data Model

The current primary database is SQLite:

- database file: `${FORGE_DATA_DIR}/forge.sqlite`
- container path: `/data/forge.sqlite`
- Docker volume: `forge-data`
- storage backend selector: `FORGE_STORE_BACKEND=sqlite` by default

The current retrieval, memory, VSA, settings, audit-adjacent records, jobs, approvals, and embeddings metadata are stored through the core SQLite path.

Postgres, Redis, and Qdrant are included as managed containers for the next storage phase:

- Postgres: future durable relational store for canonical memory, jobs, approvals, audit-adjacent records, and settings.
- Redis: future ephemeral cache, queue, job progress stream, locks, and pub/sub surface. Redis must not become truth.
- Qdrant: future vector index for embedding retrieval acceleration. Qdrant vectors must not become truth or admissibility.

In this containerization pass, the Go core still reads and writes SQLite. The new service containers are up and health-checked, but not yet used by the live memory code.

Phase 13A adds application-side backend config and capability contracts. `FORGE_STORE_BACKEND=postgres` is parsed for future migration scaffolding, but SQLite remains the live default and no live data is dual-written or read-switched in this phase.

Phase 13H adds Redis boundary flags for future ephemeral coordination. `FORGE_REDIS_ENABLED=false` is the default; enabling it does not switch live job queues, caches, retrieval, gateway, modelruntime, memory, or public API behavior. Redis remains disposable and non-canonical.

The Docker core enables the governed model runtime surface by default and points Ollama discovery at host Ollama through `http://host.docker.internal:11434`. If Ollama is not running or has no local models, FORGE remains healthy and the model runtime reports degraded backend health instead of disappearing from the API surface.

The Compose development stack sets `FORGE_CORE_BIND_HOST=0.0.0.0` and `FORGE_ALLOW_WILDCARD_BIND=true` inside the core container so the published container port can reach the process. That wildcard bind is still auth-gated: `/api/*`, `/forge/*`, and enabled `/v1/*` require `Authorization: Bearer <token>`, while `/health` remains public. Direct Go and NixOS service runs default to `127.0.0.1`, and wildcard binds fail closed unless explicitly opted in and an API token is available.

The standalone `services/core/Dockerfile` no longer defaults to wildcard bind. Compose is the explicit development-network profile that opts into wildcard binding, local CORS for browser development, and token-backed API access.

Compose binds host-published ports to `127.0.0.1` by default through `FORGE_DOCKER_BIND_HOST`. This keeps the core, browser-served desktop, managed data services, and optional provider sidecars local to the host even though containers still communicate over the Compose network. Set `FORGE_DOCKER_BIND_HOST=0.0.0.0` only when you intentionally expose these development services through a firewall or private lab network.

Optional providers such as Ollama and Hugging Face TEI can be started as sidecars, but they do not replace FORGE authority paths.

On hosts with `/dev/dri/renderD128`, `npm run docker:start` automatically layers `docker-compose.igpu.yml` into the Compose invocation. That passes the Intel iGPU render devices into the core container, adds the host render/video group IDs, enables Intel telemetry, and uses the container's `intel_gpu_top` binary for utilization sampling. Intel PMU access requires the override to run the core container as root and add telemetry-only container privileges (`CAP_PERFMON`, `CAP_SYS_ADMIN`, `seccomp=unconfined`, and host user namespace mode). Set `FORGE_DOCKER_IGPU=0` to disable this override or `FORGE_DOCKER_IGPU=1` to require it.

## Start Core And Databases

```bash
npm run docker:start
```

`npm run docker:start` starts Postgres, Redis, Qdrant, and core, and preserves existing named volumes. It does not start the browser-served `desktop-web` container. The normal operator shell is native Tauri through `npm run docker:desktop`.

If the native dev core or desktop is already using the default ports, choose alternate published ports:

```bash
FORGE_CORE_PORT=18493 npm run docker:start
```

Open:

- Core: `http://127.0.0.1:18492`
- Web UI build: `http://127.0.0.1:1420`
- Model runtime health: `http://127.0.0.1:18492/forge/model-runtime/health`
- Postgres: `127.0.0.1:5432`
- Redis: `127.0.0.1:6379`
- Qdrant HTTP: `http://127.0.0.1:6333`

## Development Web Surface

The development-only browser surface remains available when needed:

```bash
npm run docker:web
```

That starts `desktop-web`, exposes `http://127.0.0.1:1420/#/dashboard`, and best-effort opens it unless `FORGE_DOCKER_OPEN=0` is set.

The Tauri desktop shell still runs through the native desktop workflow. Docker cannot launch the native Tauri window by itself because that shell depends on the host display session, window manager, WebKit/Tauri runtime integration, and local desktop permissions. The `desktop-web` container serves the same Vite app as a browser surface for development and containerized inspection.

## Native Desktop Shell With Docker Backend

The native Tauri desktop shell is not run inside Docker by default. It depends on the host display session, window manager, WebKit/Tauri runtime integration, and local desktop permissions. Docker remains the right boundary for core services, databases, and browser-served inspection surfaces.

Use this helper when you want the desktop shell on the host with Docker-backed data services:

```bash
npm run docker:desktop
```

The helper starts Postgres, Redis, Qdrant, and the Go core through Docker, then launches the native Tauri shell with `VITE_FORGE_API_URL` pointed at the Docker-published core URL.

When `FORGE_API_TOKEN` is not already set, the native desktop helper creates an ignored local token at `.forge/docker-api-token`, passes it to the Docker core, and exposes the same process environment to Tauri. The desktop reads that token through its native command path and sends a backend-verified bearer header; it does not rely on `sessionStorage` as API authority.

If `desktop-web` was already started by `npm run docker:web`, the helper stops only that browser-served container before launching native Tauri. This frees the shared Vite/Tauri development port `1420` without stopping the Docker-backed core or data services.

If the default core port is busy, choose an alternate core port:

```bash
FORGE_CORE_PORT=18493 npm run docker:desktop
```

This does not start the `desktop-web` container. Use `npm run docker:web` when you want the browser-served desktop build for development.

## Managed Data Services

The default Compose stack starts:

- `postgres`
- `redis`
- `qdrant`
- `core`

`desktop-web` is intentionally opt-in through `npm run docker:web`.

Useful probes:

```bash
docker compose exec postgres pg_isready -U forge -d forge
docker compose exec redis redis-cli ping
curl -fsS http://127.0.0.1:6333/readyz
```

Redis ephemeral boundary flags:

```bash
FORGE_REDIS_ENABLED=false
FORGE_REDIS_ADDR=redis:6379
FORGE_REDIS_KEY_PREFIX=forge
FORGE_REDIS_TIMEOUT_MS=1000
```

Optional Redis integration tests use `FORGE_REDIS_TEST_ADDR=127.0.0.1:6379`. Default repository tests do not require Redis.

If default ports are busy, override the published ports:

```bash
FORGE_POSTGRES_PORT=15432 \
FORGE_REDIS_PORT=16379 \
FORGE_QDRANT_HTTP_PORT=16333 \
FORGE_QDRANT_GRPC_PORT=16334 \
docker compose up -d postgres redis qdrant
```

To verify the host-side published-port bindings before startup:

```bash
docker compose config | grep -E 'host_ip: 127\.0\.0\.1|published: "(18492|1420|5432|6379|6333|6334|11434|8082)"'
```

## Stop Without Deleting Databases

```bash
npm run docker:stop
```

This runs `docker compose down --remove-orphans` without `-v`, so the named volumes are preserved:

- `forge-data`
- `forge-postgres`
- `forge-redis`
- `forge-qdrant`
- `forge-models`
- `forge-workspace`

Use this for normal shutdowns.

## Start Optional Ollama Sidecar

```bash
docker compose --profile ollama up --build
```

Set these values in `.env.docker` if you want the core to talk to the Ollama service through its OpenAI-compatible endpoint:

```dotenv
FORGE_ENABLE_MODEL_RUNTIME=true
FORGE_MODEL_OPENAI_COMPAT_ENDPOINT=http://ollama:11434
FORGE_MODEL_DEFAULT_BACKEND=openai_compat
```

Model pulls are still operator-owned. Starting the container does not automatically pull or load a model.

## Start Optional TEI Sidecar

```bash
docker compose --profile tei up --build
```

Set these values in `.env.docker` if you want the core to use TEI for embeddings:

```dotenv
FORGE_EMBEDDING_PROVIDER=tei
FORGE_EMBEDDING_TEI_ENDPOINT=http://tei:80
```

TEI model selection and image sizing are operator-owned. The sidecar does not make embeddings truth and does not bypass the existing retrieval/memory authority model.

## Inspect The SQLite Database

The SQLite database lives in the `forge-data` named volume. To inspect the files:

```bash
docker compose exec core ls -la /data
```

To reset local container state:

```bash
docker compose down -v
```

This deletes the named volumes, including `/data/forge.sqlite`, Postgres data, Redis data, Qdrant indexes, model artifacts, and the workspace volume.

Do not use `-v` unless you explicitly want to erase the databases.

## Validation

Useful checks:

```bash
docker compose config
docker compose build core
docker compose up -d core
docker compose exec core wget -qO- http://127.0.0.1:18492/health
docker compose exec postgres pg_isready -U forge -d forge
docker compose exec redis redis-cli ping
curl -fsS http://127.0.0.1:6333/readyz
```

Intel iGPU telemetry checks:

```bash
npm run docker:start core
docker compose -f docker-compose.yml -f docker-compose.igpu.yml exec core ls -l /dev/dri
curl -fsS http://127.0.0.1:18492/health
```

The root repository validation remains:

```bash
npm run build:core
npm run lint
npm test
```

## Boundaries

- Docker does not change live FORGE-K authority.
- SQLite remains the current database.
- Postgres is infrastructure-ready, not yet the live application store.
- Redis is infrastructure-ready and must remain disabled-by-default cache/queue/stream/lock metadata, not truth.
- Qdrant is infrastructure-ready and must remain vector retrieval acceleration, not truth.
- Optional provider containers do not execute unless their profiles are selected.
- Optional model/embedding providers do not create truth.
- No public API or route behavior is changed by containerization.
