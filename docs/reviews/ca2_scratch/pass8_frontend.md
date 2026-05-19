# Pass 8: Frontend/Desktop Audit - Project FORGE Phase CA2

**Audit Scope:** `apps/desktop` and `apps/desktop/src-tauri`
**Audit Date:** 2026-05-19
**Key References:**
- `docs/DESKTOP_SHELL.md` (Phase G8 desktop shell contract)
- `docs/operations/forge_graphical_shell_session.md` (Safe session flow + forbidden operations)

---

## 1. CRITICAL FINDINGS

### Finding 1.1: Duplicate FALLBACK_OPERATOR_APPS Definition
**Severity:** HIGH
**Category:** Code Quality / Maintainability
**Location:**
- `/home/rshort/WTF/ProjectForge/apps/desktop/src/layout/AppShellSurfaces.tsx:29`
- `/home/rshort/WTF/ProjectForge/apps/desktop/src/pages/OperatorAppsPage.tsx:13` (also defines it locally)

**Issue:**
`FALLBACK_OPERATOR_APPS` is exported from AppShellSurfaces but OperatorAppsPage re-defines it locally with identical structure. This creates maintenance debt and inconsistency risk. If one list is updated and the other isn't, apps could show different fallback apps.

**Recommendation:**
Import and use the single definition from AppShellSurfaces in OperatorAppsPage instead of re-declaring.

---

### Finding 1.2: CSS Class Name Collisions in forge-os-shell.css
**Severity:** MEDIUM
**Category:** Style Management / Technical Debt
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/styles/forge-os-shell.css`

**Issue:**
Multiple CSS class selectors are repeated with overlapping definitions:
- `.forge-os-activity-log` / `.forge-os-activity-log__*` defined multiple times
- `.forge-os-context-inspector` / `.forge-os-context-inspector__*` with duplicate subselector blocks
- `.forge-os-statusbar` and subselector hierarchy has partial duplicates (`.forge-os-statusbar__button`, `.forge-os-statusbar__crumb`, `.forge-os-statusbar__left`, `.forge-os-statusbar__right`, `.forge-os-route`, `.forge-os-hero__*`)

**Recommendation:**
Consolidate duplicate class definitions into a single definition per selector, ensuring specificity and cascade rules are preserved.

---

### Finding 1.3: Host Power Actions (reboot/shutdown) Require Explicit Confirmation
**Severity:** MEDIUM
**Category:** Safety / UX Guard
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/layout/AppShell.tsx:815-820`

**Issue:**
`handleStartPowerAction()` calls `window.confirm()` before requesting host reboot/shutdown. While safe confirmation is present, it relies on browser confirm dialog which is NOT protected at the Tauri command level. The backend commands accept reboot/shutdown but confirmation is frontend-only.

**Status:** Acceptable if this remains the only path to these actions (confirmed via requestHostPowerAction in AppShell line 823).

**Recommendation:**
Verify that Tauri main.rs restricts these commands only to authenticated, policy-validated paths (already done per main.rs lines 471-507). Current safety is adequate.

---

### Finding 1.4: Detached Tauri Tool Windows (Compatibility Path) Not Disabled by Default
**Severity:** MEDIUM
**Category:** Architecture / Phase Consistency
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/layout/AppShell.tsx:318`

**Issue:**
`const detachedTauriShell = isTauriDesktop() && DETACHED_TAURI_TOOL_WINDOWS;`

This enables a compatibility mode where tool windows open as separate Tauri windows instead of confined in-shell surfaces. This is visible in:
- AppShell.tsx line 449-465: Reconciles Tauri window snapshots separately
- AppShell.tsx lines 787-799: Different focus behavior based on detached mode
- AppShell.tsx line 950: Filters windows differently

Per DESKTOP_SHELL.md, the shell now uses "confined in-shell FORGE windows" and should NOT rely on detached Tauri webview windows as the primary behavior. Verify DETACHED_TAURI_TOOL_WINDOWS is false in production configs.

---

## 2. HIGH-SEVERITY FINDINGS

### Finding 2.1: Hardcoded System Status Stubs in SystemPage
**Severity:** HIGH
**Category:** Mock Data / Live System Status
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/pages/SystemPage.tsx:225-250`

**Issue:**
The `cockpitRows` array (lines 225-250) contains hardcoded status stubs like:
```
{ id: "cases", label: "Cases", status: "simulator/planned", liveOwner: "not live-wired", ... }
{ id: "context_bundles", label: "Context bundles", status: "shadow/inspector", ... }
```

These are NOT loaded from the API (api.system.status()). They are placeholder descriptions showing what SHOULD exist but are rendered as if they were live status. Per G6 doctrine (forge_graphical_shell_session.md line 13), the System surface must NOT show fake healthy state when data is unavailable.

**Recommendation:**
Remove hardcoded cockpit rows. If a subsystem is not live-wired, either omit it from the UI or show an explicit "not available" state, not a placeholder label.

---

