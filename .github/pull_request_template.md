## Summary

Describe the problem and the change in a few sentences.

## Scope

- [ ] Core daemon
- [ ] Production Kernel / authority boundary
- [ ] Gateway / tools / approvals
- [ ] Modelruntime / model visibility
- [ ] Memory / retrieval / evidence
- [ ] Backup / recovery / storage
- [ ] Desktop / Tauri native capabilities
- [ ] Nix / NixOS / host integration
- [ ] Documentation only
- [ ] Other

## Authority and trust impact

State the authority owner before and after this change.

- Authority change: `none | narrowed | expanded | migrated`
- Canonical state affected: `yes | no`
- Tool execution affected: `yes | no`
- Model-visible output affected: `yes | no`
- Authentication or actor provenance affected: `yes | no`
- Host behavior affected: `yes | no`

Explain any non-`none` or `yes` answer. Do not use “no authority change” as a substitute for analysis.

## Behavior and operator impact

What changes for the operator? Include new failure, degraded, approval, or recovery states.

## Validation

List commands and results. Mark skipped checks and explain why.

```text
npm test
npm run lint
npm run validate:js
npm run validate:desktop
npm run validate:forgek
npm run validate:local
npm run smoke
```

Additional targeted tests:

```text
# Add commands and results here.
```

## Failure, rollback, and recovery

Describe:

- expected failure behavior;
- whether the change fails closed;
- rollback or disable path;
- migration or compatibility concerns;
- recovery steps for partial failure or corrupted state.

Use `not applicable` only with a brief reason.

## Documentation

- [ ] `docs/status/CURRENT_STATE.md` reviewed or updated
- [ ] authority map reviewed or updated
- [ ] runbooks reviewed or updated
- [ ] API route inventory regenerated when needed
- [ ] architecture/status labels are honest
- [ ] historical documents were not presented as current truth

## Security and privacy

- [ ] No credentials, bearer tokens, API keys, or private data are included
- [ ] Logs and fixtures use synthetic or redacted values
- [ ] Request, payload, collection, retry, and timeout bounds are explicit where relevant
- [ ] Path, workspace, symlink, archive, and capability scope were reviewed where relevant
- [ ] Asynchronous work preserves authenticated actor and provenance identity where relevant

## Checklist

- [ ] The change is focused and reviewable
- [ ] Tests cover success and fail-closed behavior
- [ ] No model or adapter gains undeclared authority
- [ ] No direct tool-execution bypass was added
- [ ] No retired mutation or restore path was silently reopened
- [ ] Documentation matches the implemented behavior
- [ ] Required checks are ready to run on this PR
