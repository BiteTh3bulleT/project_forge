# Docker Containerization

FORGE can run in Docker with the Go core service, a persisted SQLite data volume, and an optional web-served desktop build.

## Current Data Model

The current primary database is SQLite:

- database file: `${FORGE_DATA_DIR}/forge.sqlite`
- container path: `/data/forge.sqlite`
- Docker volume: `forge-data`

The current retrieval, memory, VSA, settings, audit-adjacent records, jobs, approvals, and embeddings metadata are stored through the core SQLite path. There is no active Postgres, Redis, Qdrant, Chroma, Milvus, or Weaviate dependency in the current repository runtime.

Optional providers such as Ollama and Hugging Face TEI can be started as sidecars, but they do not replace FORGE authority paths.

## Start Core And Web UI

```bash
docker compose up --build
```

If the native dev core or desktop is already using the default ports, choose alternate published ports:

```bash
FORGE_CORE_PORT=18493 FORGE_DESKTOP_PORT=1421 docker compose up -d --build
```

Open:

- Core: `http://127.0.0.1:18492`
- Web UI build: `http://127.0.0.1:1420`

The Tauri desktop shell still runs through the native desktop workflow. The `desktop-web` container serves the same Vite app as a browser surface for containerized inspection.

## Start Optional Ollama Sidecar

```bash
docker compose --profile ollama up --build
```

Set these values in `.env.docker` if you want the core to talk to the Ollama service through its OpenAI-compatible endpoint:

```dotenv
FORGE_ENABLE_MODEL_RUNTIME=true
FORGE_MODEL_OPENAI_COMPAT_ENDPOINT=http://ollama:11434/v1
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

This deletes the named volumes, including `/data/forge.sqlite`.

## Validation

Useful checks:

```bash
docker compose config
docker compose build core
docker compose up -d core
docker compose exec core wget -qO- http://127.0.0.1:18492/health
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
- Optional provider containers do not execute unless their profiles are selected.
- Optional model/embedding providers do not create truth.
- No public API or route behavior is changed by containerization.