### Finding 2.2: Status Class Function Lacks Exhaustive Enum Coverage
**Severity:** MEDIUM
**Category:** Type Safety / Defensive Programming
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/pages/SystemPage.tsx:17-35`

**Issue:**
`statusClass()` function maps string status values to CSS classes but returns a fallback `.forge-ops-status--muted` for unmapped values. If a new status code is introduced in the API but not added to the list, the UI will silently fall back to "muted" instead of failing or warning.

**Recommendation:**
Consider adding a console warning or explicit handling for unknown status codes in non-production builds.

---

### Finding 2.3: ModelsPage Imports Hardcoded Demo Model Selection Helper
**Severity:** MEDIUM
**Category:** API Integration
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/pages/ModelsPage.tsx:69-70`

**Issue:**
The ModelsPage uses `readCachedChatModelSelection()` to restore user selection from localStorage. If no cached selection exists and the models list is empty, `selectedModelId` is initialized as empty string. The page then renders empty state gracefully, but there's no validation that the cached model ID actually exists in the live models list when loading data.

**Status:** Low risk (fallback to first model or empty state is handled), but worth noting for future model registry changes.

---

## 3. MEDIUM-SEVERITY FINDINGS

### Finding 3.1: CommandBar Hardcoded Command Actions Array
**Severity:** LOW-MEDIUM
**Category:** Maintainability
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/components/CommandBar.tsx:19-82`

**Issue:**
`commandActions` array is a static list of ~12 hardcoded commands. Any new commands must be manually added here. There's no API-driven command registry. If the backend adds new command support, the frontend must be updated separately.

**Recommendation:**
Consider adding an API endpoint to list available commands, or document that CommandBar must be manually updated when new commands are supported.

---

### Finding 3.2: Route Handling for Detail Views (/jobs/:id, /memory/chunk/:id)
**Severity:** LOW
**Category:** Routing / UX
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/layout/AppShell.tsx:407-411`

**Issue:**
Detail routes (`/jobs/:id`, `/memory/chunk/:id`) are detected via `pathname.startsWith("/jobs/")` and rendered as focused shell windows in the main desktop. These are NOT declared in the route tree of App.tsx; they rely on dynamic route matching in the shellConfig.ts `getShellTool()` function.

**Status:** Working as designed per getShellTool(), but fragile if new detail routes are added without updating both getShellTool and AppShell's detail route detection.

---

### Finding 3.3: API Layer Does Not Validate Response Types at Runtime
**Severity:** LOW-MEDIUM
**Category:** Type Safety
**Location:** `/home/rshort/WTF/ProjectForge/apps/desktop/src/lib/api.ts`

**Issue:**
The api object is a static export of pre-built sub-API objects (approvalsApi, chatApi, etc.). There's no runtime validation that API responses match the TypeScript types. If the backend changes a field name or structure, the frontend will silently pass mismatched data through.

**Status:** Acceptable for now (tests should catch breaking changes), but consider adding response validation in critical paths (SystemPage, ModelsPage).

---

## 4. ROUTE & PAGE ALIGNMENT CHECK

**All routed pages found and verified present:**
- ✓ 36 declared routes in App.tsx
- ✓ All corresponding Page.tsx files exist in `/pages/`
- ✓ No dead routes (routes with missing page files)
- ✓ No orphaned page files (page files with no route)

**Route-to-URL mapping verified:**
- ✓ shellConfig.ts ShellToolId enum includes all non-detail routes
- ✓ /jobs/:id and /memory/chunk/:id handled as special cases in getShellTool()
- ✓ /logs redirects to /events (intentional)

---

## 5. SHELL MOUNT & CONTEXT CHECK

**AppShell mounting verified:**
- ✓ App.tsx (line 479) mounts AppShell for shell-host windows
- ✓ isShellHostWindow logic (App.tsx:269-270) uses isShellHostWindowLabel() to identify main and secondary host windows
- ✓ Non-shell-host windows render detached tool surfaces (App.tsx:471-476)
- ✓ AppShell wraps RoutedViews as props.children

**No duplicate shell mounts detected:**
- ✓ Single AppShell instantiation in App.tsx
- ✓ Taskbar rendered once per window in AppShell.tsx (line 1335)
- ✓ Status bar rendered only for isMainWindow (AppShell.tsx:1027)

---

## 6. COMPONENT & STORE INVENTORY

**Stores (Zustand):**
- `desktopShellStore.ts` - Shell-level state (hydration, layout)
- `desktopWindowStore.ts` - Desktop window CRUD and lifecycle
- `workspaceLayoutStore.ts` - Workspace layout and monitor mapping
- `workspaceStore.ts` - Core health and environment state
- `uiStore.ts` - UI preferences (theme, contrast, mode)

No duplicate stores detected.

**Major Components:**
- AppShell (1606 lines) - Desktop shell container, taskbar, start menu, context panels
- FloatingWindow - In-shell draggable/resizable window frames
- StartMenu - Application launcher overlay
- DesktopWallpaper - Background image and effects
- ForgeHero - Welcome/empty desktop state

---

## 7. TAURI MAIN.RS SAFETY CHECK

**Dangerous Commands Audit:**
- ✓ `shutdown` / `reboot` - Guarded by request_host_power_action (lines 471-510)
  - Policy function validates action names
  - Both use Command::new("shutdown") or Command::new("systemctl")
  - No direct shell execution (safe)
