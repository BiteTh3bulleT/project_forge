# Validation Commands

## Broad validation

```bash
npm test
npm run lint
npm run validate:js
npm run validate:local
npm run build
npm run smoke
```

## Focused backend validation

```bash
cd services/core && go test ./internal/gateway -count=1
cd services/core && go test ./internal/api -count=1
cd services/core && go test ./internal/modelruntime -count=1
cd services/core && go test ./internal/aios/controllane -count=1
cd services/core && go test ./internal/hostbridge -count=1
cd services/core && go test ./internal/forgeh -count=1
```

## Focused desktop validation

```bash
npm -w @forge/desktop run typecheck
npm -w @forge/desktop run test
npm -w @forge/desktop run build
```

## When a command fails

Record command, result, failure type, evidence, narrower fallback command, and fallback result. Do not mark environment-blocked commands as passed.
