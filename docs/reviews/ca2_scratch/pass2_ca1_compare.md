# PHASE CA2 — Pass 2: Compare Against CA1

**Date:** 2026-05-19

## Timeline

When CA2's auditors began work, no CA1 artefact was present in the working tree (`find docs -iname "*ca1*"` returned nothing; `docs/archive/phases/` contained only `PhaseCA2.txt`; the repo-root `Full-Code-Review.md` is a prompt template). Pass 2 was therefore initially recorded as "CA1 absent → all CA1 items unable to verify."

During the CA2 audit window, a remote merge brought commit `3a671b4 Add CA1 codebase integrity audit` into the local tree, adding:

- `docs/archive/phases/PhaseCA1.txt`
- `docs/reports/phase_ca1_full_codebase_integrity_audit.md`
- `docs/status/phase_ca1_full_codebase_integrity_audit.md`
- `docs/reviews/full_codebase_integrity_audit.md`
- `docs/reviews/full_codebase_integrity_findings.csv`
- `docs/reviews/full_codebase_integrity_fix_queue.md`

CA2's audit work was performed **independently** of CA1's findings. The overlap analysis below is post-hoc.

## CA1 ↔ CA2 overlap

| CA1 ID | CA1 severity | Topic | CA2 ID | CA2 severity | Notes |
|---|---|---|---|---|---|
| CA1-001 | Critical | Desktop shell shutdown/reboot bypass | M-1 | Medium | Same observation. CA2 inspected the Tauri binary directly and verified the `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` policy gate at `main.rs:520-557` with unit tests at lines 765/780. Downgraded from Critical to Medium because the gate is binary-enforced and disabled by default; the live defect is **docs drift**, not unprotected mutation. Resolved in CA3 via 6-file supersession. |
| CA1-002 | High | Default `/` workspace authority | — | — | Not surfaced by CA2 (config-security scope). Treat CA1 fix queue as authoritative. |
| CA1-003 | High | Docker Compose wildcard bind defaults | — | — | Not surfaced by CA2. Treat CA1 as authoritative. |
| CA1-004 | High | Legacy adapter gateway tool reaches Ollama outside modelruntime governance | Touched in §11/§16 | Low | CA2 classified the legacy adapter as `Executes()==false` read-shim; did not analyse Ollama reachability path. CA1's scope is more thorough on this — adopt CA1 finding. |
| CA1-005 | High | Hardcoded client-side login credentials | — | — | Not surfaced by CA2. Treat CA1 as authoritative. |
| CA1-006 | High | Persisted `ollamaBaseUrl` SSRF validation gap | — | — | Not surfaced by CA2. Treat CA1 as authoritative. |
| CA1-007 | High | VM/Ollama docs vs Nix service enablement | — | — | Not surfaced by CA2. |
| CA1-008 | Medium | Plain `http.Error` responses across API | — | — | Not surfaced by CA2. |
| CA1-009 | Medium | Dashboard zero-vs-unavailable display | — | — | Not surfaced by CA2 directly. (CA2 H-1 / M-7 address an analogous SystemPage issue.) |
| CA1-010 | Medium | `FALLBACK_OPERATOR_APPS` duplicated catalog | H-2 | High | Same finding. CA2 graded High because the lists had drifted (different `native` values + labels). CA3 resolved by adopting `AppShellSurfaces` as source of truth per operator decision. |
| CA1-011 | Medium | Deep-link routes orphaned in shell window mode | L-2 | Low | Same observation. Severity differs; both flagged for follow-up. |
| CA1-012 | Medium | Duplicate `fs.mkdir` lane metadata | — | — | Not surfaced by CA2. |
| CA1-013 | Medium | Supersession banners for stale status docs | Touched in §18 | — | CA2 §18 lists `docs/status` files as current; CA1's stale ones (`desktop_shell_status.md`, `desktop_nix_packaging_gap.md`) were not flagged. Treat CA1 as authoritative. |
| CA1-014 | Medium | G3.5 future-language mixed with G8 truth | M-1 (overlap) | — | M-1 supersession touches the same doc; CA1's narrower G3.5 wording is still open. |
| CA1-015 | Medium | VM verification status stale | — | — | Not surfaced by CA2. |
| CA1-016 | Low | Debug console window kind without owner | — | — | Not surfaced by CA2. |
| CA1-017 | Low | Tool registry placeholder hides drift | L-1, L-2 (overlap) | Low | Adjacent but not identical. |

## CA2-only findings (not in CA1)

- **H-3** — `services/core/internal/hostbridge/TestExecRunnerRejectsOversizeCommandOutput` timing fragility (CA1 reported tests passing; the failure mode is environment-sensitive). Fixed in CA3.
- **H-4** — `ChatPage.test.tsx:262` (already green at HEAD; CA1 saw 138 tests pass, CA2 saw 158 with one apparent failure that resolved itself between the audit and CA3 verification).
- **M-2** — `os.Exit(1)` from HTTP server goroutine in `services/core/main.go`. Fixed in CA3.
- **M-3** — Empty `FORGE_API_TOKEN` lacks startup warn. Fixed in CA3.
- **M-5** — Autonomy ID-prefix filter relies on convention; regression test added in CA3.
- **M-7** — `statusClass()` silent fallback in `SystemPage.tsx`. Fixed in CA3.

## Conclusion

CA1 and CA2 are complementary, not contradictory. CA1 is broader on configuration-security and docs-supersession; CA2 is deeper on test-timing fragility, startup-lifecycle defects, and the autonomy ID-prefix contract. Both flag the desktop shell power actions (CA1 as Critical, CA2 as Medium-after-verification) and the operator-app catalog duplication.

For remediation: use the **union** of CA1 and CA2 fix queues. CA3 (this session) closed CA1-001 (docs supersession), CA1-010 (catalog dedup), and the CA2-only items listed above. The remaining CA1 items (CA1-002..009, CA1-012, CA1-013, CA1-015, CA1-016) remain open and are CA1's authority to manage.
