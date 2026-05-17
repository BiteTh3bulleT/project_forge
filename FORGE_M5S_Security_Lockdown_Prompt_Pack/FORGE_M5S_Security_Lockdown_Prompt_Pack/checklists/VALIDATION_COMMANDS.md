# Validation Commands

```bash
npm test
npm run lint
npm run validate:js
npm run validate:local
npm run build
npm run smoke
```

Focused:

```bash
cd services/core && go test ./internal/api ./internal/approvals ./internal/jobs ./internal/projectcontext ./internal/gateway -count=1
npm -w @forge/desktop run test
npm -w @forge/desktop run typecheck
```
