# Cognitive Lanes

## Lane Model

| Lane | Responsibility | Status |
|---|---|---|
| Neural Lane | Ingest, normalization, event admission, evidence classification | PARTIAL |
| Arterial Lane | Context restore, working memory, reasoning, planning, response | PARTIAL |
| Lymphatic Lane | Dream Mode, consolidation, repair, diagnostics | PARTIAL |
| Kernel / Control Lane | Semantic syscalls, validation, commit, audit | IMPLEMENTED |

## Hot Path vs Background Path

Hot path:

1. Operator/chat/API request.
2. Context assembly or modelruntime inference if needed.
3. Gateway for tool execution or syscall kernel for semantic mutation.
4. Audit and trace.
5. Operator result.

Background path:

1. Ingest/retrieval/repair/Dream/autonomy sweep.
2. Proposal generation.
3. Dry-run reports or governed syscall/tool requests.
4. Operator review where required.

## Implementation Reality

IMPLEMENTED: `aios/controllane`, `aios/compute/librarian`, `aios/truth`, `aios/autonomy`, `aios/dream`, and `aios/iolane` exist.

PARTIAL: The lane model is not fully enforced as runtime isolation. Some packages are interfaces/scaffolds while current behavior lives in existing services.

