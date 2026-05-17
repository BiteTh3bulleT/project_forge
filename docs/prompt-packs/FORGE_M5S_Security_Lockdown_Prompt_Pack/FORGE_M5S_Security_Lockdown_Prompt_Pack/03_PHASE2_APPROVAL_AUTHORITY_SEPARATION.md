# Phase 2 — Approval Authority Separation

Approval identity must come from backend auth context, not JSON body actor.

High-risk approvals are non-public by default.

Tests:
- unauth approval rejected.
- spoofed body actor rejected.
- high-risk self/public approval blocked.
- authorized approval audited.
