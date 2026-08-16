# Production runtime proposal pure contract

Status: **implemented, live-wired on every model visibility surface, and tested**.

Package: `services/core/internal/forgekernel/runtimeproposal`

## Boundary

The package converts output returned by the governed `modelruntime` path or the native Ollama path into one deterministic, hash-bound `Decision`. It is pure standard-library Go: no I/O, clock, database, model, gateway, domain, Control Lane, or simulator dependency.

The envelope binds:

- exact output bytes, their recomputed SHA-256 hash, the driver-declared hash, and the verification result;
- source kind, driver/runtime/model/tokenizer identity and revisions;
- workspace, lane, normalized selected paths, context decision digest, context bundle hash, and exact prompt hash;
- provenance, request, correlation, trace, and audit identities;
- normalized gateway evidence references and their aggregate commitment.

`accepted_proposal` means only that the proposed final text passed this boundary. The envelope always remains non-canonical, proposal-only, and unable to select or execute tools, admit evidence, mutate memory, or assert canonical truth. Any later semantic mutation still requires a FORGE-K syscall, deterministic validation, Courthouse admission where applicable, and a Kernel commit.

## Deterministic visibility policy

Malformed identity, scope, hashes, provenance, or gateway evidence fails closed as an invalid input. A valid-shaped input is classified `withheld` before final visibility when:

- recomputed output bytes do not match the declared hash;
- the driver claims model-output, truth, evidence-admission, memory-mutation, tool-selection, or tool-execution authority;
- gateway evidence has a workspace/lane or correlation/trace mismatch;
- output claims a completed action without at least one successful, exactly scope- and trace-bound gateway evidence reference.

A withheld decision returns fixed safe text, never the driver output. Evidence order and selected-path order normalize before hashing. `VerifyDecision` reconstructs the complete decision and rejects visible-text, status, envelope, authority, or digest tampering.

Gateway evidence validation here proves reference shape and exact binding, not existence in a live store. A future caller must populate references from a trusted gateway/audit resolver; model-supplied references are not admissible.

## Live integration

The production API coordinator calls this package after a driver returns but before any response byte becomes user-visible or any assistant reply is persisted:

1. The current transition coordinator binds the exact prompt bytes and a deterministic prompt-bundle identity. This is fail-closed runtime binding, but it is not yet the live Kernel Context Compiler decision; that replacement remains the next cutover gate.
2. A trusted runtime adapter resolves the actual driver/runtime/model/tokenizer versions, retains the exact returned bytes, and supplies the driver's output commitment.
3. If a response reports tool completion, the API resolves the actual gateway invocation, exact request/result commitments, and durable audit-record identity. It never accepts those fields from model output.
4. The coordinator calls `runtimeproposal.Decide`. Only `accepted_proposal` content may become a final chat response. Streaming paths must buffer driver tokens or label them non-visible proposal material until this decision exists.
5. Persisted/read or replayed decisions call `runtimeproposal.VerifyDecision`. Semantic assertions remain proposals until separately admitted and committed by the Kernel.

The seam now covers ordinary chat, native Ollama chat and streaming, gateway tool-loop synthesis, `/forge/models/{id}/chat`, and `/v1/chat/completions`. Streaming buffers driver text and reasoning until the final decision. Gateway stage events expose hashes and sizes instead of raw model JSON or tool arguments.

## Deliberately not included

- no import from simulator `internal/forgek`;
- no admission of model output, canonical-state mutation, tool selection, tool execution, or approval authority;
- no claim that the transitional prompt binding is a Kernel-owned Context Compiler decision;
- no driver-supplied gateway evidence or authority claim can grant visibility.

## Verification

The focused suite covers both source kinds, exact hash acceptance and mismatch withholding, every forbidden authority claim, gateway evidence scope/trace binding and normalization, action-completion gating, malformed inputs, duplicate invocation rejection, decision tamper detection, deterministic golden digest, fuzzed no-authority invariants, and a source-import purity guard.

Commands:

```text
cd services/core
go test ./internal/forgekernel/runtimeproposal -count=1
go test -race ./internal/forgekernel/runtimeproposal -count=1
go vet ./internal/forgekernel/runtimeproposal
```

Result on 2026-08-16: all three commands passed.
