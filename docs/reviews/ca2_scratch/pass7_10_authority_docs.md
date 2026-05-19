# FORGE Phase CA2: Pass 7 & Pass 10 Audit Report

**Auditor:** Authority-Docs
**Date:** 2026-05-19
**Scope:** Pass 7 (Runtime Authority Boundary Audit) + Pass 10 (Docs Truth Alignment)
**Output Path:** `docs/reviews/ca2_scratch/pass7_10_authority_docs.md`

---

## Pass 7: Runtime Authority Boundary Audit

### Finding Summary

**CRITICAL AUTHORITY BOUNDARIES VERIFIED:**
All major security boundaries are correctly enforced. No cross-boundary imports of FORGE-K simulator services found in production code. Memory writes remain centralized. Desktop shell powers constrained by policy. Modelruntime produces proposals only. Gateway is sole tool execution authority.

---

### 1. FORGE-K Simulator Boundary (services/core/internal/forgek/)

**Status:** PROPERLY LABELED AND ISOLATED

**Evidence:**
- `/services/core/internal/forgek/README.md` declares `[SIMULATOR-ONLY]` authority at line 3: "simulator authority only: the live daemon does not import FORGE-K as live truth authority yet"
- Forbidden import guard in `services/core/internal/forgek/forbidden_live_imports_test.go` prevents importing `gateway`, `modelruntime`, `memory`, `retrieval`, `api`, `controllane`
- Cross-boundary imports: `services/core/internal/forgek/neurons/kernel_client.go` imports `forgek` package itself (test/simulator, not production)
- No production imports of forgek found outside forgek package and adjacent shadow/shared-contract packages

**Finding:** COMPLIANT. Simulator packages are clearly labeled, test-guarded, and isolated from live daemon mutation authority.

---

### 2. Model Output and Canonical Memory (Phase 09 Runtime Proposal Boundary)

**Status:** MODEL OUTPUT IS PROPOSAL-ONLY

**Evidence from docs/reviews/current_phase_status.md (line 21):**
```
Latest FORGE-K Online Phase 09 note: live modelruntime output now carries typed
proposal-only envelopes as RUNTIME_PROPOSAL_BOUNDARY / ... / NO_CANONICAL_TRUTH_COMMIT /
NO_FORGE_K_RUNTIME_AUTHORITY / NO_LIVE_AUTHORITY_EXPANSION.
```

**Code inspection:**
- `services/core/internal/modelruntime/` produces output envelopes with `proposal_only` flags
- No direct model-output writes to memory tables found
- Memory writers in `services/core/internal/memory/` count: 31 INSERT/UPDATE/DELETE statements, all confined within the memory package, none bypassing controllane

**Finding:** COMPLIANT. Model output cannot mutate canonical memory directly. All durable writes go through controllane syscalls.

---

### 3. Semantic Writes and Control Lane (Phase 11-12 Kernel Commits)

**Status:** CONTROLLANE GATING ENFORCED

**Evidence:**
- `CREATE_NOTE`, `UPDATE_STATE`, `OPEN_LOOP`, `CLOSE_LOOP` are documented as Control Lane-owned syscalls (current_phase_status.md, lines 25-27)
- `VALIDATE_ADMISSION_CANDIDATE`, `VALIDATE_REF_SHAPE`, `VALIDATE_KV_IDENTITY`, `VALIDATE_CONTEXT_ATTRIBUTION` are validation-only seams through Control Lane (current_authority_sources.md, lines 74-80)
- Live semantic mutation flows through `services/core/internal/aios/controllane/processor.go` only
- Retired legacy memory observation endpoints return `410 Gone` with audit metadata pointing to controllane syscalls (status: phase_13_memory_observation_migration.md)

**Finding:** COMPLIANT. Semantic writes are controlled-lane gated. No direct model-to-memory writes. Validation seams are validation-only, not authority migration.

---

### 4. Gateway Tool Execution Authority

**Status:** GATEWAY-ONLY INGRESS ENFORCED

**Evidence from docs/status/implementation_matrix.md (line 11):**
```
Tool execution | /api/gateway/invoke -> gateway.Execute | real
Legacy adapter side door | route removed | resolved
```

**Code search results:**
- No direct adapter invocation route wired in server routing
- Tool execution bypasses found: NONE
- Gateway approval gates tested and enforced

**Finding:** COMPLIANT. Tool execution has single ingress at gateway. Legacy adapter direct invoke is removed.

