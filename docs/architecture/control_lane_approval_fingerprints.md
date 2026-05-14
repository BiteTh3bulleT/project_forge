# Control Lane Approval Fingerprints

Status: M5A Phase 3 seam, pure deterministic helper only.

## Purpose

Control Lane approval fingerprints provide a stable, inspectable identity for the semantic operation that an approval request is about. They are meant to make approval packets, audit records, and future verification checks compare the same bounded operation shape without copying raw prompts, raw secret values, or full payload content.

The current implementation is a pure helper in `services/core/internal/aios/controllane/approval_fingerprint.go`. It does not call a database, create an approval store, evaluate policy, admit evidence, commit state, change existing approval behavior, or make FORGE-K simulator services live authority.

## Fields

The v1 fingerprint contains:

- `version`: schema version, currently `control_lane_approval_fingerprint.v1`.
- `semanticAction`: requested Control Lane semantic action.
- `capability`: action capability from the Control Lane action definition.
- `targetObjectType`: action target object type from the action definition.
- `mutating`: action definition mutation flag.
- `actionClass`: `mutation`, `read_only`, or `validation_only`.
- `actor`: actor id and kind from the syscall envelope.
- `source`: syscall source.
- `workspace`: workspace id from the explicit scope boundary.
- `traceId` and `correlationId`: included when present in the syscall.
- `payloadShapeHash`: SHA-256 over deterministic payload shape metadata, not raw payload content.
- `safeTargetIdentifiers`: sorted bounded identifiers from known safe id fields such as `id`, `noteId`, `sourceId`, and `targetId`.
- `riskClass`: caller-supplied approval risk class when known.
- `approvalRequestId`: durable approval request id when an approval packet exists.
- `decisionStatus`: approval decision status when verifying an existing decision.
- `createdAtMillis` and `expiresAtMillis`: approval lifecycle timestamps when verifying a live approval window.

## Payload Shape

The payload hash is intentionally a shape hash, not a content hash. The canonical shape records deterministic map keys, nested structure, scalar type classes, and list lengths. Map keys are sorted before hashing, so Go map iteration order cannot affect the fingerprint.

Raw strings are represented as `string`; the helper does not hash or expose prompt bodies, note content, secret values, or other full text. Target identifiers are handled separately through the bounded safe identifier list, and only accepted when the key is known and the value is short and path/id-like.

## Lifecycle

The intended lifecycle is:

1. A proposer submits a `SyscallRequest`.
2. Control Lane resolves the action definition.
3. An approval packet builder may create a fingerprint from the request, definition, risk class, and approval request id.
4. A durable approval system may store the fingerprint beside the approval request.
5. A later verifier may rebuild the fingerprint from the candidate syscall and compare it with the stored approval fingerprint before trusting the decision.

M5A Phase 3 implements only step 3 as a pure helper and tests. Steps 4 and 5 remain future integration work and must use the existing durable approval/audit systems rather than creating a second approval database.

## Dry Runs

Dry-run syscalls may be fingerprinted for preview or packet drafting, but a dry-run fingerprint is not a commit authorization. The fingerprint records the operation shape only; the existing Control Lane dry-run handling still decides whether anything is committed. In current behavior, dry-runs do not commit semantic state.

## Validation-Only Actions

Validation actions such as `VALIDATE_SEMANTIC_OPERATION`, `VALIDATE_REF_SHAPE`, and `VALIDATE_KV_IDENTITY` are represented as `actionClass: validation_only` when the action definition is non-mutating and the action name is validation/comparison shaped. This keeps validation approvals distinguishable from semantic mutation approvals.

Validation-only fingerprints do not imply authority to execute the operation being described inside the validation payload. For example, a `VALIDATE_SEMANTIC_OPERATION` payload may describe a mutating operation, but the fingerprint action remains the validation action and its capability remains the validation capability.

## System-Internal Validation

`source: system` or `source: internal_cell` is recorded as an input fact only. It is not an implicit approval, waiver, or bypass. Any future approval verifier must compare fingerprint identity and then apply the existing approval policy and audit boundaries explicitly.

## Autonomy Proposals

Autonomy, rule cells, model-driven neurons, adapters, and future IRIS may use fingerprints to draft approval packets or attach evidence to a proposal. They still propose only. The Kernel/Control Lane remains the semantic commit authority, and approvals remain separate gates.

## Future FORGE-K Migration

The fingerprint model is compatible with future FORGE-K migration because it is deterministic, bounded, and authority-neutral. It can be shared as a validation contract in a later phase, but it must not route live state mutation through FORGE-K simulator services without an explicit integration design, tests, audit story, and documentation update.

## Tests

Focused coverage lives in `services/core/internal/aios/controllane/approval_fingerprint_test.go` and proves:

- deterministic output for equivalent inputs,
- stable map ordering,
- fingerprint changes when action, source, actor, workspace, or payload shape changes,
- validation-only action representation,
- payload shape hashing does not expose raw full content.
