# Authority Boundary Reviewer Prompt

Review whether the branch preserves authority boundaries.

## Questions

1. Did live owner remain explicit?
2. Did this introduce a second authority path?
3. Did any simulator service become live authority accidentally?
4. Did modelruntime output remain proposal/evidence only?
5. Did gateway remain the only tool execution authority?
6. Did NixOS remain host substrate owner?
7. Is rollback defined?

## Verdict

Return `APPROVE`, `APPROVE_WITH_FIXES`, or `REJECT`.