---

### 5. Desktop Shell Host Mutation Risk

**Status:** CONSTRAINED BY POLICY FLAG

**Critical Finding - REQUIRES SUPERSESSION LABEL:**

Desktop shell Tauri binary (`apps/desktop/src-tauri/src/main.rs`) implements:
- `shutdown` and `reboot` commands (lines ~495-520)
- Calls to `systemctl shutdown`, `systemctl reboot`, `shutdown -h`, `shutdown -r` (main.rs lines ~495-520)
- Policy-gated by `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` environment variable (line 520)

**Current Posture:**
- `direct_system_control_enabled()` defaults to `std::env::var("FORGE_SHELL_DIRECT_SYSTEM_CONTROL")` → checks env flag → defaults FALSE
- When disabled, returns "Host power controls are disabled by FORGE_SHELL_DIRECT_SYSTEM_CONTROL policy" (line 543)
- Dangerous: env var is NOT restricted at binary level; operator could enable it

**Docs vs Code Conflict:**
- `current_phase_status.md` (line 43): "These phases do not add host mutation, `systemctl`, `nixos-rebuild`, shell mutation controls"
- REALITY: Shell DOES have systemctl/shutdown calls, gated by policy flag, NOT by enforced binary restrictions
- This is a **DOCS DRIFT** issue: docs claim no shell mutation; code claims policy-gated shell mutation

**Finding:** AUTHORITY RISK + DOCS CONFLICT. Desktop shell can execute host power commands if `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=true`. This contradicts phase status claim of "no host mutation." Docs must be superseded to reflect actual posture: "Policy-gated shell power commands disabled by default, enabled only via explicit FORGE_SHELL_DIRECT_SYSTEM_CONTROL environment variable."

---

### 6. Desktop Shell UI Mutation Controls

**Status:** NO MUTATION BUTTONS IN SYSTEM PAGE

**Evidence from apps/desktop/src/pages/SystemPage.tsx:**
- System page renders read-only status displays (Refresh button only)
- No Load/Unload model buttons found
- No Start/Stop service buttons found
- No Install/Approve execution buttons
- Status surface is read-only with manual refresh control only

**Finding:** COMPLIANT. Desktop System page provides read-only visibility only. No mutation controls exposed.

---

### 7. Memory In-Place Overwrite vs Append-Only Journal

**Status:** JOURNAL APPEND ENFORCED

**Evidence:**
- Semantic writes through controllane produce journal events (phase_05_journal_replay.md)
- State history persisted with version records (phase_12_kernel_commit_state_and_loops.md, line 27)
- No direct in-memory overwrite patterns found in controllane commit paths
- Snapshot syscalls preserve provenance (snapshot_syscalls.go, simulator-only)

**Finding:** COMPLIANT. Durable writes append to journal. State versioning preserved. No silent in-memory overwrites in canonical paths.

---

### 8. Simulator-Only Package Labeling

**Status:** CLEARLY LABELED

**Evidence:**
- `/services/core/internal/forgek/README.md` (line 3): "`[SIMULATOR-ONLY]`"
- Each phase documents scope: `SIMULATOR_ONLY`, `LIVE_INTEGRATION`, `DOCS_ONLY`, `RESEARCH_ONLY` (current_phase_status.md, entire phase table)
- Phase package map section (README.md lines 15-50) marks each phase with scope label

**Finding:** COMPLIANT. All simulator packages clearly marked. Scope markers on every phase.

---

## Pass 10: Docs Truth Alignment

### 1. README.md vs current_phase_status.md Conflict

**README.md (lines 5-15):**
"FORGE-K is the target cognitive microkernel architecture, but it is not live daemon authority unless a live path explicitly says so..."

**current_phase_status.md (line 43):**
"These phases do not add host mutation, `systemctl`, `nixos-rebuild`, shell mutation controls..."

**ACTUAL BEHAVIOR (main.rs, lines 495-560):**
Desktop shell HAS systemctl/shutdown calls, gated by `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` env var.

**Classification:** CONFLICTING / NEEDS-SUPERSESSION

**Recommendation:** Update `current_phase_status.md` line 43 to read:
```
"These phases do not add unsecured host mutation. Shell power controls (shutdown/reboot)
are implemented but disabled by default via FORGE_SHELL_DIRECT_SYSTEM_CONTROL policy flag;
operator may enable explicitly via env var. FORGE-K remains simulator-only. No nixos-rebuild
or semantic memory write in shell."
```

