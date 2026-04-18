# Dossiers

## Purpose

Dossiers are durable project-level memory profiles. They prevent FORGE from treating every task as an isolated request.

## Data Model

Primary tables:

- `dossiers`
- `dossier_sources`
- `dossier_jobs`
- `dossier_packets`
- `dossier_briefs`

A dossier stores:

- name/description
- primary paths and related repos
- constraints and preferred adapters
- important files
- routing notes

## Lifecycle

1. Create dossier from source ids and metadata.
2. Link jobs/packets during job packet preparation when `dossierId` is supplied.
3. Regenerate dossier brief snapshots as work evolves.
4. Use dossier scope in retrieval runs and insights.

## API

- `GET /api/dossiers`
- `POST /api/dossiers`
- `GET /api/dossiers/{id}`
- `PATCH /api/dossiers/{id}`
- `POST /api/dossiers/{id}/briefs/generate`

## UI

Route: `/dossiers`

- dossier list and selection
- dossier detail (sources, recent jobs, briefs)
- create flow
- brief regeneration

## Operational Guidance

- keep one dossier per meaningful operational context (repo/service/program)
- do not over-scope a dossier with unrelated source trees
- use routing notes to capture hard constraints and known adapter tendencies
