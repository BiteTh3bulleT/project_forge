# Safe Migration Control

## Migration must prove

- old path still works or is intentionally retired
- new path is feature-flagged until promoted
- rollback is tested
- audit/provenance remains linked
- no second authority path exists

## Migration sequence

1. mirror
2. validate
3. shadow compare
4. disabled live path
5. operator-approved live path
6. default live path
7. retire legacy path