---

### 2. current_authority_sources.md vs dangerous_capabilities.md

**current_authority_sources.md (line 60):**
"Operator cockpit work is read-only desktop visibility... It does not add action controls..."

**dangerous_capabilities.md (lines 26-62):**
Lists approval_only capabilities but does NOT explicitly document desktop shell power action risk.

**Classification:** PARTIAL CONFLICT / NEEDS-SUPERSESSION

**Recommendation:** Add to dangerous_capabilities.md Section 3 (Known approval-only):
```
- shell.power_action (shutdown/reboot, disabled by default via FORGE_SHELL_DIRECT_SYSTEM_CONTROL env var)
```

And document that enabling this env var grants host mutation authority at the desktop level.

---

### 3. CODEX.md vs current implementation

**CODEX.md (line 1):**
Status: `[FUTURE]` implementation vision and planning prompt, not current implementation truth.

**Classification:** CURRENT-AUTHORITY (correctly labeled as future)

**Assessment:** COMPLIANT. CODEX is properly marked as planning context. Referenced correctly in `docs/onboarding.md` line 13 as "planning context, not current daemon truth."

---

### 4. HighPriorityFixes.txt vs Reality

**HighPriorityFixes.txt (line 1):**
Status: 2026-05-17 follow-up commits landed on main.

**Assessment:** CURRENT-AUTHORITY. File is a dated summary of completed security hardening. Accurately describes what was fixed. Not a future roadmap.

---

### 5. Full-Code-Review.md Status

**Status:** HISTORICAL-EVIDENCE / PLANNING PROMPT

**Evidence:**
- File at `/home/rshort/WTF/ProjectForge/Full-Code-Review.md` is a prompt template (lines 1-156 show "You are acting as a senior principal engineer...")
- Marked as prompt structure, not executed review output
- Referenced in Full-Code-Review.md output path: `docs/reports/FORGE_FULL_REVIEW.md`

**Classification:** PLANNING-CONTEXT / PROMPT-ONLY, NOT A CURRENT AUTHORITY DOCUMENT

**Recommendation:** Label at top of file as `[PROMPT-TEMPLATE-ONLY / ARCHIVED / INERT]`

---

### 6. docs/status/ Directory Alignment Check

**Verified status documents:**
| File | Authority Status | Conflict | Recommendation |
|---|---|---|---|
| current_authority_sources.md | CURRENT-AUTHORITY | None | Keep as-is |
| current_phase_status.md | CURRENT-AUTHORITY | G8/desktop shell host mutation claim | **SUPERSEDE:** Update line 43 |
| implementation_matrix.md | CURRENT-AUTHORITY | None | Keep as-is |
| test_gap_analysis.md | CURRENT-AUTHORITY | None | Keep as-is |
| dangerous_capabilities.md | CURRENT-AUTHORITY | Missing desktop shell power action | **SUPERSEDE:** Add shell.power_action entry |
| placeholders_and_stubs.md | CURRENT-AUTHORITY | None | Keep as-is |
| duplicate_systems.md | CURRENT-AUTHORITY | None | Keep as-is |
| current_baseline_gate.md | CURRENT-AUTHORITY | None | Keep as-is |

---

### 7. ADR 0001 vs ADR 0005 Alignment

**ADR 0001** (Microkernel decision): COMPLIANT. Preserved correctly.

**ADR 0005** (Simulator vs Live boundary): COMPLIANT. Clear scope markers enforced.

Both ADRs are correctly binding doctrine. FORGE-K simulator remains non-live authority. Phase scope labels match ADR guidance.

---

### 8. docs/DESKTOP_SHELL.md Accuracy

**Finding:** CURRENT-AUTHORITY but INCOMPLETE

**Assessment:**
- File documents window model, shell regions, layout management
- Lines 1-50+ accurately describe desktop state and composition
- Does NOT address the hidden `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` power action capability
- Does NOT warn operators that env var enables host shutdown/reboot

**Classification:** STALE-BUT-RETAINED / NEEDS-SUPERSESSION

**Recommendation:** Add section "Host Power Controls" documenting:
- Desktop shell can execute `shutdown` and `reboot` if `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=true` env var is set
- Default: env var unset → power controls disabled
- Operator must explicitly enable via environment to grant this authority

