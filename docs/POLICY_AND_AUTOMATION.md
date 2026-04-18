# Policy and Automation

## Approval Presets

Approval presets are reusable safety profiles stored in `approval_presets`.

Default profiles:

- `conservative`
- `balanced`
- `aggressive`

Global profile key: `settings.approval_preset_global`

Dossier profile can override the global preset.

## Routing Policy Recommendations

API:

- `POST /api/policy/recommend`
- `GET /api/policy/recommendations`

Request inputs:

- `taskType`
- optional `dossierId`
- optional `strategyId`
- `objective`

Output fields include:

- `targetAdapter`
- `retrievalMode`
- `packetShape`
- `approvalPresetId`
- `approvalRequired`
- `confidence`
- `reasons`
- `evidence`
- `inferred`
- `operatorOverrideAllowed`

## Execution Strategies

API:

- `GET /api/strategies`
- `POST /api/strategies`

Each strategy defines:

- task type
- adapter
- retrieval mode
- packet rules
- approval requirement
- expected artifacts
- success criteria
- retry guidance

## Dossier Profiles

API:

- `GET /api/policy/dossiers/{id}`
- `POST /api/policy/dossiers/{id}`

Profile fields:

- preferred strategies
- preferred adapters
- approval preset override
- retrieval defaults
- high-value files
- noisy files
- routing notes
- automation bindings

## Automation Rules

API:

- `GET /api/automation/rules`
- `POST /api/automation/rules`
- `POST /api/automation/run`
- `GET /api/automation/history`

Rule structure:

- trigger
- condition
- action
- scope
- enabled
- dry-run default

Current bounded action types:

- `create_job`
- `generate_dossier_brief`
- `create_review`
- `suggest_strategy_adjustment`

Automation history is append-only and stores matched/dry-run/executed status with preview/result payloads.

## Packet Guidance

API:

- `POST /api/packet-guidance/analyze`
- `GET /api/packet-guidance`

Guidance records store:

- score
- issues
- recommendations
- evidence

No automatic packet mutation is performed.
