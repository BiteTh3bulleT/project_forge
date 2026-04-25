# Hugging Face TEI Embeddings

Status date: 2026-04-24.

FORGE can optionally use Hugging Face Text Embeddings Inference (TEI) as an embedding provider. TEI is an accelerator/provider bolt-on: embedding vectors remain retrieval evidence and never become canonical memory truth.

## Enable

```bash
export FORGE_EMBEDDING_PROVIDER=tei
export FORGE_EMBEDDING_TEI_ENDPOINT=http://127.0.0.1:8081
```

Optional:

```bash
export FORGE_EMBEDDING_MODEL=bge-large
export FORGE_EMBEDDING_DIMS=1024
export FORGE_EMBEDDING_TEI_API_KEY=...
export FORGE_EMBEDDING_TEI_TIMEOUT_MS=30000
```

TEI can also be configured through durable settings:

- `embedding_provider=tei`
- `embedding_tei_endpoint`
- `embedding_tei_api_key`
- `embedding_tei_timeout_ms`

## Behavior

- `GET /api/embeddings/status` reports current provider health.
- `POST /api/embeddings/reembed` uses the configured provider for vector rebuilds.
- `GET /api/providers/capabilities` exposes TEI capability state.
- Dream Mode v0 dry-run reports include embedding refresh proposals for memory/snapshot candidates, but do not run or commit refreshes automatically.

If TEI is unavailable, the embedding backend reports degraded while the core remains healthy. The local hash provider remains the safe default.

## Safety

- TEI is not required for memory.
- TEI is not a generation backend.
- TEI does not write canonical memory/state/loop/journal rows.
- Vector search results are retrieval evidence only and cannot replace syscall-controlled truth.