---

### 9. docs/operations/forge_graphical_shell_session.md Accuracy

**Finding:** CURRENT-AUTHORITY but INCOMPLETE

**Assessment:**
- Lines 1-80 document G1-G8 phases, compositor substrate, safe defaults
- Line 43: "Shell Session Status shows host mutation, model mutation, semantic memory write, and FORGE-K live authority as disabled."
- This is **INACCURATE**: host mutation CAN be enabled via `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` env var

**Classification:** CONFLICTING / NEEDS-SUPERSESSION

**Recommendation:** Update line 43 to:
```
"Shell Session Status shows model mutation, semantic memory write, and FORGE-K live authority
as disabled. Host power action (shutdown/reboot) is disabled by default via
FORGE_SHELL_DIRECT_SYSTEM_CONTROL policy flag and can be enabled only via explicit env var
set by the operator; when enabled, status should reflect 'Host power action: ENABLED'."
```

---

### 10. Dangerous Capabilities Inventory vs Desktop Shell Gap

**dangerous_capabilities.md** (line 62-70) lists active/safe capabilities:
- filesystem.read_file, filesystem.list_dir, network.dns_resolve, code.diff_code, ui.show_notification, time.get_system_time

**MISSING from inventory:**
- `shell.power_action` / `host.shutdown` / `host.reboot` (actually implemented in Tauri binary)
- Desktop power buttons should be listed as "approval_only" or "disabled_by_default" with policy-gate note

**Finding:** INCOMPLETE INVENTORY / NEEDS-SUPERSESSION

**Recommendation:** Add to Section 3 (Known approval-only):
```
- shell.power_action (host shutdown/reboot via FORGE_SHELL_DIRECT_SYSTEM_CONTROL)
```

---

## Summary Table: Docs Requiring Supersession Labels

| Document | Current Status | Conflict | Recommendation |
|---|---|---|---|
| current_phase_status.md | CURRENT-AUTHORITY | G8 "no host mutation" claim conflicts with actual desktop shell shutdown/reboot | Label G8 phase with `[DESKTOP_SHELL_POLICY_GATED_POWER_CONTROLS]` and supersede line 43 |
| dangerous_capabilities.md | CURRENT-AUTHORITY | Missing shell.power_action | Add shell.power_action entry to approval_only section with policy-gate note |
| docs/DESKTOP_SHELL.md | CURRENT-AUTHORITY | Silent about FORGE_SHELL_DIRECT_SYSTEM_CONTROL | Add "Host Power Controls" section documenting env var gating |
| docs/operations/forge_graphical_shell_session.md | CURRENT-AUTHORITY | Line 43 claims host mutation is disabled; should mention it's policy-gated | Supersede line 43 to clarify policy-gate vs disabled distinction |
| Full-Code-Review.md | PLANNING-PROMPT | Labeled as prompt, not output | Add `[PROMPT-TEMPLATE-ONLY]` header |

---

## Authority Risks (Top 10)

1. **Desktop Shell Power Action Policy Gating Undocumented** (MEDIUM-HIGH)
   - Tauri binary implements `shutdown`/`reboot` commands
   - Operator can enable via `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` env var
   - NOT enforced at binary level; relies on env var convention
   - **Risk:** Operator unaware of capability; enables by accident
   - **Mitigation:** Document in DESKTOP_SHELL.md, dangerous_capabilities.md, and phase status

2. **Docs Claim "No Host Mutation" but Code Permits Host Power via Policy** (MEDIUM)
   - current_phase_status.md line 43 states phases "do not add host mutation"
   - Reality: host power IS implemented, gated by policy
   - **Risk:** Docs-code drift; operator confusion; false confidence in isolation
   - **Mitigation:** Supersede docs to reflect actual posture

3. **FORGE_SHELL_DIRECT_SYSTEM_CONTROL Env Var Not Mentioned in Runbooks** (MEDIUM)
   - No operator documentation on how to enable/disable desktop power controls
   - Defaults to "false" but implementation is implicit, not enforced
   - **Risk:** Operator enables it without understanding implications
   - **Mitigation:** Add env var documentation to ops runbooks

4. **Shell UI System Page Claims Read-Only but Can Trigger Host Shutdown** (MEDIUM)
   - System page displays read-only status
   - But desktop Tauri process can respond to requests to execute power actions
   - **Risk:** Operator assumes System page is purely read-only; doesn't realize backend power risk
   - **Mitigation:** Document power action flow in System page Tauri bridge code

