# Gateway / Approval / Tool Review

## Scorecard

- Gateway-only execution: GOOD
- Tool taxonomy: GOOD
- Capability defaults: GOOD/PARTIAL
- Approval gates: GOOD/PARTIAL
- Approval fingerprinting: GOOD
- Dangerous defaults: GOOD/PARTIAL
- Service-specific tests: PARTIAL

## Findings

GOOD: Gateway has a governed tool registry, policy checks, invocation records, audit/trace propagation, and approval fingerprint binding.

GOOD: Dangerous tools are generally approval-only or guarded by capability status.

RISK: Capability status management can change dangerous tool posture without high-risk approval semantics.

RISK: `net.fetch` and network connectivity tools need SSRF/private-network denial.

RISK: `proc.run` buffers stdout/stderr before final truncation, creating memory-pressure risk.

PARTIAL: Service-specific harness tests exist, but coverage should expand for archive extraction, symlink paths, network denial, and output limits.

## Punchlist

- `GATE-001`: Add high-risk approval gate for dangerous capability activation.
- `GATE-002`: Add SSRF denial tests and implementation.
- `GATE-003`: Bound process output before buffering in memory.
- `GATE-004`: Add symlink traversal tests for file tools.
- `GATE-005`: Add route test proving direct adapter execution remains unavailable.