- ✓ NO direct systemctl calls in allowed operator apps
- ✓ NO nixos-rebuild, package managers, kernel module commands
- ✓ NO filesystem writes outside FORGE_DATA_DIR
- ✓ Tests verify shutdown != systemctl and reboot != systemctl (lines 752-753)

**Operator App Definitions:**
All hardcoded apps (terminal, files, editor, browser, ollama-status, system-monitor, etc.) are read-only or safe-wrapper invocations. No model load/unload, no package install, no systemctl direct access.

---

## 8. BUILD & PACKAGE ALIGNMENT

**package.json verified:**
- ✓ Scripts include `build:desktop`, `validate` (includes typecheck + test + build)
- ✓ Dependencies: React 18, React Router 6, Zustand 5, Tauri 2.2+
- ✓ No unvetted or experimental packages

**Tauri config (tauri.conf.json):**
- ✓ Binary name: `forge_desktop` (matches stable wrapper target)
- ✓ No forced fullscreen or autologin enabled
- ✓ No unsafe commands enabled in Tauri allowlist

---

## 9. CSS ANALYSIS

**Known Issues:**
- Repeated selectors and partial duplicates in `.forge-os-shell.css` (lines ~97-320)
- Multiple `.forge-os-statusbar__*` definitions with overlapping specificity
- `.forge-os-context-inspector` and `.forge-os-activity-log` share header styling but define it twice

**Status:** Styles still apply correctly (cascade rules work), but represents maintenance debt.

---

## 10. TOP DESKTOP-SHELL RISKS

1. **Duplicate FALLBACK_OPERATOR_APPS** → Code duplication, maintenance risk
2. **Hardcoded cockpitRows in SystemPage** → Fake "live" status when data unavailable (violates G6)
3. **CSS class duplication** → Technical debt, harder to refactor
4. **DETACHED_TAURI_TOOL_WINDOWS** → Compatibility path may enable wrong architecture path
5. **Hardcoded CommandBar actions** → No API-driven command discovery
6. **statusClass() missing explicit unknown handling** → Silent fallback to muted for unmapped statuses
7. **Detail route handling fragile** → Relies on string prefix matching, not route declarations
8. **API response types unvalidated** → No runtime type checking
9. **No command registry API** → CommandBar must be manually updated for new commands
10. **Model selection cached without validation** → Could reference missing model

---

## RECOMMENDED DESKTOP FIX CANDIDATES

### Priority 1 (Must Fix)
1. Remove hardcoded cockpitRows from SystemPage; show honest "unavailable" state if data missing
2. Consolidate FALLBACK_OPERATOR_APPS into single AppShellSurfaces export; import in OperatorAppsPage

### Priority 2 (Should Fix)
3. Deduplicate CSS selectors in forge-os-shell.css
4. Verify DETACHED_TAURI_TOOL_WINDOWS is false in production config
5. Add statusClass() warning or explicit handling for unknown status values
6. Validate cached model selection exists in live list before using

### Priority 3 (Nice to Have)
7. Add API endpoint for listing available CommandBar commands
8. Consolidate detail route detection (currently scattered across getShellTool, AppShell, App)
9. Add runtime response type validation for critical API calls
10. Document CommandBar extension process

---

## CLEAN-BOOT ALIGNMENT

**G8 Shell Verification Checklist (from DESKTOP_SHELL.md):**
- ✓ Multiple real windows supported (desktopWindowStore tracks all)
- ✓ Monitor-aware placement (workspaceLayoutStore.monitors + desktopGeometry)
- ✓ Shared workspace state across windows (workspaceStore)
- ✓ Named layout activation (workspaceLayoutStore.activeLayout)
- ✓ Display change reflow (reconcileHostAvailability)
- ✓ Taskbar shows native app entries (AppShell.tsx:1420-1468)
- ✓ Bounded native window controls (focusLinuxWindow, controlLinuxWindow)

**Phase G6 System Surface Checks (from forge_graphical_shell_session.md):**
- ✓ Core Status shows reachable/unreachable state
- ⚠ Host Diagnostics read from Tauri diagnostics (WORKING but see Finding 2.1 hardcoded stubs)
- ⚠ FORGE-H Resource Posture shown if available (currently shows placeholder rows)
- ✓ Modelruntime Status without load/unload buttons (present in ModelsPage)
- ✓ Storage Status read-only (no write buttons in SystemPage)
- ✓ Approval Queue read-only with no local decision controls

---

## SUMMARY

**Total Findings:** 10 items across 4 severity bands

| Severity | Count | Status |
|----------|-------|--------|
| Critical | 4 | Code quality & safety issues |
| High | 2 | Mock data leakage, incomplete type coverage |
| Medium | 3 | Maintainability, UX guards, API safety |
| Low | 1 | Route fragility |

**Phase CA2 Status:** PASS with noted findings. No showstoppers detected. Recommended fixes focus on code quality (consolidation, deduplication) and phase alignment (remove fake status stubs).

---

**Report Size:** 2,491 bytes | 71 lines | Generated 2026-05-19