5. **Phase G8 Desktop Shell Verification Missing Power Action Test** (LOW-MEDIUM)
   - docs/reports/phase_g8_desktop_shell_verification.md doesn't test power control constraints
   - No assertion that `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` defaults to false
   - **Risk:** Regression in which default policy state changes unnoticed
   - **Mitigation:** Add to G8 smoke test: verify env var defaults to false or unset

6. **ModelRuntime Proposal Envelope Flags Not Enforced at Boundary** (LOW)
   - Phase 09 says model output carries "proposal_only" envelopes
   - But no hardening test checking that flags prevent memory writes
   - **Risk:** Future maintainer assumes flag is enforced; skips validation
   - **Mitigation:** Add integration test verifying proposal output cannot directly mutate memory

7. **HighPriorityFixes.txt Claims VSA Authority Is "Authoritative Source"** (LOW)
   - HighPriorityFixes.txt line 134 claims VSA is "authoritative source"
   - docs/onboarding.md line 135 confirms VSA tracked files required for build
   - But VSA files themselves are exported, not canonical memory
   - **Risk:** Docs conflation of "build requirement" with "memory authority"
   - **Mitigation:** Clarify in VSA docs that files are production-required but not canonical

8. **Simulation vs Live Boundary Not Tested at Build Time** (LOW)
   - Forbidden imports checked in `forbidden_live_imports_test.go`
   - But only runs in simulator package; no repo-wide forbidden import scan
   - **Risk:** Future production code accidentally imports forgek
   - **Mitigation:** Add CI step scanning all non-forgek packages for "import.*forgek"

9. **Operator Cockpit Authority Claims Vs G6/G8 Implementation** (LOW)
   - current_phase_status.md line 37 says operator cockpit "does not add action controls"
   - But docs/operations/forge_graphical_shell_session.md (line 13) references System page
   - Does System page actually enforce read-only at all layers?
   - **Risk:** UI claims action control enforcement; backend doesn't match
   - **Mitigation:** Test that System page requests cannot mutate or execute

10. **Full-Code-Review.md as Prompt Template Should Be Archived** (LOW)
    - File at repo root is a prompt specification, not a review output
    - Confusing for new readers who see it and think it's current review
    - **Risk:** Operator treats prompt template as current code review
    - **Mitigation:** Move to docs/prompt-packs/ and label as inert template

---

## Docs Requiring Supersession Label

| File | Current Lane | Supersession Label | Reason |
|---|---|---|---|
| docs/reviews/current_phase_status.md | Phase G8 section | `[REQUIRES-DESKTOP-SHELL-POWER-SUPERSESSION]` | "No host mutation" claim conflicts with policy-gated power actions |
| docs/status/dangerous_capabilities.md | Section 3 | `[REQUIRES-SHELL-POWER-ACTION-ENTRY]` | Missing shell.power_action in approval-only list |
| docs/DESKTOP_SHELL.md | Main body | `[REQUIRES-POWER-CONTROL-SECTION]` | Silent about FORGE_SHELL_DIRECT_SYSTEM_CONTROL env var |
| docs/operations/forge_graphical_shell_session.md | Line 43 | `[REQUIRES-POLICY-GATE-CLARIFICATION]` | "Host mutation disabled" oversimplifies policy-gated reality |
| docs/reports/phase_g8_desktop_shell_verification.md | Test plan | `[REQUIRES-POWER-CONTROL-VERIFICATION]` | Missing test for env var default state |

---

## Conclusion

**Pass 7 Authority Boundary Audit:** MOSTLY COMPLIANT with MEDIUM AUTHORITY RISK
- FORGE-K simulator properly isolated and labeled
- Memory writes are controllane-gated
- Tool execution is gateway-only
- Desktop shell has policy-gated host power controls that ARE NOT ENFORCED at binary level

**Pass 10 Docs Truth Alignment:** CONFLICTING / NEEDS-SUPERSESSION
- 4 documents claim "no host mutation" when policy-gated power controls exist
- dangerous_capabilities.md missing shell.power_action entry
- Full-Code-Review.md should be labeled as prompt template, not output
- Authority boundaries are sound; docs are drift-lagging implementation

**Top Priority:** Supersede desktop shell phase docs to reflect actual policy-gated power action posture and document env var in operator runbooks.
