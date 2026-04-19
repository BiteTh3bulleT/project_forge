# Semantic Syscall Validation Matrix (Phase 2)

| Action | Required Fields | Capability | Mutating? | Approval Possible? | Primary Validation |
|---|---|---|---|---|---|
| `CREATE_NOTE` | `payload.type`, `payload.title`, `payload.content`, scope/provenance envelope | `memory.note.create` | yes | yes | note type/status enum, confidence range, non-empty title/content |
| `CREATE_LINK` | `payload.type`, `payload.sourceId`, `payload.targetId` | `memory.link.create` | yes | yes | link enum, distinct source/target, confidence range, referenced objects exist |
| `UPDATE_STATE` | `payload.key`, `payload.value`, `payload.derivedFrom` | `state.update` | yes | yes | key/value presence, state status enum, non-empty evidence chain |
| `OPEN_LOOP` | `payload.title` (state defaults to `open`) | `loop.open` | yes | yes | initial state must be `open/in_progress/blocked`, priority enum, related ids valid |
| `CLOSE_LOOP` | `payload.loopId`, `payload.reason` or `payload.outcome` | `loop.close` | yes | yes | loop exists, valid loop transition to `resolved` |
| `MARK_SUPERSEDED` | `payload.oldObjectId`, `payload.newObjectId`, `payload.reason` | `memory.supersession.mark` | yes | yes | old/new ids differ and exist, preserve both records, create supersession relation |
| `REGISTER_CONTRADICTION` | `payload.leftObjectId`, `payload.rightObjectId`, `payload.reason` | `memory.contradiction.register` | yes | yes | distinct ids, severity/confidence validation, preserve both sides |
| `DERIVE_MODEL` | `payload.type`, `payload.expression`, `payload.derivedFrom` | `model.derive` | yes | yes | evidence required, support count consistency, confidence range, status starts provisional |
| `ARCHIVE_NOTE` | `payload.noteId`, `payload.reason` | `memory.note.archive` | yes | yes | note exists, valid note transition to `archived`, no hard delete |
| `COMPILE_CONTEXT` | `payload.query` (or metadata query), optional valid budget | `context.compile` | no | no | deterministic read-only compile path, no live LLM requirement |

## Global envelope checks

Applied to all actions:

- supported action lookup
- actor/source required
- workspace scope required
- provenance required for mutating actions
- timestamp/correlation normalization
- idempotency handling
- dry-run no-commit guarantee
