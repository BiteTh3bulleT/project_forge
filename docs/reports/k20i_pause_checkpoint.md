# K20I pause checkpoint

Date: 2026-08-14

## Last completed cutover

K20H is complete and pushed to `main` as commit `684bdf7` (`feat: materialize admitted memory through forge-k`). It provides the governed Court-to-memory materialization and revision path, retires live legacy observation writers, and moves active VSA projection identity to immutable admitted FORGE-K memory evidence.

The K20H aggregate gate passed before that commit:

- `npm run validate:local`
- `npm test`
- `npm run lint`
- `npm run validate:js`
- repository hygiene, generated route, OS-integration, Rust parity, and Go build checks

## Bounded K20I checkpoint

Work stopped before K20I shared action, storage, API, or compiler integration began. This checkpoint contains only two independently testable foundations:

1. `internal/forgekernel/semanticdiff` is a pure production package for the proposed `semantic.diff.v1` operator. It performs bounded UTF-8/NFKC/case-folded token-set difference with deterministic sorting and a content digest. It has no database, clock, model, gateway, simulator, or mutation authority.
2. The semantic syscall facade now reports persisted `COMPILE_CONTEXT` requests as durable mutations with rollback requirements. It continues to mark snapshots as non-canonical: `mutatesDurableData=true` and `mutatesCanonicalData=false`.

Focused checkpoint validation:

```text
go test ./internal/forgekernel/semanticdiff ./internal/aios/controllane \
  -run 'TestCompute|TestFingerprint|TestBuildSemanticSyscallFacade' -count=1
```

## Resume order

K20I is not complete and none of the following may be claimed live yet:

1. Integrate `COMPUTE_SEMANTIC_DIFF` as a FORGE-K-only syscall over exact-scope, current, admitted K20H evidence. Add immutable operation/result/non-canonical derived-object storage, sealed content commitments, replay, rollback, backup inspection, and sole-writer guards.
2. Make `COMPILE_CONTEXT` FORGE-K-only, require idempotency for persistence, deny Adapter/FutureIRIS execution, and remove the authority-affecting global restore cache from the production path.
3. Land the pure production Context Compiler contract with exact selected-path source manifests, governed feedback heads, fixed-point scoring, policy digests, decision commitments, and commit-time CAS revalidation.
4. Replace legacy context inputs with fail-closed admitted K20H evidence reads. Preserve snapshots as non-canonical evidence and mark old snapshots inspection-only.
5. Add narrow authenticated Court admission/appeal and memory materialization/revision routes. Do not expose a generic syscall endpoint; derive identity, scope, policy, provenance, hashes, timestamps, and idempotency server-side.
6. Run full Go, desktop, Rust parity, repository hygiene, OS-integration, and local aggregate gates before declaring K20I complete.

## Authority status

FORGE-K remains partial production authority at this checkpoint. Semantic Diff is pure computation only, and the live Context Compiler decision remains outside the production Kernel. `legacy_v1` and the temporary Control Lane durable adapter are still present and must not be retired until later cutover gates pass.
